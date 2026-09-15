// Package service holds the application services of UC-03: the Deployment
// Orchestrator (request side), the Deployment Worker (background execution)
// and the Deployment Query Service (result tracking).
package service

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/graphbuilder"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/infraplanner"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/resourceresolver"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/targetadapter"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/waveplanner"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/secretstore"
	"github.com/pr3s3nt/final_idp/uc03/internal/persistence"
)

// ImageChecker verifies that an image tag exists in its registry.
type ImageChecker interface {
	ImageExists(ctx context.Context, imageRef string) error
}

// Repositories groups the persistence collaborators shared by the services.
type Repositories struct {
	Apps        *persistence.ApplicationRepository
	Configs     *persistence.EnvironmentConfigurationRepository
	Catalog     *persistence.ResourceDefinitionCatalog
	Resources   *persistence.ResourceInstanceRepository
	Workloads   *persistence.WorkloadInstanceRepository
	Deployments *persistence.DeploymentRepository
}

// Orchestrator is the Deployment Orchestrator. It validates, builds the graph
// and plan, persists the deployment and, on confirmation, creates the execution
// job and returns. It never executes the deployment itself.
type Orchestrator struct {
	Repositories
	Images  ImageChecker
	Secrets secretstore.Store
}

type CreateRequest struct {
	Application         string            `json:"application"`
	Version             string            `json:"version"`
	Environment         string            `json:"environment"`
	Target              string            `json:"target"`
	CloudProvider       string            `json:"cloudProvider,omitempty"`
	Region              string            `json:"region,omitempty"`
	Workloads           []string          `json:"workloads"` // names or IDs of selected workloads
	Images              map[string]string `json:"images"`    // workload name or ID -> image tag
	TargetSpecificInput map[string]string `json:"targetSpecificInput,omitempty"`
}

// PlanView is what the Developer reviews before confirming.
type PlanView struct {
	Deployment      *domain.Deployment `json:"deployment"`
	Plan            *domain.Plan       `json:"plan"`
	Fingerprint     string             `json:"fingerprint"`
	FingerprintAlgo string             `json:"fingerprintAlgo"`
	// Stale is set when the plan rebuilt for display differs from the stored
	// fingerprint (inputs changed since the plan was created).
	Stale bool `json:"stale"`
}

// planContext carries everything a rebuilt plan was derived from, so the
// worker can execute against exactly those inputs.
type planContext struct {
	Deployment    *domain.Deployment
	Version       *domain.ApplicationVersion
	Configuration *domain.EnvironmentConfiguration
	Resolver      *resourceresolver.Resolver
	Graph         *domain.DeploymentGraph
	Waves         *domain.WavePlan
	Running       []domain.WorkloadInstance
	Instances     []domain.ResourceInstance
	Plan          *domain.Plan
}

func (o *Orchestrator) resolver(ctx context.Context) (*resourceresolver.Resolver, error) {
	defs, err := o.Catalog.List(ctx)
	if err != nil {
		return nil, err
	}
	return &resourceresolver.Resolver{Definitions: defs}, nil
}

// resolveContext completes and validates the deployment context against the
// targets the catalog can build a cluster for (A1-11).
func resolveContext(r *resourceresolver.Resolver, target, cloud, region string, input map[string]string) (domain.DeploymentContext, error) {
	var matches []resourceresolver.TargetOption
	for _, t := range r.SupportedTargets() {
		if t.Target == target && (cloud == "" || cloud == t.CloudProvider) && (region == "" || region == t.Region) {
			matches = append(matches, t)
		}
	}
	if len(matches) == 0 {
		return domain.DeploymentContext{}, domain.Reject(domain.CodeUnsupportedTarget,
			"target %q with cloud provider %q and region %q is not supported by any k8s-cluster definition", target, cloud, region)
	}
	if len(matches) > 1 {
		return domain.DeploymentContext{}, domain.Reject(domain.CodeUnsupportedTarget, "target %q needs a region; choose one", target)
	}
	return domain.DeploymentContext{Target: target, CloudProvider: matches[0].CloudProvider, Region: matches[0].Region, TargetSpecificInput: input}, nil
}

