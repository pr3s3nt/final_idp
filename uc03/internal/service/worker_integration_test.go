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

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/manifest"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/resourceoutput"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/cd"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/kubernetes"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/provisioner"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/secretstore"
	"github.com/pr3s3nt/final_idp/uc03/internal/service"
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

func (f *fakeProvisioner) module(ref string) string { return strings.TrimPrefix(ref, "terraform://modules/") }

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
func (f *fakeCD) WaitForDelivery(context.Context, cd.DesiredState, string, time.Duration) error { return nil }
func (f *fakeCD) RemoveApplication(context.Context, cd.DesiredState) error                    { return nil }
func (f *fakeCD) GetCDStatus(context.Context, cd.DesiredState) (*cd.Status, error) {
	return &cd.Status{Sync: "SYNCED", Health: "HEALTHY"}, nil
}

type fakeKube struct {
	unhealthyImage string
	removed        []string
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
	if got := strings.Join(h.prov.calls, ","); got != "apply kind-cluster,apply postgres-k8s,apply redis-k8s" {
		t.Fatalf("provisioning order: %s", got)
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

	// 9. Teardown: workloads, then data resources, then the cluster.
	h.prov.calls = nil
	td, err := h.orch.CreateTeardown(ctx, "shop-app", "STAGING", "kind-local")
	if err != nil {
		t.Fatal(err)
	}
	d9 := h.confirmAndRun(t, td.Deployment.ID, nil)
	if d9.Deployment.Status != domain.Succeeded {
		t.Fatalf("teardown: %s", d9.Record.ErrorSummary)
	}
	calls := h.prov.calls
	if len(calls) != 3 || calls[2] != "destroy kind-cluster" {
		t.Fatalf("teardown must destroy data resources before the cluster: %v", calls)
	}
	left, _ := h.repo.Resources.FindResourceInstances(ctx, d9.Deployment.ApplicationID, domain.Staging, "kind-local")
	runningAfter, _ := h.repo.Workloads.FindWorkloadInstances(ctx, d9.Deployment.ApplicationID, domain.Staging, "kind-local")
	if len(left) != 0 || len(runningAfter) != 0 {
		t.Fatalf("teardown must leave no active instances: %d resources, %d workloads", len(left), len(runningAfter))
	}
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
	if statuses["postgres-k8s"] != "FAILED" || statuses["kind-cluster"] != "READY" {
		t.Fatalf("instance statuses after failure: %v", statuses)
	}

	// Retry: the failed instance is updated (not duplicated) and succeeds.
	h.prov.failFor = ""
	if d := h.deploy(t, fullShop("1"), nil); d.Deployment.Status != domain.Succeeded {
		t.Fatalf("retry after failure: %s", d.Record.ErrorSummary)
	}

	// A2-7: catalog changes between confirmation and execution.
	view, err := h.orch.CreateDeployment(ctx, fullShop("1"))
	if err != nil {
		t.Fatal(err)
	}
	if res, err := h.orch.ConfirmDeployment(ctx, view.Deployment.ID, nil); err != nil || !res.Accepted {
		t.Fatal(err)
	}
	def, _ := h.repo.Catalog.List(ctx)
	for i := range def {
		if def[i].Name == "redis-k8s" {
			def[i].DefaultParameters["maxmemory"] = "64mb"
			if err := h.repo.Catalog.Upsert(ctx, &def[i]); err != nil {
				t.Fatal(err)
			}
		}
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
