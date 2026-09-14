package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/fixture"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/identity"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/platform"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/service"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/store/postgres"
)

func TestPrepareConfirmClaimAndReserve(t *testing.T) {
	databaseURL := os.Getenv("IDP_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("IDP_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	data, err := fixture.Load("../../../fixtures/uc3-demo.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SeedFixture(ctx, data); err != nil {
		t.Fatal(err)
	}
	deploymentService := service.NewDeploymentService(store, ".state/resources")
	created, err := deploymentService.Create(ctx, service.CreateDeploymentRequest{ApplicationID: data.Snapshot.ApplicationDefinition.ID, Environment: "dev", Context: domain.DeploymentContext{TargetID: "aws-demo", CloudProvider: "aws", Region: "ap-southeast-1", ClusterIdentity: "test-cluster", Namespace: "idp-demo-dev", NamingPolicy: "mvp-v1", RendererVersion: "score-k8s-0.15.0", AdapterVersions: map[string]string{"kubeconfigPath": "/tmp/test", "kubeContext": "test"}, ProvisionerInputs: map[string]any{"dbSubnetGroupName": "test-subnets", "vpcSecurityGroupIds": []string{"sg-0123456789abcdef0"}}}, Images: data.Snapshot.Images})
	if err != nil {
		t.Fatal(err)
	}
	confirmed, err := deploymentService.Confirm(ctx, created.DeploymentID, "integration-key", service.ConfirmDeploymentRequest{ExpectedPlanFingerprint: created.PlanFingerprint, Overrides: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.TrackingID == "" {
		t.Fatal("missing tracking id")
	}
	workerID, err := identity.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	job, found, err := store.ClaimNext(ctx, workerID)
	if err != nil || !found {
		t.Fatalf("claim found=%v err=%v", found, err)
	}
	definitions, instances, err := store.PlanningData(ctx, job.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := platform.NewEngine(".state/resources").Prepare(job.Snapshot, definitions, instances)
	if err != nil {
		t.Fatal(err)
	}
	item := prepared.Plan.Items[0]
	if err := store.ReserveResource(ctx, job.JobID, workerID, job.Snapshot, item); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkProvisioning(ctx, job.JobID, workerID, item.ReferencedResourceInstanceID); err != nil {
		t.Fatal(err)
	}
	if err := store.FailExecution(ctx, job.JobID, workerID, "TEST_STOP", "integration test stops before provider apply", true); err != nil {
		t.Fatal(err)
	}
	detail, err := deploymentService.Get(ctx, created.DeploymentID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.LifecycleStatus != "FAILED" || detail.Execution == nil || detail.Execution.FailureCode != "TEST_STOP" {
		t.Fatalf("detail = %#v", detail)
	}
	if len(detail.Steps) != 3 || detail.Steps[0].Status != "FAILED" || detail.Steps[1].Status != "SKIPPED" {
		t.Fatalf("steps = %#v", detail.Steps)
	}
}
