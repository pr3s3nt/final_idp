package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/configresolver"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/infraplanner"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/manifest"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/resourceoutput"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/targetadapter"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/waveplanner"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/cd"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/kubernetes"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/provisioner"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/secretstore"
	"github.com/pr3s3nt/final_idp/uc03/internal/persistence"
)

// Worker is the Deployment Worker: it claims execution jobs and executes them
// wave by wave. It owns no operation of its own; every step is delegated to
// the component that owns it.
type Worker struct {
	Orch            *Orchestrator
	Provisioner     provisioner.Provisioner
	ResourceOutputs *resourceoutput.Collector
	Kube            WorkloadStatusProvider
	CD              cd.Integration
	Renderer        *manifest.ScoreRenderer
	Secrets         secretstore.Store
	HashKey         []byte
	HealthTimeout   time.Duration
	SyncTimeout     time.Duration
	Logf            func(format string, args ...any)
}

func (w *Worker) logf(format string, args ...any) {
	if w.Logf != nil {
		w.Logf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

// Run polls for jobs until ctx is cancelled. One job runs at a time.
func (w *Worker) Run(ctx context.Context, poll time.Duration) error {
	w.logf("deployment worker started")
	for {
		processed, err := w.RunOnce(ctx)
		if err != nil {
			w.logf("worker: %v", err)
		}
		if processed {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(poll):
		}
	}
}

// RunOnce claims and executes at most one job.
func (w *Worker) RunOnce(ctx context.Context) (bool, error) {
	job, err := w.Orch.Deployments.ClaimNextExecutionJob(ctx)
	if err != nil || job == nil {
		return false, err
	}
	w.logf("deployment %s: claimed job %s", job.DeploymentID, job.JobID)
	ex := &execution{
		w: w, job: job, scope: map[string]bool{}, done: map[string]bool{},
		resourceOutputs: map[string]domain.Outputs{}, workloadOutputs: map[string]domain.Outputs{},
		instances: map[string]*domain.ResourceInstance{}, workloadInstanceIDs: map[string]string{},
		wds: map[string]*domain.WorkloadDeployment{}, touched: map[string]bool{},
		extra: map[string]*domain.PlanItem{}, reapplied: map[string]string{},
	}
	runErr := ex.run(ctx)

	bookkeeping := context.WithoutCancel(ctx)
	in := persistence.FinishInput{
		DeploymentID: job.DeploymentID, JobID: job.JobID, RecordID: job.RecordID, Succeeded: runErr == nil,
		DeliveryReference: ex.delivery, RemovedComponents: ex.removed, ResourceInstanceIDs: ex.touchedIDs(),
	}
	if runErr != nil {
		in.ErrorSummary = summarize(runErr)
		w.logf("deployment %s: FAILED: %s", job.DeploymentID, in.ErrorSummary)
	} else {
		w.logf("deployment %s: SUCCEEDED", job.DeploymentID)
	}
	if err := w.Orch.Deployments.FinishDeployment(bookkeeping, in); err != nil {
		return true, fmt.Errorf("finish deployment %s: %w", job.DeploymentID, err)
	}
	return true, nil
}

type execution struct {
	w   *Worker
	job *persistence.ClaimedJob
	pc  *planContext

	namespace           string
	scope, done         map[string]bool
	resourceOutputs     map[string]domain.Outputs // component ID -> outputs (transient)
	workloadOutputs     map[string]domain.Outputs
	instances           map[string]*domain.ResourceInstance // requirement ID -> instance
	workloadInstanceIDs map[string]string
	wds                 map[string]*domain.WorkloadDeployment
	cluster             *kubernetes.ClusterAccess
	touched             map[string]bool
	extra               map[string]*domain.PlanItem // resources outside the plan applied again
	reapplied           map[string]string           // resource ID -> names of changed components it requires
	delivery            string
	removed             []string
	nextWave            int
}

func (ex *execution) touchedIDs() []string {
	ids := make([]string, 0, len(ex.touched))
	for id := range ex.touched {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (ex *execution) repo() *persistence.DeploymentRepository { return ex.w.Orch.Deployments }

func (ex *execution) startStep(ctx context.Context, wave int, name domain.StepName, component string) string {
	id, err := ex.repo().StartStep(context.WithoutCancel(ctx), ex.job.DeploymentID, ex.job.RecordID, wave, name, component)
	if err != nil {
		ex.w.logf("record step %s: %v", name, err)
	}
	return id
}

func (ex *execution) finishStep(ctx context.Context, id string, status domain.StepStatus, errSummary string, detail map[string]any) {
	if id == "" {
		return
	}
	if err := ex.repo().FinishStep(context.WithoutCancel(ctx), id, status, errSummary, detail); err != nil {
		ex.w.logf("finish step: %v", err)
	}
}

func (ex *execution) fail(ctx context.Context, stepIDs []string, wave int, component string, step domain.StepName, err error) error {
	for _, id := range stepIDs {
		ex.finishStep(ctx, id, domain.StepFailed, summarize(err), nil)
	}
	return &domain.ExecutionError{Wave: wave, Component: component, Step: step, Err: err}
}

func (ex *execution) run(ctx context.Context) error {
	w := ex.w
	step := ex.startStep(ctx, 0, domain.StepPlanVerified, "")
	d, err := w.Orch.Deployments.FindByIDWithPersistedInputs(ctx, ex.job.DeploymentID)
	if err != nil {
		return ex.fail(ctx, []string{step}, 0, "", domain.StepPlanVerified, err)
	}
	pc, err := w.Orch.rebuildFor(ctx, d)
	if err != nil {
		return ex.fail(ctx, []string{step}, 0, "", domain.StepPlanVerified, fmt.Errorf("the plan can no longer be built: %w", err))
	}
	if fp := pc.Plan.Fingerprint(); fp != d.PlanFingerprint {
		return ex.fail(ctx, []string{step}, 0, "", domain.StepPlanVerified,
			fmt.Errorf("%s: inputs changed after confirmation (plan fingerprint %s, confirmed %s)", domain.CodePlanChangedBeforeExecute, fp[:12], d.PlanFingerprint[:12]))
	}
	if err := infraplanner.ApplyInfrastructureOverrides(pc.Plan, ex.job.SelectedOverrides); err != nil {
		return ex.fail(ctx, []string{step}, 0, "", domain.StepPlanVerified, err)
	}
	ex.finishStep(ctx, step, domain.StepSucceeded, "", map[string]any{"fingerprint": d.PlanFingerprint, "kind": string(d.Kind)})

	ex.pc = pc
	ex.namespace = Namespace(pc.Version.ApplicationName, d.Environment)
	for i := range pc.Instances {
		ex.instances[pc.Instances[i].RequirementID] = &pc.Instances[i]
	}
	for i := range d.WorkloadDeployments {
		ex.wds[d.WorkloadDeployments[i].WorkloadID] = &d.WorkloadDeployments[i]
	}

	if d.Kind == domain.KindDeploy {
		if err := ex.runWaves(ctx); err != nil {
			return err
		}
		if !pc.Waves.FullDeployment {
			return nil
		}
	}
	return ex.runRemovals(ctx)
}

// Namespace is the Kubernetes namespace of an application + environment.
func Namespace(application string, env domain.Environment) string {
	ns := strings.ToLower(application + "-" + string(env))
	ns = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(ns, "-")
	if len(ns) > 63 {
		ns = ns[:63]
	}
	return strings.Trim(ns, "-")
}

// runWaves executes waves in dependency order. After each wave it compares
// output fingerprints and, when outputs changed, applies managed resources that
// require the changed components again, adds dependent workloads (with their
// running image) and re-layers the waves not yet executed.
func (ex *execution) runWaves(ctx context.Context) error {
	g := ex.pc.Graph
	for id := range ex.pc.Waves.Scope {
		ex.scope[id] = true
	}
	waves := append([][]string{}, ex.pc.Waves.Waves...)
	for wi := 0; wi < len(waves); wi++ {
		var resources, workloads []string
		for _, id := range waves[wi] {
			if g.Nodes[id].Kind == domain.NodeResource {
				resources = append(resources, id)
			} else {
				workloads = append(workloads, id)
			}
		}
		for _, id := range resources {
			if err := ex.reconcileResource(ctx, wi, id); err != nil {
				return err
			}
		}
		if len(workloads) > 0 {
			if err := ex.deployWorkloads(ctx, wi, workloads); err != nil {
				return err
			}
		}
		changed, err := ex.propagateOutputChanges(ctx, waves[wi])
		if err != nil {
			return &domain.ExecutionError{Wave: wi, Step: domain.StepApplicationReady, Err: err}
		}
		for _, id := range waves[wi] {
			ex.done[id] = true
		}
		for _, id := range waveplanner.ResourcesToReapply(g, changed, ex.scope, ex.done, ex.pc.Plan) {
			if it := ex.pc.Plan.Item(id); it != nil {
				it.Action = domain.ActionUpdate
			} else {
				ri := ex.instances[id]
				if ri == nil || ri.Status != domain.RIReady {
					continue
				}
				it, err := infraplanner.ReapplyItem(g, id, ri, ex.pc.Resolver)
				if err != nil {
					return &domain.ExecutionError{Wave: wi, Component: g.Nodes[id].Name, Step: domain.StepInfrastructureReady, Err: err}
				}
				ex.extra[id] = it
			}
			var causes []string
			for _, dep := range g.Nodes[id].DependsOn {
				if contains(changed, dep) {
					causes = append(causes, g.Nodes[dep].Name)
				}
			}
			ex.scope[id], ex.reapplied[id] = true, strings.Join(causes, ", ")
			ex.w.logf("deployment %s: %s is applied again (outputs of %s changed)", ex.job.DeploymentID, g.Nodes[id].Name, ex.reapplied[id])
		}
		for _, c := range waveplanner.PropagateOutputChanges(g, ex.scope, changed, ex.pc.Running) {
			wd := &domain.WorkloadDeployment{WorkloadID: c.WorkloadID, ImageRepository: c.Running.ImageRepository, ImageVersion: c.Running.ImageVersion, WaveNumber: wi + 1}
			if err := ex.repo().InsertCascadedWorkloadDeployment(ctx, ex.job.DeploymentID, wd); err != nil {
				return err
			}
			ex.wds[c.WorkloadID], ex.scope[c.WorkloadID] = wd, true
			ex.w.logf("deployment %s: %s added (CASCADED, image %s)", ex.job.DeploymentID, g.Nodes[c.WorkloadID].Name, wd.ImageRef())
		}
		remaining := map[string]bool{}
		for id := range ex.scope {
			if !ex.done[id] {
				remaining[id] = true
			}
		}
		layers := waveplanner.Layer(g, remaining)
		waves = append(waves[:wi+1], layers...)
		renumber := map[string]int{}
		for li, layer := range layers {
			for _, id := range layer {
				if g.Nodes[id].Kind == domain.NodeWorkload {
					renumber[id] = wi + 1 + li
					ex.wds[id].WaveNumber = wi + 1 + li
				}
			}
		}
		if err := ex.repo().UpdateWaveNumbers(ctx, ex.job.DeploymentID, renumber); err != nil {
			return err
		}
	}
	ex.nextWave = len(waves)
	return nil
}

// item returns the plan item of a resource, including resources outside the
// plan that are applied again during execution.
func (ex *execution) item(componentID string) *domain.PlanItem {
	if it := ex.pc.Plan.Item(componentID); it != nil {
		return it
	}
	return ex.extra[componentID]
}

func (ex *execution) definition(componentID string) *domain.ResourceDefinition {
	if res, ok := ex.pc.Graph.Resolutions[componentID]; ok {
		return res.Definition
	}
	if ri := ex.instances[componentID]; ri != nil {
		return ex.pc.Resolver.DefinitionByID(ri.DefinitionID)
	}
	return nil
}

// reconcileInfrastructure for one resource item of a wave.
func (ex *execution) reconcileResource(ctx context.Context, wave int, id string) error {
	item := ex.item(id)
	def := ex.definition(id)
	step := ex.startStep(ctx, wave, domain.StepInfrastructureReady, item.Name)
	ex.w.logf("deployment %s: wave %d %s %s (%s)", ex.job.DeploymentID, wave, item.Action, item.Name, def.Name)
	ri, err := ex.applyResource(ctx, item, def)
	if err != nil {
		return ex.fail(ctx, []string{step}, wave, item.Name, domain.StepInfrastructureReady, err)
	}
	ex.touched[ri.ID] = true
	if _, err := ex.outputsOf(ctx, id); err != nil {
		return ex.fail(ctx, []string{step}, wave, item.Name, domain.StepInfrastructureReady, err)
	}
	detail := map[string]any{"action": item.Action, "definition": def.Name, "managementMode": string(def.ManagementMode), "resourceInstanceId": ri.ID}
	if cause, ok := ex.reapplied[id]; ok {
		detail["appliedAgainBecauseOutputsChanged"] = cause
	}
	ex.finishStep(ctx, step, domain.StepSucceeded, "", detail)
	return nil
}

func (ex *execution) applyResource(ctx context.Context, item *domain.PlanItem, def *domain.ResourceDefinition) (*domain.ResourceInstance, error) {
	repo := ex.w.Orch.Resources
	d := ex.pc.Deployment
	platformType := ""
	if item.Platform {
		platformType = def.ResourceType
	}
	switch item.Action {
	case domain.ActionReuse:
		ri := ex.instances[item.ComponentID]
		if ri == nil || ri.Status != domain.RIReady {
			return nil, fmt.Errorf("instance to reuse is not READY")
		}
		// A deployment with another catalog version reuses the instance under
		// the same-named definition of that version.
		ref := ri.InfrastructureReference
		if def.ManagementMode == domain.Existing {
			ref = def.ExistingResourceReference
		}
		if ri.DefinitionID != def.ID || ri.InfrastructureReference != ref {
			if err := repo.UpdateDefinition(ctx, ri.ID, def.ID, ref); err != nil {
				return nil, err
			}
			ri.DefinitionID, ri.InfrastructureReference = def.ID, ref
		}
		return ri, nil
	case domain.ActionLink:
		ri := &domain.ResourceInstance{ApplicationID: d.ApplicationID, Environment: d.Environment, RequirementID: item.ComponentID,
			DefinitionID: def.ID, Target: d.Target, InfrastructureReference: def.ExistingResourceReference, Status: domain.RIReady,
			AppliedOverrides: map[string]any{}}
		if err := repo.Create(ctx, ri, platformType); err != nil {
			return nil, err
		}
		ex.instances[item.ComponentID] = ri
		return ri, nil
	case domain.ActionCreate:
		id := uuid.NewString()
		ri := &domain.ResourceInstance{ID: id, ApplicationID: d.ApplicationID, Environment: d.Environment, RequirementID: item.ComponentID,
			DefinitionID: def.ID, Target: d.Target, InfrastructureReference: "terraform-workspace://" + id, Status: domain.RIProvisioning,
			AppliedOverrides: map[string]any{}}
		if err := repo.Create(ctx, ri, platformType); err != nil {
			return nil, err
		}
		ex.instances[item.ComponentID] = ri
		return ex.provision(ctx, ri, item, def)
	case domain.ActionUpdate:
		ri := ex.instances[item.ComponentID]
		if ri == nil {
			return nil, fmt.Errorf("instance to update not found")
		}
		if err := repo.UpdateStatus(ctx, ri.ID, domain.RIProvisioning); err != nil {
			return nil, err
		}
		ri.Status = domain.RIProvisioning
		return ex.provision(ctx, ri, item, def)
	}
	return nil, fmt.Errorf("unexpected action %s", item.Action)
}

func (ex *execution) provision(ctx context.Context, ri *domain.ResourceInstance, item *domain.PlanItem, def *domain.ResourceDefinition) (*domain.ResourceInstance, error) {
	if def.ManagementMode != domain.Managed {
		return nil, fmt.Errorf("refusing to provision %s: definition %s is %s", item.Name, def.Name, def.ManagementMode)
	}
	vars, err := ex.variables(ctx, item, def, ri)
	if err != nil {
		return nil, err
	}
	res, err := ex.w.Provisioner.Reconcile(ctx, provisioner.Request{InstanceID: ri.ID, ProvisionerReference: def.ProvisionerReference, Variables: vars})
	repo := ex.w.Orch.Resources
	if err != nil {
		ri.Status = domain.RIFailed
		if saveErr := repo.SaveState(context.WithoutCancel(ctx), ri); saveErr != nil {
			ex.w.logf("save failed resource state: %v", saveErr)
		}
		return nil, err
	}
	ri.Status, ri.InfrastructureReference, ri.ProviderStateReference = domain.RIReady, res.InfrastructureReference, res.ProviderStateReference
	ri.DefinitionID = def.ID
	ri.AppliedOverrides = item.NewBaseline
	ri.AppliedInputFingerprint = infraplanner.InputFingerprint(def, item.FinalParameters,
		infraplanner.RequiredOutputFingerprints(ex.pc.Graph, item.ComponentID, func(dep string) string { return ex.resourceOutputs[dep].Fingerprint() }))
	delete(ex.resourceOutputs, item.ComponentID)
	return ri, repo.SaveState(ctx, ri)
}

// variables are the module inputs: final parameters, naming and tags, and the
// outputs of every resource the definition requires.
func (ex *execution) variables(ctx context.Context, item *domain.PlanItem, def *domain.ResourceDefinition, ri *domain.ResourceInstance) (map[string]any, error) {
	d := ex.pc.Deployment
	vars := infraplanner.Merge(item.FinalParameters)
	vars["name"] = resourceName(ex.pc.Version.ApplicationName, d.Environment, item.Name)
	vars["tags"] = map[string]string{
		"idp-app": ex.pc.Version.ApplicationName, "idp-env": strings.ToLower(string(d.Environment)), "idp-target": d.Target,
		"idp-resource-instance": ri.ID, "idp-uc03-run": "idp-uc03",
	}
	if d.Context.Region != "" {
		vars["region"] = d.Context.Region
	}
	for _, dep := range item.DependsOn {
		node := ex.pc.Graph.Nodes[dep]
		if node == nil || node.Kind != domain.NodeResource {
			continue
		}
		outs, err := ex.outputsOf(ctx, dep)
		if err != nil {
			return nil, fmt.Errorf("outputs of required %s: %w", node.Name, err)
		}
		values := map[string]string{}
		for k, o := range outs {
			values[k] = o.Value
		}
		vars[provisioner.Snake(node.ResourceType)] = values
	}
	return vars, nil
}

func resourceName(app string, env domain.Environment, name string) string {
	n := strings.ToLower(fmt.Sprintf("idp-%s-%s-%s", app, env, name))
	n = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(n, "-")
	if len(n) > 38 {
		n = n[:38]
	}
	return strings.Trim(n, "-")
}

// outputsOf collects Resource Outputs of a READY instance once per execution.
func (ex *execution) outputsOf(ctx context.Context, componentID string) (domain.Outputs, error) {
	if outs, ok := ex.resourceOutputs[componentID]; ok {
		return outs, nil
	}
	ri := ex.instances[componentID]
	def := ex.definition(componentID)
	if ri == nil || def == nil {
		return nil, fmt.Errorf("no resource instance for component %s", componentID)
	}
	outs, err := ex.w.ResourceOutputs.CollectResourceOutputs(ctx, ri, def)
	if err != nil {
		return nil, err
	}
	ex.resourceOutputs[componentID] = outs
	if def.ResourceType == domain.ResourceTypeCluster {
		if ex.cluster, err = kubernetes.AccessFromOutputs(outs); err != nil {
			return nil, err
		}
	}
	return outs, nil
}

func (ex *execution) clusterAccess(ctx context.Context) (*kubernetes.ClusterAccess, error) {
	if ex.cluster != nil {
		return ex.cluster, nil
	}
	id := domain.PlatformRequirementID(ex.pc.Deployment.ApplicationID, domain.ResourceTypeCluster)
	if _, err := ex.outputsOf(ctx, id); err != nil {
		return nil, fmt.Errorf("k8s-cluster is not available: %w", err)
	}
	return ex.cluster, nil
}

// workloadOutputsOf returns Workload Outputs collected in this execution or
// read from the running instance of a dependency outside the scope.
func (ex *execution) workloadOutputsOf(ctx context.Context, workloadID string) (domain.Outputs, error) {
	if outs, ok := ex.workloadOutputs[workloadID]; ok {
		return outs, nil
	}
	w := ex.pc.Version.Workload(workloadID)
	if w == nil {
		return nil, fmt.Errorf("workload %s is not in the deployed version", workloadID)
	}
	cluster, err := ex.clusterAccess(ctx)
	if err != nil {
		return nil, err
	}
	outs, err := ex.w.Kube.ReadWorkloadOutputs(ctx, cluster, ex.namespace, workloadID, w.ExposedOutputs)
	if err != nil {
		return nil, err
	}
	ex.workloadOutputs[workloadID] = outs
	return outs, nil
}

func (ex *execution) desiredState(wave int) cd.DesiredState {
	d := ex.pc.Deployment
	return cd.DesiredState{Cluster: ex.cluster, Target: d.Target, ApplicationID: d.ApplicationID,
		Application: ex.pc.Version.ApplicationName, Environment: string(d.Environment),
		Namespace: ex.namespace, DeploymentID: d.ID, Wave: wave}
}

// deployWorkloads resolves configuration, renders and publishes one wave of
// workloads, waits for the delivery and for the new rollout to be healthy,
// then collects Workload Outputs.
func (ex *execution) deployWorkloads(ctx context.Context, wave int, ids []string) error {
	w := ex.w
	cluster, err := ex.clusterAccess(ctx)
	if err != nil {
		return &domain.ExecutionError{Wave: wave, Step: domain.StepConfigurationResolved, Err: err}
	}
	mirror := targetadapter.RegistryMirror(ex.pc.Graph)
	var states []cd.WorkloadState
	var refs []kubernetes.WorkloadRef
	var names []string
	for _, id := range ids {
		wl := ex.pc.Version.Workload(id)
		wd := ex.wds[id]
		names = append(names, wl.Name)

		step := ex.startStep(ctx, wave, domain.StepConfigurationResolved, wl.Name)
		for _, dep := range ex.pc.Graph.Nodes[id].DependsOn {
			var err error
			if ex.pc.Graph.Nodes[dep].Kind == domain.NodeResource {
				_, err = ex.outputsOf(ctx, dep)
			} else {
				_, err = ex.workloadOutputsOf(ctx, dep)
			}
			if err != nil {
				return ex.fail(ctx, []string{step}, wave, wl.Name, domain.StepConfigurationResolved, err)
			}
		}
		rc, err := configresolver.ResolveEnvironmentConfiguration(ctx, ex.pc.Configuration, wl, ex.resourceOutputs, ex.workloadOutputs, w.Secrets)
		if err != nil {
			return ex.fail(ctx, []string{step}, wave, wl.Name, domain.StepConfigurationResolved, err)
		}
		ex.finishStep(ctx, step, domain.StepSucceeded, "", map[string]any{"variables": sortedNames(rc.Variables), "secrets": sortedNames(rc.Secrets)})

		step = ex.startStep(ctx, wave, domain.StepManifestGenerated, wl.Name)
		target := manifest.TargetInfo{Application: ex.pc.Version.ApplicationName, Environment: string(ex.pc.Deployment.Environment), Namespace: ex.namespace, RegistryMirror: mirror}
		body, secrets, err := ex.render(ctx, wl, *wd, rc, target)
		if err != nil {
			return ex.fail(ctx, []string{step}, wave, wl.Name, domain.StepManifestGenerated, err)
		}
		image := targetadapter.ResolveImage(wd.ImageRepository, wd.ImageVersion, mirror)
		ex.finishStep(ctx, step, domain.StepSucceeded, "", map[string]any{"image": image, "inclusionReason": wd.InclusionReason})
		states = append(states, cd.WorkloadState{WorkloadID: id, Manifests: body, Secrets: secrets})
		refs = append(refs, kubernetes.WorkloadRef{WorkloadID: id, Name: wl.Name, Namespace: ex.namespace, Image: image})
	}

	component := strings.Join(names, ", ")
	var synced []string
	for _, n := range names {
		synced = append(synced, ex.startStep(ctx, wave, domain.StepCDSynced, n))
	}
	ds := ex.desiredState(wave)
	ds.Cluster = cluster
	ds.Upsert = states
	ref, err := w.CD.PublishDesiredDeploymentState(ctx, ds)
	if err != nil {
		return ex.fail(ctx, synced, wave, component, domain.StepCDSynced, err)
	}
	ex.delivery = ref
	for _, id := range ids {
		wiID, err := w.Orch.Workloads.SaveDeploying(ctx, ex.pc.Deployment.ApplicationID, id, ex.pc.Deployment.Environment, ex.pc.Deployment.Target, ex.wds[id].ID)
		if err != nil {
			return ex.fail(ctx, synced, wave, component, domain.StepCDSynced, err)
		}
		ex.workloadInstanceIDs[id] = wiID
	}
	if err := w.CD.WaitForDelivery(ctx, ds, ref, w.SyncTimeout); err != nil {
		ex.markWorkloads(ctx, ids, domain.WIFailed)
		return ex.fail(ctx, synced, wave, component, domain.StepCDSynced, err)
	}
	for _, id := range synced {
		ex.finishStep(ctx, id, domain.StepSucceeded, "", map[string]any{"deliveryReference": ref})
	}

	var ready []string
	for _, n := range names {
		ready = append(ready, ex.startStep(ctx, wave, domain.StepApplicationReady, n))
	}
	if err := w.Kube.WaitForWorkloadsHealthy(ctx, cluster, refs, w.HealthTimeout); err != nil {
		ex.markWorkloads(ctx, ids, domain.WIFailed)
		return ex.fail(ctx, ready, wave, component, domain.StepApplicationReady, err)
	}
	for i, id := range ids {
		wl := ex.pc.Version.Workload(id)
		outs, err := w.Kube.ReadWorkloadOutputs(ctx, cluster, ex.namespace, id, wl.ExposedOutputs)
		if err != nil {
			return ex.fail(ctx, ready[i:], wave, wl.Name, domain.StepApplicationReady, err)
		}
		ex.workloadOutputs[id] = outs
		detail := map[string]any{"image": refs[i].Image}
		if o, ok := outs["endpoint"]; ok {
			detail["endpoint"] = o.Value
		}
		ex.finishStep(ctx, ready[i], domain.StepSucceeded, "", detail)
	}
	return nil
}

func (ex *execution) markWorkloads(ctx context.Context, ids []string, status domain.WorkloadInstanceStatus) {
	for _, id := range ids {
		if wiID := ex.workloadInstanceIDs[id]; wiID != "" {
			if err := ex.w.Orch.Workloads.UpdateStatus(context.WithoutCancel(ctx), wiID, status, nil); err != nil {
				ex.w.logf("mark workload %s: %v", id, err)
			}
		}
	}
}

func (ex *execution) render(ctx context.Context, wl *domain.Workload, wd domain.WorkloadDeployment, rc *manifest.ResolvedConfiguration, t manifest.TargetInfo) ([]byte, []cd.SecretObject, error) {
	spec, err := manifest.GenerateResolvedApplicationSpecification(wl, wd, rc)
	if err != nil {
		return nil, nil, err
	}
	objs, err := ex.w.Renderer.GenerateKubernetesManifest(ctx, spec, t.Namespace)
	if err != nil {
		return nil, nil, err
	}
	if objs, err = manifest.AdaptManifestForTarget(objs, wl, wd, t); err != nil {
		return nil, nil, err
	}
	if objs, err = manifest.MaterializeEnvironmentConfiguration(objs, wl, wd, t, ex.w.HashKey); err != nil {
		return nil, nil, err
	}
	objs, secrets, err := manifest.MaterializeSecretConfiguration(objs, wl, wd, rc, ex.w.HashKey)
	if err != nil {
		return nil, nil, err
	}
	body, err := manifest.Encode(objs)
	return body, secrets, err
}

// propagateOutputChanges compares the output fingerprints of a finished wave
// with the previous ones, stores the new fingerprints and marks workloads
// HEALTHY. It returns the components whose outputs changed, including the
// cluster and network: what runs on or requires them is redone.
func (ex *execution) propagateOutputChanges(ctx context.Context, ids []string) ([]string, error) {
	var changed []string
	previous := map[string]string{}
	for _, wi := range ex.pc.Running {
		previous[wi.WorkloadID] = wi.OutputFingerprint
	}
	for _, id := range ids {
		node := ex.pc.Graph.Nodes[id]
		switch node.Kind {
		case domain.NodeResource:
			ri := ex.instances[id]
			fp := ex.resourceOutputs[id].Fingerprint()
			if ri.OutputFingerprint != fp {
				changed = append(changed, id)
			}
			if err := ex.w.Orch.Resources.UpdateOutputFingerprint(ctx, ri.ID, fp); err != nil {
				return nil, err
			}
			ri.OutputFingerprint = fp
		case domain.NodeWorkload:
			fp := ex.workloadOutputs[id].Fingerprint()
			if previous[id] != fp {
				changed = append(changed, id)
			}
			if err := ex.w.Orch.Workloads.UpdateStatus(ctx, ex.workloadInstanceIDs[id], domain.WIHealthy, &fp); err != nil {
				return nil, err
			}
		}
	}
	return changed, nil
}

// runRemovals removes workloads no longer in the deployed version (or all of
// them for a teardown), verifies they are gone, then destroys managed resources
// and unlinks shared ones in dependency order.
func (ex *execution) runRemovals(ctx context.Context) error {
	w := ex.w
	d := ex.pc.Deployment
	for ri, removal := range ex.pc.Plan.Removals {
		wave := ex.nextWave + ri
		var workloadIDs, names []string
		for _, it := range removal.Items {
			if it.Kind == domain.NodeWorkload {
				workloadIDs, names = append(workloadIDs, it.ComponentID), append(names, it.Name)
			}
		}
		if len(workloadIDs) > 0 {
			var steps []string
			for _, n := range names {
				steps = append(steps, ex.startStep(ctx, wave, domain.StepRemoved, n))
			}
			component := strings.Join(names, ", ")
			cluster, err := ex.clusterAccess(ctx)
			if err != nil && d.Kind == domain.KindDeploy {
				return ex.fail(ctx, steps, wave, component, domain.StepRemoved, err)
			}
			if cluster != nil {
				ds := ex.desiredState(wave)
				ds.Cluster, ds.Remove = cluster, workloadIDs
				ref, err := w.CD.PublishDesiredDeploymentState(ctx, ds)
				if err != nil {
					return ex.fail(ctx, steps, wave, component, domain.StepRemoved, err)
				}
				ex.delivery = ref
				if err := w.CD.WaitForDelivery(ctx, ds, ref, w.SyncTimeout); err != nil {
					return ex.fail(ctx, steps, wave, component, domain.StepRemoved, err)
				}
				if err := w.Kube.WaitForWorkloadsRemoved(ctx, cluster, ex.namespace, workloadIDs, w.HealthTimeout); err != nil {
					return ex.fail(ctx, steps, wave, component, domain.StepRemoved, err)
				}
				for _, id := range workloadIDs {
					if err := w.Kube.DeleteSecretsByLabel(ctx, cluster, ex.namespace, "idp.dev/workload-id="+id); err != nil {
						return ex.fail(ctx, steps, wave, component, domain.StepRemoved, err)
					}
				}
				if d.Kind == domain.KindTeardown {
					if err := w.CD.RemoveApplication(ctx, ds); err != nil {
						return ex.fail(ctx, steps, wave, component, domain.StepRemoved, err)
					}
					if err := w.Kube.DeleteNamespace(ctx, cluster, ex.namespace); err != nil {
						return ex.fail(ctx, steps, wave, component, domain.StepRemoved, err)
					}
				}
			}
			for _, wi := range ex.pc.Running {
				if contains(workloadIDs, wi.WorkloadID) {
					if err := w.Orch.Workloads.UpdateStatus(ctx, wi.ID, domain.WIRemoved, nil); err != nil {
						return ex.fail(ctx, steps, wave, component, domain.StepRemoved, err)
					}
				}
			}
			for _, s := range steps {
				ex.finishStep(ctx, s, domain.StepSucceeded, "", map[string]any{"deliveryReference": ex.delivery})
			}
			ex.removed = append(ex.removed, workloadIDs...)
		}

		for _, it := range removal.Items {
			if it.Kind != domain.NodeResource {
				continue
			}
			instance := ex.instances[it.ComponentID]
			if instance == nil {
				continue
			}
			def := w.Orch.resolverDefinition(ex.pc, instance.DefinitionID)
			stepName, status := domain.StepDestroyed, domain.RIDestroyed
			if it.Action == domain.ActionUnlink {
				stepName, status = domain.StepUnlinked, domain.RIUnlinked
			}
			step := ex.startStep(ctx, wave, stepName, it.Name)
			if it.Action == domain.ActionDestroy {
				if def == nil || def.ManagementMode != domain.Managed {
					return ex.fail(ctx, []string{step}, wave, it.Name, stepName, errors.New("refusing to destroy a resource whose definition is not MANAGED"))
				}
				if err := w.Provisioner.Destroy(ctx, provisioner.Request{InstanceID: instance.ID, ProvisionerReference: def.ProvisionerReference}); err != nil {
					return ex.fail(ctx, []string{step}, wave, it.Name, stepName, err)
				}
			}
			if err := w.Orch.Resources.UpdateStatus(ctx, instance.ID, status); err != nil {
				return ex.fail(ctx, []string{step}, wave, it.Name, stepName, err)
			}
			ex.touched[instance.ID] = true
			ex.removed = append(ex.removed, it.ComponentID)
			if def != nil && def.ResourceType == domain.ResourceTypeCluster {
				ex.cluster = nil
			}
			ex.finishStep(ctx, step, domain.StepSucceeded, "", map[string]any{"action": it.Action, "resourceInstanceId": instance.ID})
		}
	}
	return nil
}

func (o *Orchestrator) resolverDefinition(pc *planContext, id string) *domain.ResourceDefinition {
	return pc.Resolver.DefinitionByID(id)
}

func sortedNames(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// summarize turns an execution error into an error summary for the record.
func summarize(err error) string {
	s := err.Error()
	if len(s) > 2000 {
		s = s[:2000] + "…"
	}
	return s
}
