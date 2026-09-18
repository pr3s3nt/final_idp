package configvalidator_test

import (
	"testing"

	"idp/internal/domain"
	"idp/internal/domain/configvalidator"
	"idp/internal/domain/domaintest"
	"idp/internal/domain/resourceresolver"
)

// definitions resolves every Resource Requirement of the version the way UC-02
// does: one definition per requirement, chosen by catalog version and target.
func definitions(t *testing.T, v *domain.ApplicationVersion, target string) map[string]*domain.ResourceDefinition {
	t.Helper()
	r := domaintest.Catalog()
	context := domain.DeploymentContext{Target: target}
	for _, o := range r.SupportedTargets() {
		if o.Target == target {
			context = domain.DeploymentContext{Target: o.Target, CloudProvider: o.CloudProvider, Region: o.Region}
		}
	}
	scope := resourceresolver.Scope{ApplicationID: v.ApplicationID, ApplicationName: v.ApplicationName, Environment: domain.Staging, Context: context}
	out := map[string]*domain.ResourceDefinition{}
	for _, req := range v.Resources {
		resolution, problem := r.Resolve(req.ID, req.ResourceType, scope)
		if problem == nil {
			out[req.ID] = resolution.Definition
		}
	}
	return out
}

func input(t *testing.T, cfg *domain.EnvironmentConfiguration) configvalidator.Input {
	t.Helper()
	v := domaintest.ShopV1()
	return configvalidator.Input{Version: v, Configuration: cfg, Definitions: definitions(t, v, "kind-local")}
}

func TestCompleteConfigurationIsValid(t *testing.T) {
	if err := configvalidator.Validate(input(t, domaintest.Config())); err != nil {
		t.Fatalf("the complete configuration of ShopV1 must be valid: %v", err)
	}
}

func TestMissingRequiredBindingIsRejected(t *testing.T) {
	cfg := domaintest.Config()
	cfg.Variables = cfg.Variables[1:] // drop backend.DB_HOST

	err := configvalidator.Validate(input(t, cfg))
	if !domain.HasCode(err, domain.CodeMissingConfiguration) {
		t.Fatalf("a missing required variable must be MISSING_REQUIRED_CONFIGURATION, got %v", err)
	}
}

func TestOutputOfAComponentTheWorkloadDoesNotDependOnIsRejected(t *testing.T) {
	cfg := domaintest.Config()
	// frontend depends on backend only, never on postgresql.
	cfg.Variables[2] = domain.ConfiguredValue{WorkloadID: domaintest.Frontend, DefinitionID: cfg.Variables[2].DefinitionID,
		Name: "BACKEND_URL", Source: domain.SourceResourceOutput, RefID: domaintest.Postgres, OutputName: "host"}

	err := configvalidator.Validate(input(t, cfg))
	if !domain.HasCode(err, domain.CodeConfigurationMismatch) {
		t.Fatalf("binding an output of a component the workload does not depend on must be CONFIGURATION_VERSION_MISMATCH, got %v", err)
	}
}

func TestOutputTheResolvedDefinitionDoesNotExposeIsRejected(t *testing.T) {
	cfg := domaintest.Config()
	// postgres-k8s exposes host, port, database and username - not reader_host.
	cfg.Variables[0].OutputName = "reader_host"

	err := configvalidator.Validate(input(t, cfg))
	if !domain.HasCode(err, domain.CodeInvalidOutputReference) {
		t.Fatalf("an output the resolved definition does not expose must be INVALID_OUTPUT_REFERENCE, got %v", err)
	}
}

func TestSecretMustUseASensitiveOutput(t *testing.T) {
	cfg := domaintest.Config()
	cfg.Secrets[0].OutputName = "host" // exposed, but not sensitive

	err := configvalidator.Validate(input(t, cfg))
	if !domain.HasCode(err, domain.CodeInvalidOutputReference) {
		t.Fatalf("a secret bound to a non-sensitive output must be INVALID_OUTPUT_REFERENCE, got %v", err)
	}
}

func TestSecretCannotBeADirectPlaintextValue(t *testing.T) {
	cfg := domaintest.Config()
	cfg.Secrets[0] = domain.ConfiguredValue{WorkloadID: domaintest.Backend, DefinitionID: cfg.Secrets[0].DefinitionID,
		Name: "DB_PASSWORD", Source: domain.SourceDirect, DirectValue: "hunter2"}

	err := configvalidator.Validate(input(t, cfg))
	if !domain.HasCode(err, domain.CodeInvalidOutputReference) {
		t.Fatalf("a plaintext secret must be INVALID_OUTPUT_REFERENCE, got %v", err)
	}
}

func TestVariableCannotUseASensitiveOutput(t *testing.T) {
	cfg := domaintest.Config()
	cfg.Variables[0].OutputName = "password"

	err := configvalidator.Validate(input(t, cfg))
	if !domain.HasCode(err, domain.CodeInvalidOutputReference) {
		t.Fatalf("a variable bound to a sensitive output must be INVALID_OUTPUT_REFERENCE, got %v", err)
	}
}

func TestUnresolvedResourceDefinitionIsRejected(t *testing.T) {
	v := domaintest.ShopV1()
	cfg := domaintest.Config()
	// The selected catalog version and target resolve no definition for redis,
	// so the service leaves it out and worker.REDIS_HOST cannot be validated.
	resolved := definitions(t, v, "kind-local")
	delete(resolved, domaintest.Redis)

	err := configvalidator.Validate(configvalidator.Input{Version: v, Configuration: cfg, Definitions: resolved})
	if !domain.HasCode(err, domain.CodeNoResourceDefinition) {
		t.Fatalf("a requirement without a definition for the selected target must be NO_MATCHING_RESOURCE_DEFINITION, got %v", err)
	}
}

func TestTheSelectedTargetDecidesWhichDefinitionIsChecked(t *testing.T) {
	v := domaintest.ShopV1()
	kind, aws := definitions(t, v, "kind-local"), definitions(t, v, "aws")
	if kind[domaintest.Postgres].Name != "postgres-k8s" {
		t.Fatalf("kind-local must resolve postgresql to postgres-k8s, got %s", kind[domaintest.Postgres].Name)
	}
	if aws[domaintest.Postgres].Name != "aurora-postgresql" {
		t.Fatalf("aws must resolve postgresql to aurora-postgresql, got %s", aws[domaintest.Postgres].Name)
	}
}

func TestEveryProblemIsReportedInOnePass(t *testing.T) {
	cfg := domaintest.Config()
	cfg.Variables[0].OutputName = "reader_host"
	cfg.Secrets[0].OutputName = "host"

	err := configvalidator.Validate(input(t, cfg))
	var v *domain.ValidationError
	if !asValidation(err, &v) || len(v.Problems) < 2 {
		t.Fatalf("both problems must be reported together, got %v", err)
	}
}

func asValidation(err error, target **domain.ValidationError) bool {
	v, ok := err.(*domain.ValidationError)
	if ok {
		*target = v
	}
	return ok
}
