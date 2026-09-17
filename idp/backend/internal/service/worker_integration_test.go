//go:build integration

package service_test

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pr3s3nt/final_idp/idp/backend/internal/domain"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/domain/manifest"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/domain/resourceoutput"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/integration/cd"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/integration/kubernetes"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/integration/provisioner"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/integration/secretstore"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/service"
)

// fakeProvisioner stands in for Terraform: it records calls and returns
// outputs per module. password_revision changes the password output.
type fakeProvisioner struct {
	mu        sync.Mutex
	calls     []string
	failFor   string
	vars      map[string]map[string]any
	destroyed []string
}

func (f *fakeProvisioner) module(ref string) string {
	return strings.TrimPrefix(ref, "terraform://modules/")
}

func (f *fakeProvisioner) Reconcile(_ context.Context, r provisioner.Request) (*provisioner.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := f.module(r.ProvisionerReference)
	f.calls = append(f.calls, "apply "+m)
	if f.vars == nil {
		f.vars = map[string]map[string]any{}
	}
	f.vars[r.InstanceID] = r.Variables
	if f.failFor == m {
		return nil, errors.New("simulated provider error for " + m)
	}
	return &provisioner.Result{InfrastructureReference: "fake://" + r.InstanceID, ProviderStateReference: "fake-state"}, nil
}

func (f *fakeProvisioner) Destroy(_ context.Context, r provisioner.Request) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "destroy "+f.module(r.ProvisionerReference))
	f.destroyed = append(f.destroyed, r.InstanceID)
	return nil
}

func (f *fakeProvisioner) Outputs(_ context.Context, id string) (map[string]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v := f.vars[id]
	switch {
	case v["node_image"] != nil:
		return map[string]string{"cluster_kind": "kind", "cluster_name": "fake", "kubeconfig": "fake", "image_registry_mirror": "localhost:5055"}, nil
	case v["password_revision"] != nil:
		return map[string]string{"host": "pg." + id[:8], "port": "5432", "database": "app", "username": "app",
			"password": fmt.Sprintf("pw-%v", v["password_revision"])}, nil
	default:
		return map[string]string{"host": "redis." + id[:8], "port": "6379"}, nil
	}
}

type fakeCD struct {
	mu        sync.Mutex
	publishes []cd.DesiredState
	n         int
}

func (f *fakeCD) PublishDesiredDeploymentState(_ context.Context, ds cd.DesiredState) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, w := range ds.Upsert {
		for _, s := range w.Secrets {
			if strings.Contains(string(w.Manifests), string(s.Data["DB_PASSWORD"])) && len(s.Data["DB_PASSWORD"]) > 0 {
				return "", errors.New("secret value leaked into manifests")
			}
		}
	}
	f.publishes = append(f.publishes, ds)
	f.n++
	return fmt.Sprintf("git:%040d", f.n), nil
}
func (f *fakeCD) WaitForDelivery(context.Context, cd.DesiredState, string, time.Duration) error {
	return nil
}
func (f *fakeCD) RemoveApplication(context.Context, cd.DesiredState) error { return nil }
func (f *fakeCD) GetCDStatus(context.Context, cd.DesiredState) (*cd.Status, error) {
	return &cd.Status{Sync: "SYNCED", Health: "HEALTHY"}, nil
}

type fakeKube struct {
	unhealthyImage    string
	removed           []string
	deletedNamespaces []string
}

func (f *fakeKube) WaitForWorkloadsHealthy(_ context.Context, _ *kubernetes.ClusterAccess, ws []kubernetes.WorkloadRef, _ time.Duration) error {
	for _, w := range ws {
		if w.Image == f.unhealthyImage {
			return fmt.Errorf("%s did not become healthy within 1s: CrashLoopBackOff", w.Name)
		}
	}
	return nil
}
func (f *fakeKube) WaitForWorkloadsRemoved(_ context.Context, _ *kubernetes.ClusterAccess, _ string, ids []string, _ time.Duration) error {
	f.removed = append(f.removed, ids...)
	return nil
}
func (f *fakeKube) ReadWorkloadOutputs(_ context.Context, _ *kubernetes.ClusterAccess, ns, id string, outputs []string) (domain.Outputs, error) {
	out := domain.Outputs{}
	for _, o := range outputs {
		out[o] = domain.Output{Value: "http://" + id[:8] + "." + ns + ".svc:8080"}
	}
	return out, nil
}
func (f *fakeKube) DeleteSecretsByLabel(context.Context, *kubernetes.ClusterAccess, string, string) error {
	return nil
}
func (f *fakeKube) DeleteNamespace(_ context.Context, _ *kubernetes.ClusterAccess, ns string) error {
	f.deletedNamespaces = append(f.deletedNamespaces, ns)
	return nil
}
func (f *fakeKube) GetWorkloadStatus(context.Context, *kubernetes.ClusterAccess, string, string) (*kubernetes.WorkloadStatus, error) {
	return nil, nil
}