// CreateDeployment validates the request, builds graph, waves and plan, and
// persists the deployment in AWAITING_CONFIRMATION (contract C4).
func (o *Orchestrator) CreateDeployment(ctx context.Context, req CreateRequest) (*PlanView, error) {
	app, err := o.Apps.FindApplication(ctx, req.Application)
	if err != nil {
		return nil, err
	}
	version, err := o.Apps.FindVersion(ctx, app.ID, req.Version)
	if err != nil {
		return nil, err
	}
	env := domain.Environment(strings.ToUpper(req.Environment))
	if !env.Valid() {
		return nil, domain.Reject(domain.CodeInvalidInput, "environment must be STAGING or PRODUCTION")
	}
	resolver, err := o.resolver(ctx)
	if err != nil {
		return nil, err
	}
	dctx, err := resolveContext(resolver, req.Target, req.CloudProvider, req.Region, req.TargetSpecificInput)
	if err != nil {
		return nil, err
	}
	if err := o.Deployments.CheckNotInProgress(ctx, app.ID, env, dctx.Target); err != nil {
		return nil, err
	}
	cfg, err := o.Configs.FindByApplicationAndEnvironment(ctx, app.ID, env)
	if err != nil {
		return nil, err
	}

	selected, images, err := selection(version, req.Workloads, req.Images)
	if err != nil {
		return nil, err
	}
	d := &domain.Deployment{
		ApplicationID: app.ID, VersionID: version.VersionID, Environment: env, Target: dctx.Target,
		Kind: domain.KindDeploy, Context: dctx, PlanFingerprintAlgo: domain.FingerprintAlgo,
	}
	if cfg != nil {
		d.EnvironmentConfigurationID = cfg.ID
	}
	for _, id := range selected {
		w := version.Workload(id)
		d.WorkloadDeployments = append(d.WorkloadDeployments, domain.WorkloadDeployment{
			WorkloadID: id, ImageRepository: w.ImageRepository, ImageVersion: images[id], InclusionReason: domain.Selected,
		})
	}

	pc, err := o.buildPlan(ctx, d, version, cfg, resolver)
	if err != nil {
		return nil, err
	}
	if err := o.checkImages(ctx, pc); err != nil {
		return nil, err
	}
	for i := range d.WorkloadDeployments {
		d.WorkloadDeployments[i].WaveNumber = pc.Plan.WaveOf(d.WorkloadDeployments[i].WorkloadID)
	}
	d.PlanFingerprint = pc.Plan.Fingerprint()
	if err := o.Deployments.PersistDeployment(ctx, d); err != nil {
		return nil, err
	}
	return &PlanView{Deployment: d, Plan: pc.Plan, Fingerprint: d.PlanFingerprint, FingerprintAlgo: d.PlanFingerprintAlgo}, nil
}

// CreateTeardown plans removing an application from an environment + target:
// every workload, resource and the target infrastructure the deployments built.
func (o *Orchestrator) CreateTeardown(ctx context.Context, application, environment, target string) (*PlanView, error) {
	app, err := o.Apps.FindApplication(ctx, application)
	if err != nil {
		return nil, err
	}
	env := domain.Environment(strings.ToUpper(environment))
	if !env.Valid() {
		return nil, domain.Reject(domain.CodeInvalidInput, "environment must be STAGING or PRODUCTION")
	}
	if err := o.Deployments.CheckNotInProgress(ctx, app.ID, env, target); err != nil {
		return nil, err
	}
	last, err := o.Deployments.LastDeploymentOfOwner(ctx, app.ID, env, target)
	if err != nil {
		return nil, err
	}
	if last == nil {
		return nil, domain.Reject(domain.CodeNothingToTeardown, "%s has never been deployed to %s on %s", app.Name, env, target)
	}
	version, err := o.Apps.FindVersion(ctx, app.ID, last.VersionID)
	if err != nil {
		return nil, err
	}
	resolver, err := o.resolver(ctx)
	if err != nil {
		return nil, err
	}
	d := &domain.Deployment{
		ApplicationID: app.ID, VersionID: last.VersionID, EnvironmentConfigurationID: last.EnvironmentConfigurationID,
		Environment: env, Target: target, Kind: domain.KindTeardown, Context: last.Context, PlanFingerprintAlgo: domain.FingerprintAlgo,
	}
	pc, err := o.buildPlan(ctx, d, version, nil, resolver)
	if err != nil {
		return nil, err
	}
	if len(pc.Plan.Removals) == 0 {
		return nil, domain.Reject(domain.CodeNothingToTeardown, "nothing of %s is running or provisioned on %s / %s", app.Name, env, target)
	}
	d.PlanFingerprint = pc.Plan.Fingerprint()
	if err := o.Deployments.PersistDeployment(ctx, d); err != nil {
		return nil, err
	}
	return &PlanView{Deployment: d, Plan: pc.Plan, Fingerprint: d.PlanFingerprint, FingerprintAlgo: d.PlanFingerprintAlgo}, nil
}

