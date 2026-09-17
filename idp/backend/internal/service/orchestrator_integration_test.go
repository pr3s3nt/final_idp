//go:build integration

package service_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pr3s3nt/final_idp/idp/backend/internal/domain"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/fixtures"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/integration/secretstore"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/persistence"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/service"
)

type allImagesExist struct{ missing string }

func (a allImagesExist) ImageExists(_ context.Context, ref string) error {
	if a.missing != "" && ref == a.missing {
		return os.ErrNotExist
	}
	return nil
}

type env struct {
	db   *persistence.DB
	repo service.Repositories
	orch *service.Orchestrator
}

func setup(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	url := os.Getenv("IDP_TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://idp:idp@127.0.0.1:55433/idp_test?sslmode=disable"
	}
	db, err := persistence.Open(ctx, url)
	if err != nil {
		t.Fatalf("test database: %v", err)
	}
	t.Cleanup(db.Close)
	if err := db.ResetForTests(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	secrets, err := secretstore.NewEncryptedFile(t.TempDir(), "integration-test-secret-key")
	if err != nil {
		t.Fatal(err)
	}
	// The platform-owned internal cluster the kind-local catalog definition points to.
	if _, err := secrets.Put(ctx, "platform/kind-internal-cluster", []byte(clusterRecord("fake"))); err != nil {
		t.Fatal(err)
	}
	repo := service.Repositories{
		Apps: &persistence.ApplicationRepository{DB: db}, Configs: &persistence.EnvironmentConfigurationRepository{DB: db},
		Catalog: &persistence.ResourceDefinitionCatalog{DB: db}, Resources: &persistence.ResourceInstanceRepository{DB: db},
		Workloads: &persistence.WorkloadInstanceRepository{DB: db}, Deployments: &persistence.DeploymentRepository{DB: db},
	}
	_, file, _, _ := runtime.Caller(0)
	os.Setenv("IDP_AWS_ECR_REGISTRY", "123456789012.dkr.ecr.ap-southeast-1.amazonaws.com")
	im := &fixtures.Importer{Apps: repo.Apps, Configs: repo.Configs, Catalog: repo.Catalog, Secrets: secrets, Log: t.Logf}
	if err := im.ImportDirectory(ctx, filepath.Join(filepath.Dir(file), "..", "..", "fixtures")); err != nil {
		t.Fatal(err)
	}
	return &env{db: db, repo: repo, orch: &service.Orchestrator{Repositories: repo, Images: allImagesExist{}, Secrets: secrets}}
}

func clusterRecord(kubeconfig string) string {
	return `{"cluster_name":"idp-internal","cluster_kind":"kind","image_registry_mirror":"localhost:5055","kubeconfig":"` + kubeconfig + `"}`
}

func (e *env) count(t *testing.T, table string) int {
	t.Helper()
	var n int
	if err := e.db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func fullShop(version string) service.CreateRequest {
	return service.CreateRequest{Application: "shop-app", Version: version, Environment: "STAGING", Target: "kind-local",
		Workloads: []string{"backend", "worker", "frontend"},
		Images:    map[string]string{"backend": "v1", "worker": "v1", "frontend": "v1"}}
}

func TestCreateAndConfirmFullDeployment(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	view, err := e.orch.CreateDeployment(ctx, fullShop("1"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Deployment.Status != domain.AwaitingConfirmation || len(view.Plan.Waves) != 4 {
		t.Fatalf("unexpected plan: status %s, %d waves", view.Deployment.Status, len(view.Plan.Waves))
	}
	if got := view.Plan.Waves[0].Items[0]; got.Name != "k8s-cluster" || got.Action != domain.ActionLink || got.DefinitionName != "kind-internal-cluster" {
		t.Fatalf("wave 0 should link the internal cluster, got %+v", got)
	}
	if view.Plan.CatalogVersion != 2 || view.Deployment.CatalogVersionNumber != 2 {
		t.Fatalf("a first deployment without a catalog version uses the newest one, got %d", view.Plan.CatalogVersion)
	}
	if e.count(t, "deployment") != 1 || e.count(t, "workload_deployment") != 3 || e.count(t, "deployment_context") != 1 || e.count(t, "deployment_execution_job") != 0 {
		t.Fatal("create must persist deployment, 3 workload deployments and context, and no job")
	}

	// Plan view rebuilt from persisted inputs matches the stored fingerprint.
	pv, err := e.orch.GetPlanView(ctx, view.Deployment.ID)
	if err != nil || pv.Stale {
		t.Fatalf("rebuilt plan must match: stale=%v err=%v", pv != nil && pv.Stale, err)
	}

	// Invalid override keeps AWAITING_CONFIRMATION and creates no job (A1-9).
	_, err = e.orch.ConfirmDeployment(ctx, view.Deployment.ID, map[string]map[string]any{"postgresql": {"storage_gb": 99}})
	if !domain.HasCode(err, domain.CodeInvalidOverride) || e.count(t, "deployment_execution_job") != 0 {
		t.Fatalf("invalid override must be rejected without a job: %v", err)
	}

	res, err := e.orch.ConfirmDeployment(ctx, view.Deployment.ID, map[string]map[string]any{"postgresql": {"storage_gb": 2}})
	if err != nil || !res.Accepted {
		t.Fatalf("confirm: %+v %v", res, err)
	}
	d, _ := e.repo.Deployments.FindByIDWithPersistedInputs(ctx, view.Deployment.ID)
	if d.Status != domain.Confirmed || e.count(t, "deployment_execution_job") != 1 {
		t.Fatalf("confirm must set CONFIRMED with exactly one job, got %s", d.Status)
	}

	// Second confirm is rejected and creates no second job (A1-10).
	if _, err := e.orch.ConfirmDeployment(ctx, view.Deployment.ID, nil); !domain.HasCode(err, domain.CodeAlreadyConfirmed) || e.count(t, "deployment_execution_job") != 1 {
		t.Fatalf("second confirm must fail: %v", err)
	}

	// Another deployment on the same environment + target is blocked (A1-12).
	if _, err := e.orch.CreateDeployment(ctx, fullShop("1")); !domain.HasCode(err, domain.CodeDeploymentInProgress) {
		t.Fatalf("expected DEPLOYMENT_IN_PROGRESS, got %v", err)
	}
	// Production is a different environment and is not blocked.
	prod := fullShop("1")
	prod.Environment = "PRODUCTION"
	if _, err := e.orch.CreateDeployment(ctx, prod); err != nil {
		t.Fatalf("production must not be blocked by staging: %v", err)
	}
}

func TestRejectedDeploymentsLeaveNoRows(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	cases := map[string]struct {
		req  service.CreateRequest
		code string
	}{
		"missing required configuration (v3 FEATURE_FLAGS)": {func() service.CreateRequest {
			r := fullShop("3")
			r.Workloads, r.Images = []string{"backend", "frontend"}, map[string]string{"backend": "v1", "frontend": "v1"}
			return r
		}(), domain.CodeMissingConfiguration},
		"invalid image tag": {func() service.CreateRequest {
			r := fullShop("1")
			r.Images["backend"] = "not a tag!"
			return r
		}(), domain.CodeInvalidImage},
		"partial deployment with nothing running": {func() service.CreateRequest {
			r := fullShop("1")
			r.Workloads = []string{"frontend"}
			return r
		}(), domain.CodePartialVersionMismatch},
		"unknown catalog version": {func() service.CreateRequest {
			r := fullShop("1")
			r.CatalogVersion = "9"
			return r
		}(), domain.CodeNotFound},
		"unsupported target": {func() service.CreateRequest {
			r := fullShop("1")
			r.Target = "gcp"
			return r
		}(), domain.CodeUnsupportedTarget},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := e.orch.CreateDeployment(ctx, c.req)
			if !domain.HasCode(err, c.code) {
				t.Fatalf("expected %s, got %v", c.code, err)
			}
			if e.count(t, "deployment") != 0 || e.count(t, "workload_deployment") != 0 || e.count(t, "deployment_context") != 0 {
				t.Fatal("a rejected attempt must not leave deployment rows")
			}
		})
	}

	missing := &service.Orchestrator{Repositories: e.repo, Images: allImagesExist{missing: "localhost:5055/shop-backend:v9"}}
	r := fullShop("1")
	r.Images["backend"] = "v9"
	if _, err := missing.CreateDeployment(ctx, r); !domain.HasCode(err, domain.CodeInvalidImage) {
		t.Fatalf("an image missing from the target registry must be rejected, got %v", err)
	}
}

func TestAWSPlanAndSharedResourceResolution(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	r := fullShop("1")
	r.Target = "aws"
	view, err := e.orch.CreateDeployment(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	var wave0, wave1 []string
	for _, it := range view.Plan.Waves[0].Items {
		wave0 = append(wave0, it.DefinitionName)
	}
	for _, it := range view.Plan.Waves[1].Items {
		wave1 = append(wave1, it.DefinitionName)
	}
	if len(wave0) != 1 || wave0[0] != "aws-network" || len(wave1) != 3 {
		t.Fatalf("aws waves: %v / %v", wave0, wave1)
	}

	rep := service.CreateRequest{Application: "reporting-app", Version: "1", Environment: "STAGING", Target: "kind-local",
		Workloads: []string{"reporter"}, Images: map[string]string{"reporter": "v1"}}
	view, err = e.orch.CreateDeployment(ctx, rep)
	if err != nil {
		t.Fatal(err)
	}
	it := view.Plan.Item(view.Plan.Waves[0].Items[0].ComponentID)
	var db *domain.PlanItem
	for _, x := range view.Plan.Items() {
		if x.Name == "reportsdb" {
			db = x
		}
	}
	if it == nil || db == nil || db.Action != domain.ActionLink || db.ManagementMode != domain.Existing {
		t.Fatalf("reporting-app staging must LINK the shared database, got %+v", db)
	}
}
