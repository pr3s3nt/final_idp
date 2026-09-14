package configuration_test

import (
	"testing"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/configuration"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
)

func TestResolveResourceAndWorkloadOutputsWithoutSecretValues(t *testing.T) {
	snapshot := domain.DeploymentInputSnapshot{
		ApplicationDefinition: domain.ApplicationDefinition{ID: "app", Name: "notes", Workloads: []domain.Workload{
			{ID: "backend", Name: "backend", Port: 8080, ExposedOutputs: []domain.WorkloadOutputDefinition{{Name: "serviceUrl", Availability: domain.OutputPlanTime, Port: 8080}}},
			{ID: "frontend", Name: "frontend", Port: 8080},
		}},
		EnvironmentConfiguration: domain.EnvironmentConfiguration{Values: []domain.ConfigurationValue{
			{ID: "host", WorkloadID: "backend", Name: "DB_HOST", Source: domain.ValueResourceOutput, ResourceRequirementID: "db", ResourceOutputName: "host"},
			{ID: "url", WorkloadID: "frontend", Name: "BACKEND_URL", Source: domain.ValueWorkloadOutput, ReferencedWorkloadID: "backend", WorkloadOutputName: "serviceUrl"},
		}, Secrets: []domain.SecretReference{{ID: "password", WorkloadID: "backend", Name: "DB_PASSWORD", Target: "aws", Namespace: "demo", SecretName: "db-v1", UID: "uid-1", Key: "password"}}},
		RenderContext: domain.DeploymentContext{TargetID: "aws", Namespace: "demo", NamingPolicy: "mvp-v1"},
	}
	resolved, err := configuration.NewResolver().Resolve(snapshot, configuration.ResourceOutputs{"db": {"host": "postgres.demo.svc"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := resolved.Workloads["frontend"].Environment["BACKEND_URL"]; got != "http://notes-backend.demo.svc.cluster.local:8080" {
		t.Fatalf("BACKEND_URL = %q", got)
	}
	if got := resolved.Workloads["backend"].Environment["DB_HOST"]; got != "postgres.demo.svc" {
		t.Fatalf("DB_HOST = %q", got)
	}
	secret := resolved.Workloads["backend"].Secrets[0]
	if secret.SecretName != "db-v1" || secret.Key != "password" {
		t.Fatalf("secret = %#v", secret)
	}
}

func TestResolveAuroraCredentialAsReferenceOnly(t *testing.T) {
	snapshot := domain.DeploymentInputSnapshot{
		ApplicationDefinition: domain.ApplicationDefinition{ID: "app", Name: "notes", Workloads: []domain.Workload{{ID: "backend", Name: "backend"}}},
		EnvironmentConfiguration: domain.EnvironmentConfiguration{Secrets: []domain.SecretReference{{
			ID: "password", WorkloadID: "backend", Name: "DB_PASSWORD", Source: domain.SecretResource,
			Target: "aws", Namespace: "demo", SecretName: "notes-db", Key: "password",
			ResourceRequirementID: "db", ResourceOutputName: "credential_secret_arn", RemoteProperty: "password",
		}}},
		RenderContext: domain.DeploymentContext{TargetID: "aws", CloudProvider: "aws", Namespace: "demo"},
	}
	arn := "arn:aws:secretsmanager:ap-southeast-1:123456789012:secret:rds!cluster-demo"
	resolved, err := configuration.NewResolver().Resolve(snapshot, configuration.ResourceOutputs{"db": {"credential_secret_arn": arn}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Materializations) != 1 || resolved.Materializations[0].RemoteReference != arn {
		t.Fatalf("materializations = %#v", resolved.Materializations)
	}
	if got := resolved.Workloads["backend"].Secrets[0].SecretName; got != "notes-db" {
		t.Fatalf("secret name = %q", got)
	}
}