// GetPlanView rebuilds the transient plan of a stored deployment for display.
func (o *Orchestrator) GetPlanView(ctx context.Context, deploymentID string) (*PlanView, error) {
	pc, err := o.rebuild(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	fp := pc.Plan.Fingerprint()
	return &PlanView{Deployment: pc.Deployment, Plan: pc.Plan, Fingerprint: fp, FingerprintAlgo: pc.Deployment.PlanFingerprintAlgo,
		Stale: fp != pc.Deployment.PlanFingerprint}, nil
}

type ConfirmResult struct {
	Accepted    bool         `json:"accepted"`
	PlanChanged bool         `json:"planChanged"`
	Plan        *domain.Plan `json:"plan,omitempty"`
	Fingerprint string       `json:"fingerprint"`
}

// ConfirmDeployment rebuilds the plan from persisted inputs, compares
// fingerprints, validates overrides and atomically confirms + creates the job
// (contract C5). It returns immediately; execution belongs to the worker.
func (o *Orchestrator) ConfirmDeployment(ctx context.Context, deploymentID string, overrides map[string]map[string]any) (*ConfirmResult, error) {
	d, err := o.Deployments.FindByIDWithPersistedInputs(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	if d.Status != domain.AwaitingConfirmation {
		return nil, domain.Reject(domain.CodeAlreadyConfirmed, "deployment %s is %s, not AWAITING_CONFIRMATION", d.ID, d.Status)
	}
	pc, err := o.rebuildFor(ctx, d)
	if err != nil {
		return nil, err
	}
	rebuilt := pc.Plan.Fingerprint()
	if rebuilt != d.PlanFingerprint {
		if _, err := o.Deployments.CompareAndSetPlanFingerprint(ctx, d.ID, domain.AwaitingConfirmation, d.PlanFingerprint, rebuilt); err != nil {
			return nil, err
		}
		return &ConfirmResult{PlanChanged: true, Plan: pc.Plan, Fingerprint: rebuilt}, nil
	}
	if err := infraplanner.ApplyInfrastructureOverrides(pc.Plan, overrides); err != nil {
		return nil, err
	}
	accepted, err := o.Deployments.ConfirmDeploymentAndCreateJob(ctx, d, overrides)
	if err != nil {
		return nil, err
	}
	if !accepted {
		return nil, domain.Reject(domain.CodeAlreadyConfirmed, "deployment %s was confirmed by another request", d.ID)
	}
	return &ConfirmResult{Accepted: true, Fingerprint: rebuilt}, nil
}

func (o *Orchestrator) rebuild(ctx context.Context, deploymentID string) (*planContext, error) {
	d, err := o.Deployments.FindByIDWithPersistedInputs(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	return o.rebuildFor(ctx, d)
}

// rebuildFor rebuilds graph, waves and plan from a stored deployment, reading
// current catalog, configuration and instance state.
func (o *Orchestrator) rebuildFor(ctx context.Context, d *domain.Deployment) (*planContext, error) {
	version, err := o.Apps.FindVersion(ctx, d.ApplicationID, d.VersionID)
	if err != nil {
		return nil, err
	}
	var cfg *domain.EnvironmentConfiguration
	if d.Kind == domain.KindDeploy {
		if cfg, err = o.Configs.FindByID(ctx, d.EnvironmentConfigurationID); err != nil {
			return nil, err
		}
	}
	resolver, err := o.resolver(ctx)
	if err != nil {
		return nil, err
	}
	// Only the Developer's selection defines the plan; cascaded workloads are
	// added during execution.
	selectedOnly := *d
	selectedOnly.WorkloadDeployments = nil
	for _, wd := range d.WorkloadDeployments {
		if wd.InclusionReason == domain.Selected {
			selectedOnly.WorkloadDeployments = append(selectedOnly.WorkloadDeployments, wd)
		}
	}
	pc, err := o.buildPlan(ctx, &selectedOnly, version, cfg, resolver)
	if err != nil {
		return nil, err
	}
	pc.Deployment = d
	return pc, nil
}

// buildPlan runs validateDeploymentInput -> buildDeploymentGraph ->
// planDeploymentWaves -> resolveResourceDefinitions -> findResourceInstances ->
// planInfrastructureChanges for create, confirm and execution alike.
func (o *Orchestrator) buildPlan(ctx context.Context, d *domain.Deployment, version *domain.ApplicationVersion,
	cfg *domain.EnvironmentConfiguration, resolver *resourceresolver.Resolver) (*planContext, error) {

	running, err := o.Workloads.FindWorkloadInstances(ctx, d.ApplicationID, d.Environment, d.Target)
	if err != nil {
		return nil, err
	}
	instances, err := o.Resources.FindResourceInstances(ctx, d.ApplicationID, d.Environment, d.Target)
	if err != nil {
		return nil, err
	}
	names, err := o.Apps.ComponentNames(ctx, d.ApplicationID)
	if err != nil {
		return nil, err
	}
	pc := &planContext{Deployment: d, Version: version, Configuration: cfg, Resolver: resolver, Running: running, Instances: instances}

	if d.Kind == domain.KindTeardown {
		pc.Graph = graphbuilder.NodesOnly(version, d.Context)
		pc.Graph.ComponentNames = names
		pc.Plan, err = infraplanner.PlanInfrastructureChanges(infraplanner.Input{
			Kind: domain.KindTeardown, Graph: pc.Graph, Environment: d.Environment,
			Instances: instances, Running: running, Resolver: resolver,
		})
		return pc, err
	}

	selected := make([]string, 0, len(d.WorkloadDeployments))
	images := map[string]string{}
	for _, wd := range d.WorkloadDeployments {
		selected = append(selected, wd.WorkloadID)
		images[wd.WorkloadID] = wd.ImageVersion
	}
	if err := o.validateDeploymentInput(ctx, version, cfg, selected, images, running); err != nil {
		return nil, err
	}
	pc.Graph, err = graphbuilder.BuildDeploymentGraph(version, cfg, d.Environment, d.Context, resolver)
	if err != nil {
		return nil, err
	}
	pc.Graph.ComponentNames = names
	if err := validateOutputReferences(pc.Graph, cfg); err != nil {
		return nil, err
	}
	pc.Waves, err = waveplanner.PlanDeploymentWaves(pc.Graph, selected, running)
	if err != nil {
		return nil, err
	}
	pc.Plan, err = infraplanner.PlanInfrastructureChanges(infraplanner.Input{
		Kind: domain.KindDeploy, Graph: pc.Graph, Waves: pc.Waves, Environment: d.Environment, Images: images,
		Instances: instances, Running: running, Configuration: cfg, Resolver: resolver,
	})
	return pc, err
}

var tagPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)

// validateDeploymentInput checks images, configuration against the selected
// version and the partial-deployment version rule (A1).
func (o *Orchestrator) validateDeploymentInput(ctx context.Context, v *domain.ApplicationVersion, cfg *domain.EnvironmentConfiguration,
	selected []string, images map[string]string, running []domain.WorkloadInstance) error {

	p := &domain.ValidationError{}
	if len(v.Workloads) == 0 {
		p.Add(domain.CodeInvalidInput, "version %d has no workloads", v.VersionNumber)
	}
	for _, id := range selected {
		w := v.Workload(id)
		if w == nil {
			p.Add(domain.CodeInvalidInput, "workload %s is not in version %d", id, v.VersionNumber)
			continue
		}
		if !tagPattern.MatchString(images[id]) {
			p.Add(domain.CodeInvalidImage, "%s: image version %q is not a valid tag", w.Name, images[id])
		}
	}

	if cfg == nil {
		p.Add(domain.CodeMissingConfiguration, "the environment has no configuration")
		return p.OrNil()
	}
	bound := map[string]domain.ConfiguredValue{}
	for _, b := range append(append([]domain.ConfiguredValue{}, cfg.Variables...), cfg.Secrets...) {
		bound[b.WorkloadID+"/"+b.DefinitionID] = b
	}
	for _, w := range v.Workloads {
		for _, def := range w.Variables {
			b, ok := bound[w.ID+"/"+def.ID]
			if !ok {
				if def.Required {
					p.Add(domain.CodeMissingConfiguration, "%s.%s is required by version %d but not configured", w.Name, def.Name, v.VersionNumber)
				}
				continue
			}
			o.checkBinding(p, v, &w, def.Name, b, false)
		}
		for _, def := range w.Secrets {
			b, ok := bound[w.ID+"/"+def.ID]
			if !ok {
				if def.Required {
					p.Add(domain.CodeMissingConfiguration, "secret %s.%s is required by version %d but not configured", w.Name, def.Name, v.VersionNumber)
				}
				continue
			}
			o.checkBinding(p, v, &w, def.Name, b, true)
			if b.Source == domain.SourceSecretRef && o.Secrets != nil {
				if _, err := o.Secrets.Get(ctx, b.SecretRef); err != nil {
					p.Add(domain.CodeMissingConfiguration, "secret %s.%s references a value that is not available in the Secret Store", w.Name, def.Name)
				}
			}
		}
	}

	if len(selected) < len(v.Workloads) {
		versions := map[string]int{}
		for _, wi := range running {
			versions[wi.RunningVersionID] = wi.RunningVersionNumber
		}
		switch {
		case len(versions) == 0:
			p.Add(domain.CodePartialVersionMismatch, "a partial deployment needs version %d already running on this environment and target; deploy the whole application", v.VersionNumber)
		case len(versions) > 1:
			p.Add(domain.CodeRunningVersionAmbiguous, "running workloads use different versions; deploy the whole application to converge first")
		default:
			for id, number := range versions {
				if id != v.VersionID {
					p.Add(domain.CodePartialVersionMismatch, "version %d is running; a partial deployment can only use the running version, deploy the whole application to change version", number)
				}
			}
		}
	}
	return p.OrNil()
}

func (o *Orchestrator) checkBinding(p *domain.ValidationError, v *domain.ApplicationVersion, w *domain.Workload, name string, b domain.ConfiguredValue, secret bool) {
	switch b.Source {
	case domain.SourceResourceOutput:
		if v.Resource(b.RefID) == nil {
			p.Add(domain.CodeConfigurationMismatch, "%s.%s references a resource that is not in version %d", w.Name, name, v.VersionNumber)
		} else if !v.DependsOn(w.ID, b.RefID) {
			p.Add(domain.CodeConfigurationMismatch, "%s.%s references %s, which %s does not depend on in version %d", w.Name, name, v.Resource(b.RefID).Name, w.Name, v.VersionNumber)
		}
	case domain.SourceWorkloadOutput:
		target := v.Workload(b.RefID)
		switch {
		case secret:
			p.Add(domain.CodeInvalidOutputReference, "secret %s.%s cannot come from a workload output", w.Name, name)
		case target == nil:
			p.Add(domain.CodeConfigurationMismatch, "%s.%s references a workload that is not in version %d", w.Name, name, v.VersionNumber)
		case !v.DependsOn(w.ID, b.RefID):
			p.Add(domain.CodeConfigurationMismatch, "%s.%s references %s, which %s does not depend on in version %d", w.Name, name, target.Name, w.Name, v.VersionNumber)
		case !contains(target.ExposedOutputs, b.OutputName):
			p.Add(domain.CodeInvalidOutputReference, "%s.%s references output %q, which %s does not expose", w.Name, name, b.OutputName, target.Name)
		}
	case domain.SourceDirect:
		if secret {
			p.Add(domain.CodeInvalidOutputReference, "secret %s.%s cannot be a plaintext direct value", w.Name, name)
		}
	case domain.SourceSecretRef:
		if !secret {
			p.Add(domain.CodeInvalidOutputReference, "%s.%s is a variable but uses a secret reference", w.Name, name)
		}
	}
}

// validateOutputReferences checks resource outputs against the resolved
// definitions: variables use normal outputs, secrets use sensitive outputs.
func validateOutputReferences(g *domain.DeploymentGraph, cfg *domain.EnvironmentConfiguration) error {
	p := &domain.ValidationError{}
	check := func(b domain.ConfiguredValue, secret bool) {
		w := g.Version.Workload(b.WorkloadID)
		if w == nil || b.Source != domain.SourceResourceOutput || g.Version.Resource(b.RefID) == nil {
			return
		}
		res, ok := g.Resolutions[b.RefID]
		if !ok {
			return // resolution problems are reported by the graph builder
		}
		d := res.Definition
		switch {
		case !d.HasOutput(b.OutputName):
			p.Add(domain.CodeInvalidOutputReference, "%s.%s references output %q, which definition %s does not expose", w.Name, b.Name, b.OutputName, d.Name)
		case secret && !d.IsSensitiveOutput(b.OutputName):
			p.Add(domain.CodeInvalidOutputReference, "secret %s.%s must use a sensitive output; %q is not sensitive", w.Name, b.Name, b.OutputName)
		case !secret && d.IsSensitiveOutput(b.OutputName):
			p.Add(domain.CodeInvalidOutputReference, "%s.%s uses sensitive output %q; bind it as a Secret", w.Name, b.Name, b.OutputName)
		}
	}
	for _, b := range cfg.Variables {
		check(b, false)
	}
	for _, b := range cfg.Secrets {
		check(b, true)
	}
	return p.OrNil()
}

// checkImages verifies every selected image exists in the registry the target
// cluster pulls from (A1-1).
func (o *Orchestrator) checkImages(ctx context.Context, pc *planContext) error {
	if o.Images == nil {
		return nil
	}
	mirror := targetadapter.RegistryMirror(pc.Graph)
	p := &domain.ValidationError{}
	for _, wd := range pc.Deployment.WorkloadDeployments {
		ref := targetadapter.ResolveImage(wd.ImageRepository, wd.ImageVersion, mirror)
		if err := o.Images.ImageExists(ctx, ref); err != nil {
			p.Add(domain.CodeInvalidImage, "%s: image %s is not available: %v", pc.Version.Workload(wd.WorkloadID).Name, ref, err)
		}
	}
	return p.OrNil()
}

func selection(v *domain.ApplicationVersion, workloads []string, images map[string]string) ([]string, map[string]string, error) {
	p := &domain.ValidationError{}
	byName := map[string]string{}
	for _, w := range v.Workloads {
		byName[w.Name], byName[w.ID] = w.ID, w.ID
	}
	set := map[string]bool{}
	for _, key := range workloads {
		id, ok := byName[key]
		if !ok {
			p.Add(domain.CodeInvalidInput, "workload %q is not in version %d", key, v.VersionNumber)
			continue
		}
		set[id] = true
	}
	if len(set) == 0 && len(p.Problems) == 0 {
		p.Add(domain.CodeInvalidInput, "select at least one workload to deploy")
	}
	resolved := map[string]string{}
	for key, tag := range images {
		if id, ok := byName[key]; ok {
			resolved[id] = strings.TrimSpace(tag)
		}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, resolved, p.OrNil()
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

// DeploymentForm is the data behind the deployment form (loadDeploymentContext).
type DeploymentForm struct {
	Application    *persistence.ApplicationSummary
	Versions       []persistence.VersionSummary
	Targets        []resourceresolver.TargetOption
	Environment    domain.Environment
	Target         string
	Version        *domain.ApplicationVersion
	RunningVersion int // 0 when nothing runs
	Running        map[string]domain.WorkloadInstance
	Warning        string
}

// LoadDeploymentContext returns versions, supported targets and what currently
// runs on the environment + target.
func (o *Orchestrator) LoadDeploymentContext(ctx context.Context, application, environment, target, version string) (*DeploymentForm, error) {
	app, err := o.Apps.FindApplication(ctx, application)
	if err != nil {
		return nil, err
	}
	versions, err := o.Apps.ListVersions(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	resolver, err := o.resolver(ctx)
	if err != nil {
		return nil, err
	}
	f := &DeploymentForm{Application: app, Versions: versions, Targets: resolver.SupportedTargets(),
		Environment: domain.Environment(strings.ToUpper(environment)), Target: target, Running: map[string]domain.WorkloadInstance{}}
	if !f.Environment.Valid() {
		f.Environment = domain.Staging
	}
	if f.Target == "" && len(f.Targets) > 0 {
		f.Target = f.Targets[0].Target
	}
	running, err := o.Workloads.FindWorkloadInstances(ctx, app.ID, f.Environment, f.Target)
	if err != nil {
		return nil, err
	}
	seen := map[int]bool{}
	for _, wi := range running {
		f.Running[wi.WorkloadID] = wi
		seen[wi.RunningVersionNumber] = true
		f.RunningVersion = wi.RunningVersionNumber
	}
	if len(seen) > 1 {
		f.Warning = "running workloads use different versions; only a full deployment is possible"
	}
	if version == "" {
		switch {
		case f.RunningVersion > 0:
			version = fmt.Sprint(f.RunningVersion)
		case len(versions) > 0:
			version = fmt.Sprint(versions[0].Number)
		}
	}
	if version != "" {
		if f.Version, err = o.Apps.FindVersion(ctx, app.ID, version); err != nil {
			return nil, err
		}
	}
	return f, nil
}