type harness struct {
	*env
	prov   *fakeProvisioner
	cd     *fakeCD
	kube   *fakeKube
	worker *service.Worker
}

func newHarness(t *testing.T) *harness {
	if _, err := exec.LookPath("score-k8s"); err != nil {
		t.Skip("score-k8s not installed")
	}
	e := setup(t)
	secrets := e.orch.Secrets.(*secretstore.EncryptedFile)
	h := &harness{env: e, prov: &fakeProvisioner{}, cd: &fakeCD{}, kube: &fakeKube{}}
	h.worker = &service.Worker{Orch: e.orch, Provisioner: h.prov, ResourceOutputs: &resourceoutput.Collector{Provisioner: h.prov, Secrets: secrets},
		Kube: h.kube, CD: h.cd, Renderer: &manifest.ScoreRenderer{}, Secrets: secrets, HashKey: []byte("k"),
		HealthTimeout: time.Second, SyncTimeout: time.Second, Logf: t.Logf}
	return h
}

// deploy creates, confirms and executes a deployment, returning its detail.
func (h *harness) deploy(t *testing.T, req service.CreateRequest, overrides map[string]map[string]any) *service.DeploymentDetail {
	t.Helper()
	ctx := context.Background()
	view, err := h.orch.CreateDeployment(ctx, req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return h.confirmAndRun(t, view.Deployment.ID, overrides)
}

func (h *harness) confirmAndRun(t *testing.T, id string, overrides map[string]map[string]any) *service.DeploymentDetail {
	t.Helper()
	ctx := context.Background()
	if res, err := h.orch.ConfirmDeployment(ctx, id, overrides); err != nil || !res.Accepted {
		t.Fatalf("confirm: %+v %v", res, err)
	}
	if processed, err := h.worker.RunOnce(ctx); err != nil || !processed {
		t.Fatalf("worker: processed=%v err=%v", processed, err)
	}
	q := &service.QueryService{Repositories: h.repo, Orch: h.orch}
	detail, err := q.GetDeploymentDetail(ctx, id, false)
	if err != nil {
		t.Fatal(err)
	}
	return detail
}

func stepsOf(d *service.DeploymentDetail) []string {
	var out []string
	for _, s := range d.Steps {
		out = append(out, fmt.Sprintf("%d %s %s %s", s.Wave, s.Component, s.Name, s.Status))
	}
	return out
}

func images(d *service.DeploymentDetail) map[string]string {
	out := map[string]string{}
	for _, i := range d.Images {
		out[i.Workload] = i.InclusionReason + " " + i.Image
	}
	return out
}

func TestWorkerLifecycleOnKindLocal(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	// 1. First full deployment: cluster, data resources, then workloads by wave.
	d1 := h.deploy(t, fullShop("1"), nil)
	if d1.Deployment.Status != domain.Succeeded || d1.JobStatus != "COMPLETED" {
		t.Fatalf("first deployment: %s / job %s\n%s", d1.Deployment.Status, d1.JobStatus, strings.Join(stepsOf(d1), "\n"))
	}
	if got := strings.Join(h.prov.calls, ","); got != "apply postgres-k8s,apply redis-k8s" {
		t.Fatalf("provisioning order (the internal cluster is only linked): %s", got)
	}
	// Resource waves publish nothing; wave 2 publishes backend+worker, wave 3 frontend.
	if len(h.cd.publishes) != 2 || len(h.cd.publishes[0].Upsert) != 2 || len(h.cd.publishes[1].Upsert) != 1 {
		t.Fatalf("expected one publish per workload wave (backend+worker, then frontend); got %d", len(h.cd.publishes))
	}
	for id, vars := range h.prov.vars {
		if vars["password_revision"] != nil && vars["k8s_cluster"] == nil {
			t.Fatalf("postgres-k8s %s did not receive the outputs of the required k8s-cluster", id)
		}
	}
	running, _ := h.repo.Workloads.FindWorkloadInstances(ctx, d1.Deployment.ApplicationID, domain.Staging, "kind-local")
	if len(running) != 3 {
		t.Fatalf("expected 3 running workload instances, got %d", len(running))
	}
	for _, wi := range running {
		if wi.Status != domain.WIHealthy || wi.OutputFingerprint == "" {
			t.Fatalf("workload instance %+v must be HEALTHY with an output fingerprint", wi)
		}
	}
	if len(d1.Infrastructure) != 3 {
		t.Fatalf("record must reference 3 resource instances, got %+v", d1.Infrastructure)
	}

	// 2. Redeploy the same input: every resource REUSE, nothing re-provisioned.
	h.prov.calls = nil
	d2 := h.deploy(t, fullShop("1"), nil)
	if d2.Deployment.Status != domain.Succeeded || len(h.prov.calls) != 0 {
		t.Fatalf("redeploy must reuse infrastructure: %s calls=%v", d2.Deployment.Status, h.prov.calls)
	}

	// 3. A2: a broken frontend image fails at APPLICATION_READY; nothing later runs.
	h.kube.unhealthyImage = "localhost:5055/shop-frontend:broken"
	partial := service.CreateRequest{Application: "shop-app", Version: "1", Environment: "STAGING", Target: "kind-local",
		Workloads: []string{"frontend"}, Images: map[string]string{"frontend": "broken"}}
	d3 := h.deploy(t, partial, nil)
	if d3.Deployment.Status != domain.Failed || d3.Record.ErrorSummary == "" || !strings.Contains(d3.Record.ErrorSummary, "APPLICATION_READY") {
		t.Fatalf("broken image must fail at APPLICATION_READY: %s %q", d3.Deployment.Status, d3.Record.ErrorSummary)
	}
	h.kube.unhealthyImage = ""

	// 4. Partial deployment of the worker with a password rotation: postgresql
	// UPDATE changes its output, so backend (not selected) is redeployed with
	// its running image; frontend's input (backend.endpoint) is unchanged.
	fix := service.CreateRequest{Application: "shop-app", Version: "1", Environment: "STAGING", Target: "kind-local",
		Workloads: []string{"frontend"}, Images: map[string]string{"frontend": "v1"}}
	if d := h.deploy(t, fix, nil); d.Deployment.Status != domain.Succeeded {
		t.Fatalf("recovering frontend: %s", d.Deployment.Status)
	}
	h.prov.calls = nil
	rotate := service.CreateRequest{Application: "shop-app", Version: "1", Environment: "STAGING", Target: "kind-local",
		Workloads: []string{"worker"}, Images: map[string]string{"worker": "v2"}}
	d4 := h.deploy(t, rotate, map[string]map[string]any{"postgresql": {"password_revision": 1}})
	if d4.Deployment.Status != domain.Succeeded {
		t.Fatalf("rotation deployment failed: %s\n%s", d4.Record.ErrorSummary, strings.Join(stepsOf(d4), "\n"))
	}
	imgs := images(d4)
	if imgs["worker"] != "SELECTED registry.company.local/shop-worker:v2" || imgs["backend"] != "CASCADED registry.company.local/shop-backend:v1" {
		t.Fatalf("cascade: %v", imgs)
	}
	if _, ok := imgs["frontend"]; ok {
		t.Fatalf("frontend must not cascade when backend's outputs did not change: %v", imgs)
	}
	if strings.Join(h.prov.calls, ",") != "apply postgres-k8s" {
		t.Fatalf("only postgresql should be updated: %v", h.prov.calls)
	}

	// 5. Next deployment without overrides keeps the rotated password (baseline).
	h.prov.calls = nil
	if d := h.deploy(t, rotate, nil); d.Deployment.Status != domain.Succeeded || len(h.prov.calls) != 0 {
		t.Fatalf("override baseline must be kept (REUSE): %v", h.prov.calls)
	}

	// 6. Version 2: worker removed first, then redis destroyed; resource rows kept.
	h.prov.calls = nil
	v2 := fullShop("2")
	v2.Workloads, v2.Images = []string{"backend", "frontend"}, map[string]string{"backend": "v2", "frontend": "v2"}
	d6 := h.deploy(t, v2, nil)
	if d6.Deployment.Status != domain.Succeeded {
		t.Fatalf("v2: %s\n%s", d6.Record.ErrorSummary, strings.Join(stepsOf(d6), "\n"))
	}
	if strings.Join(h.prov.calls, ",") != "destroy redis-k8s" || len(h.kube.removed) != 1 {
		t.Fatalf("v2 must remove worker then destroy redis: calls=%v removed=%v", h.prov.calls, h.kube.removed)
	}
	sort.Strings(d6.Removed)
	if strings.Join(d6.Removed, ",") != "redis,worker" {
		t.Fatalf("removed components: %v", d6.Removed)
	}
	last := stepsOf(d6)
	if !strings.Contains(last[len(last)-2], "worker REMOVED SUCCEEDED") || !strings.Contains(last[len(last)-1], "redis DESTROYED SUCCEEDED") {
		t.Fatalf("removal must come after all waves: %v", last)
	}

	// 7. Partial deployment with a different version is rejected (A1-4).
	p := service.CreateRequest{Application: "shop-app", Version: "1", Environment: "STAGING", Target: "kind-local",
		Workloads: []string{"backend"}, Images: map[string]string{"backend": "v1"}}
	if _, err := h.orch.CreateDeployment(ctx, p); !domain.HasCode(err, domain.CodePartialVersionMismatch) {
		t.Fatalf("expected PARTIAL_DEPLOYMENT_VERSION_MISMATCH, got %v", err)
	}

	// 8. Back to version 1: worker is deployed again (new instance row) and redis re-created.
	h.prov.calls = nil
	d8 := h.deploy(t, fullShop("1"), nil)
	if d8.Deployment.Status != domain.Succeeded || strings.Join(h.prov.calls, ",") != "apply redis-k8s" {
		t.Fatalf("returning to v1: %s calls=%v\n%s", d8.Deployment.Status, h.prov.calls, d8.Record.ErrorSummary)
	}

	// 9. Teardown: workloads, then data resources; the internal cluster is only unlinked.
	h.prov.calls = nil
	td, err := h.orch.CreateTeardown(ctx, "shop-app", "STAGING", "kind-local")
	if err != nil {
		t.Fatal(err)
	}
	if td.Deployment.Kind != domain.KindTeardown || td.Deployment.Status != domain.AwaitingConfirmation {
		t.Fatalf("UC-05 must create an awaiting TEARDOWN, got %s / %s", td.Deployment.Kind, td.Deployment.Status)
	}
	if len(td.Deployment.WorkloadDeployments) != 0 || len(td.Plan.Waves) != 0 {
		t.Fatalf("a teardown must not deploy workloads: snapshots=%d waves=%d", len(td.Deployment.WorkloadDeployments), len(td.Plan.Waves))
	}
	var workloadRows, jobsBefore, recordsBefore int
	if err := h.db.Pool.QueryRow(ctx, `SELECT count(*) FROM workload_deployment WHERE deployment_id = $1`, td.Deployment.ID).Scan(&workloadRows); err != nil {
		t.Fatal(err)
	}
	if err := h.db.Pool.QueryRow(ctx, `SELECT count(*) FROM deployment_execution_job WHERE deployment_id = $1`, td.Deployment.ID).Scan(&jobsBefore); err != nil {
		t.Fatal(err)
	}
	if err := h.db.Pool.QueryRow(ctx, `SELECT count(*) FROM deployment_record WHERE deployment_id = $1`, td.Deployment.ID).Scan(&recordsBefore); err != nil {
		t.Fatal(err)
	}
	if workloadRows != 0 || jobsBefore != 0 || recordsBefore != 0 {
		t.Fatalf("planning UC-05 must persist no workload, job or record rows: workloads=%d jobs=%d records=%d", workloadRows, jobsBefore, recordsBefore)
	}
	actions := map[string]int{}
	for _, wave := range td.Plan.Removals {
		for _, it := range wave.Items {
			actions[it.Action]++
			if it.Action == domain.ActionDestroy && !it.DataLossWarning {
				t.Fatalf("managed resource %s must carry a data-loss warning", it.Name)
			}
			if it.Action != domain.ActionDestroy && it.DataLossWarning {
				t.Fatalf("non-destructive action %s on %s must not carry a data-loss warning", it.Action, it.Name)
			}
		}
	}
	if actions[domain.ActionRemove] != 3 || actions[domain.ActionDestroy] != 2 || actions[domain.ActionUnlink] != 1 {
		t.Fatalf("UC-05 removal plan actions = %v, want 3 REMOVE, 2 DESTROY, 1 UNLINK", actions)
	}
	confirmed, err := h.orch.ConfirmDeployment(ctx, td.Deployment.ID, nil)
	if err != nil || !confirmed.Accepted {
		t.Fatalf("confirm teardown: %+v %v", confirmed, err)
	}
	var jobsAfter int
	if err := h.db.Pool.QueryRow(ctx, `SELECT count(*) FROM deployment_execution_job WHERE deployment_id = $1`, td.Deployment.ID).Scan(&jobsAfter); err != nil {
		t.Fatal(err)
	}
	if jobsAfter != 1 {
		t.Fatalf("confirming UC-05 must create exactly one job, got %d", jobsAfter)
	}
	if _, err := h.orch.ConfirmDeployment(ctx, td.Deployment.ID, nil); !domain.HasCode(err, domain.CodeAlreadyConfirmed) {
		t.Fatalf("confirming UC-05 twice must be rejected, got %v", err)
	}
	if processed, err := h.worker.RunOnce(ctx); err != nil || !processed {
		t.Fatalf("worker: processed=%v err=%v", processed, err)
	}
	q := &service.QueryService{Repositories: h.repo, Orch: h.orch}
	d9, err := q.GetDeploymentDetail(ctx, td.Deployment.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if d9.Deployment.Status != domain.Succeeded {
		t.Fatalf("teardown: %s", d9.Record.ErrorSummary)
	}
	calls := h.prov.calls
	if strings.Join(calls, ",") != "destroy postgres-k8s,destroy redis-k8s" {
		t.Fatalf("teardown must destroy the data resources and never the internal cluster: %v", calls)
	}
	var unlinked bool
	for _, s := range d9.Steps {
		unlinked = unlinked || (s.Component == "k8s-cluster" && s.Name == "UNLINKED" && s.Status == "SUCCEEDED")
	}
	if !unlinked {
		t.Fatalf("teardown must unlink the internal cluster: %v", stepsOf(d9))
	}
	if strings.Join(h.kube.deletedNamespaces, ",") != "shop-app-staging" {
		t.Fatalf("teardown must delete the application namespace on the shared cluster: %v", h.kube.deletedNamespaces)
	}
	left, _ := h.repo.Resources.FindResourceInstances(ctx, d9.Deployment.ApplicationID, domain.Staging, "kind-local")
	runningAfter, _ := h.repo.Workloads.FindWorkloadInstances(ctx, d9.Deployment.ApplicationID, domain.Staging, "kind-local")
	if len(left) != 0 || len(runningAfter) != 0 {
		t.Fatalf("teardown must leave no active instances: %d resources, %d workloads", len(left), len(runningAfter))
	}
	deploymentsBefore := h.count(t, "deployment")
	if _, err := h.orch.CreateTeardown(ctx, "shop-app", "STAGING", "kind-local"); !domain.HasCode(err, domain.CodeNothingToTeardown) {
		t.Fatalf("a second teardown with nothing left must be rejected, got %v", err)
	}
	if got := h.count(t, "deployment"); got != deploymentsBefore {
		t.Fatalf("a rejected empty teardown must not leave a deployment row: before=%d after=%d", deploymentsBefore, got)
	}
}

func TestUC05PlanDriftAndConfirmedOwnerGuard(t *testing.T) {
	t.Run("plan drift before confirmation", func(t *testing.T) {
		h := newHarness(t)
		ctx := context.Background()
		deployed := h.deploy(t, fullShop("1"), nil)
		td, err := h.orch.CreateTeardown(ctx, "shop-app", "STAGING", "kind-local")
		if err != nil {
			t.Fatal(err)
		}
		instances, err := h.repo.Resources.FindResourceInstances(ctx, deployed.Deployment.ApplicationID, domain.Staging, "kind-local")
		if err != nil || len(instances) == 0 {
			t.Fatalf("active resources: %v / %d", err, len(instances))
		}
		if err := h.repo.Resources.UpdateStatus(ctx, instances[0].ID, domain.RIDestroyed); err != nil {
			t.Fatal(err)
		}
		res, err := h.orch.ConfirmDeployment(ctx, td.Deployment.ID, nil)
		if err != nil || !res.PlanChanged || res.Accepted {
			t.Fatalf("changed teardown plan must be re-presented: %+v %v", res, err)
		}
		var jobs int
		if err := h.db.Pool.QueryRow(ctx, `SELECT count(*) FROM deployment_execution_job WHERE deployment_id = $1`, td.Deployment.ID).Scan(&jobs); err != nil {
			t.Fatal(err)
		}
		if jobs != 0 {
			t.Fatalf("PLAN_CHANGED must not create a job, got %d", jobs)
		}
	})

	t.Run("confirmed owner blocks another teardown", func(t *testing.T) {
		h := newHarness(t)
		ctx := context.Background()
		h.deploy(t, fullShop("1"), nil)
		td, err := h.orch.CreateTeardown(ctx, "shop-app", "STAGING", "kind-local")
		if err != nil {
			t.Fatal(err)
		}
		if res, err := h.orch.ConfirmDeployment(ctx, td.Deployment.ID, nil); err != nil || !res.Accepted {
			t.Fatalf("confirm teardown: %+v %v", res, err)
		}
		if _, err := h.orch.CreateTeardown(ctx, "shop-app", "STAGING", "kind-local"); !domain.HasCode(err, domain.CodeDeploymentInProgress) {
			t.Fatalf("a confirmed teardown must hold the owner, got %v", err)
		}
	})
}

func TestWorkerFailuresAndPlanDrift(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	// A2-1: provisioning failure stops the deployment before any workload.
	h.prov.failFor = "postgres-k8s"
	d := h.deploy(t, fullShop("1"), nil)
	if d.Deployment.Status != domain.Failed || len(h.cd.publishes) != 0 {
		t.Fatalf("provisioning failure must fail before publishing: %s publishes=%d", d.Deployment.Status, len(h.cd.publishes))
	}
	var failed []string
	for _, s := range d.Steps {
		if s.Status == "FAILED" {
			failed = append(failed, s.Component+" "+s.Name)
		}
	}
	if strings.Join(failed, ",") != "postgresql INFRASTRUCTURE_READY" {
		t.Fatalf("failed steps: %v", failed)
	}
	instances, _ := h.repo.Resources.FindResourceInstances(ctx, d.Deployment.ApplicationID, domain.Staging, "kind-local")
	statuses := map[string]string{}
	for _, ri := range instances {
		def, _ := h.repo.Catalog.FindByID(ctx, ri.DefinitionID)
		statuses[def.Name] = string(ri.Status)
	}
	if statuses["postgres-k8s"] != "FAILED" || statuses["kind-internal-cluster"] != "READY" {
		t.Fatalf("instance statuses after failure: %v", statuses)
	}

	// Retry: the failed instance is updated (not duplicated) and succeeds.
	h.prov.failFor = ""
	if d := h.deploy(t, fullShop("1"), nil); d.Deployment.Status != domain.Succeeded {
		t.Fatalf("retry after failure: %s", d.Record.ErrorSummary)
	}

	// A2-7: infrastructure state changes between confirmation and execution
	// (catalog versions are immutable, so the drift comes from an instance).
	view, err := h.orch.CreateDeployment(ctx, fullShop("1"))
	if err != nil {
		t.Fatal(err)
	}
	if res, err := h.orch.ConfirmDeployment(ctx, view.Deployment.ID, nil); err != nil || !res.Accepted {
		t.Fatal(err)
	}
	if _, err := h.db.Pool.Exec(ctx, `UPDATE resource_instance ri SET applied_input_fingerprint = 'drift' FROM resource_definition d
		WHERE d.resource_definition_id = ri.resource_definition_id AND d.name = 'redis-k8s'`); err != nil {
		t.Fatal(err)
	}
	h.prov.calls = nil
	if _, err := h.worker.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	after, _ := h.repo.Deployments.FindByIDWithPersistedInputs(ctx, view.Deployment.ID)
	rec, _ := h.repo.Deployments.GetDeploymentRecord(ctx, view.Deployment.ID)
	if after.Status != domain.Failed || !strings.Contains(rec.ErrorSummary, domain.CodePlanChangedBeforeExecute) || len(h.prov.calls) != 0 {
		t.Fatalf("drift after confirmation must fail before side effects: %s %q calls=%v", after.Status, rec.ErrorSummary, h.prov.calls)
	}
}

func TestCatalogVersionsAndClusterOutputPropagation(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	v1 := fullShop("1")
	v1.CatalogVersion = "1"
	if d := h.deploy(t, v1, nil); d.Deployment.Status != domain.Succeeded || d.Deployment.CatalogVersionNumber != 1 {
		t.Fatalf("deploy with catalog 1: %s, catalog %d", d.Deployment.Status, d.Deployment.CatalogVersionNumber)
	}

	// A partial deployment cannot switch the catalog version (A1); without a
	// catalog version it uses the running one.
	partial := service.CreateRequest{Application: "shop-app", Version: "1", CatalogVersion: "2", Environment: "STAGING", Target: "kind-local",
		Workloads: []string{"frontend"}, Images: map[string]string{"frontend": "v1"}}
	if _, err := h.orch.CreateDeployment(ctx, partial); !domain.HasCode(err, domain.CodePartialCatalogMismatch) {
		t.Fatalf("expected PARTIAL_DEPLOYMENT_CATALOG_VERSION_MISMATCH, got %v", err)
	}
	partial.CatalogVersion = ""
	if view, err := h.orch.CreateDeployment(ctx, partial); err != nil || view.Plan.CatalogVersion != 1 {
		t.Fatalf("the running catalog version is preselected: %v", err)
	}

	// The whole application moves to catalog 2, where only postgres-k8s changed.
	h.prov.calls = nil
	v2 := fullShop("1")
	v2.CatalogVersion = "2"
	d := h.deploy(t, v2, nil)
	if d.Deployment.Status != domain.Succeeded || strings.Join(h.prov.calls, ",") != "apply postgres-k8s" {
		t.Fatalf("catalog 2: %s calls=%v", d.Deployment.Status, h.prov.calls)
	}
	instances, _ := h.repo.Resources.FindResourceInstances(ctx, d.Deployment.ApplicationID, domain.Staging, "kind-local")
	for _, ri := range instances {
		def, _ := h.repo.Catalog.FindByID(ctx, ri.DefinitionID)
		if def.CatalogVersionID != d.Deployment.CatalogVersionID {
			t.Fatalf("%s still points at another catalog version after reuse", def.Name)
		}
	}
	// Going back to an older catalog version is allowed.
	h.prov.calls = nil
	if d := h.deploy(t, v1, nil); d.Deployment.Status != domain.Succeeded || strings.Join(h.prov.calls, ",") != "apply postgres-k8s" {
		t.Fatalf("back to catalog 1: %s calls=%v", d.Deployment.Status, h.prov.calls)
	}

	// The platform changes the internal cluster's connection record. During a
	// partial frontend deployment the outputs of k8s-cluster change, so
	// postgresql and redis (they require it) are applied again and backend and
	// worker (they run on it) are redeployed with their running images.
	secrets := h.orch.Secrets.(*secretstore.EncryptedFile)
	if _, err := secrets.Put(ctx, "platform/kind-internal-cluster", []byte(clusterRecord("fake-rotated"))); err != nil {
		t.Fatal(err)
	}
	h.prov.calls = nil
	partial.CatalogVersion = "1"
	d = h.deploy(t, partial, nil)
	if d.Deployment.Status != domain.Succeeded {
		t.Fatalf("cluster change: %s\n%s", d.Record.ErrorSummary, strings.Join(stepsOf(d), "\n"))
	}
	calls := append([]string{}, h.prov.calls...)
	sort.Strings(calls)
	if strings.Join(calls, ",") != "apply postgres-k8s,apply redis-k8s" {
		t.Fatalf("resources requiring the changed cluster must be applied again: %v", h.prov.calls)
	}
	imgs := images(d)
	if !strings.HasPrefix(imgs["backend"], "CASCADED") || !strings.HasPrefix(imgs["worker"], "CASCADED") || !strings.HasPrefix(imgs["frontend"], "SELECTED") {
		t.Fatalf("workloads on the changed cluster must cascade: %v", imgs)
	}
	again := 0
	for _, s := range d.Steps {
		if s.Detail["appliedAgainBecauseOutputsChanged"] == "k8s-cluster" {
			again++
		}
	}
	if again != 2 {
		t.Fatalf("steps must show why postgresql and redis were applied again: %v", stepsOf(d))
	}
	// The applied input now includes the new cluster outputs: nothing is applied next time.
	h.prov.calls = nil
	if d := h.deploy(t, partial, nil); d.Deployment.Status != domain.Succeeded || len(h.prov.calls) != 0 {
		t.Fatalf("after propagation the resources are up to date: %s calls=%v", d.Deployment.Status, h.prov.calls)
	}
}
