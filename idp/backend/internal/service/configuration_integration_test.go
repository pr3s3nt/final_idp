//go:build integration

package service_test

import (
	"context"
	"strings"
	"testing"

	"idp/internal/domain"
	"idp/internal/integration/secretstore"
	"idp/internal/service"
)

var uc02Tables = []string{"environment_configuration", "environment_variable", "configuration_value", "secret"}

func configService(t *testing.T, e *env) (*service.ConfigurationService, secretstore.Store) {
	t.Helper()
	secrets, err := secretstore.NewEncryptedFile(t.TempDir(), "integration-test-secret-key")
	if err != nil {
		t.Fatal(err)
	}
	return &service.ConfigurationService{Apps: e.repo.Apps, Configs: e.repo.Configs, Catalog: e.repo.Catalog, Secrets: secrets}, secrets
}

func (e *env) uc02Snapshot(t *testing.T) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range uc02Tables {
		out[table] = e.count(t, table)
	}
	return out
}

// draftOf builds the complete draft the Web UI would send: every variable and
// secret the latest version declares, prefilled with the stored binding when
// there is one and with a direct value or a staged secret reference otherwise.
func draftOf(t *testing.T, r *service.ConfigurationRequirements, secretRef, catalogVersion, target string) service.EnvironmentConfigurationDraft {
	t.Helper()
	d := service.EnvironmentConfigurationDraft{ApplicationID: r.ApplicationID, Environment: r.Environment,
		BaseApplicationDefinitionVersion: r.BaseApplicationDefinitionVersion, BaseConfigurationRevision: r.BaseConfigurationRevision,
		CatalogVersion: catalogVersion, DeploymentTarget: target}
	stored := map[string]service.BindingDraft{}
	if r.Configuration != nil {
		for _, b := range r.Configuration.Variables {
			stored["variable:"+b.WorkloadID+"/"+b.DefinitionID] = b
		}
		for _, b := range r.Configuration.Secrets {
			stored["secret:"+b.WorkloadID+"/"+b.DefinitionID] = b
		}
	}
	for _, w := range r.Workloads {
		for _, def := range w.Variables {
			if b, ok := stored["variable:"+w.ID+"/"+def.ID]; ok {
				d.Variables = append(d.Variables, b)
				continue
			}
			d.Variables = append(d.Variables, service.BindingDraft{WorkloadID: w.ID, DefinitionID: def.ID, Name: def.Name,
				Source: domain.SourceDirect, Value: "INFO"})
		}
		for _, def := range w.Secrets {
			if b, ok := stored["secret:"+w.ID+"/"+def.ID]; ok {
				d.Secrets = append(d.Secrets, b)
				continue
			}
			if secretRef == "" {
				t.Fatalf("secret %s.%s needs a staged reference", w.Name, def.Name)
			}
			d.Secrets = append(d.Secrets, service.BindingDraft{WorkloadID: w.ID, DefinitionID: def.ID, Name: def.Name,
				Source: domain.SourceSecretRef, SecretRef: secretRef})
		}
	}
	return d
}

func TestSelectEnvironmentLoadsRequirementsBasesAndCatalogChoices(t *testing.T) {
	e := setup(t)
	configs, _ := configService(t, e)

	view, err := configs.SelectEnvironment(context.Background(), "shop-app", "staging")
	if err != nil {
		t.Fatal(err)
	}

	if view.Environment != string(domain.Staging) || view.BaseApplicationDefinitionVersion < 1 {
		t.Fatalf("environment and base version must be loaded, got %+v", view)
	}
	if len(view.Workloads) == 0 || len(view.Resources) == 0 {
		t.Fatal("the requirements of UC-01 must be listed by workload and resource")
	}
	if len(view.CatalogVersions) == 0 || len(view.Targets) == 0 {
		t.Fatal("the Developer must be able to choose a catalog version and a deployment target (ADR-020)")
	}
	if view.Configuration == nil || view.BaseConfigurationRevision == "" {
		t.Fatal("the fixture configuration and its opaque revision must be returned")
	}
	// Every workload must know what it may reference at all.
	for _, w := range view.Workloads {
		if w.Name == "frontend" && len(w.DependsOnWorkloads) == 0 {
			t.Fatal("frontend depends on backend, so its workload dependencies must be listed")
		}
	}
}

func TestSaveWithCurrentBasesWritesAndAdvancesTheRevision(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	configs, _ := configService(t, e)

	view, err := configs.SelectEnvironment(ctx, "shop-app", "staging")
	if err != nil {
		t.Fatal(err)
	}
	draft := draftOf(t, view, "", "", "kind-local")
	for i := range draft.Variables {
		if draft.Variables[i].Source == domain.SourceDirect {
			draft.Variables[i].Value = "DEBUG"
		}
	}

	saved, err := configs.SaveEnvironmentConfiguration(ctx, "shop-app", "staging", draft)
	if err != nil {
		t.Fatalf("saving the loaded configuration again must succeed: %v", err)
	}
	if saved.BaseConfigurationRevision == "" || saved.BaseConfigurationRevision == view.BaseConfigurationRevision {
		t.Fatalf("the revision must advance, was %q now %q", view.BaseConfigurationRevision, saved.BaseConfigurationRevision)
	}
	if len(saved.Variables) != len(draft.Variables) || len(saved.Secrets) != len(draft.Secrets) {
		t.Fatalf("the stored binding set must equal the submitted one, got %d variables and %d secrets",
			len(saved.Variables), len(saved.Secrets))
	}
	var rows int
	if err := e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM environment_configuration
		WHERE application_id = $1 AND environment = 'STAGING'`, view.ApplicationID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("saving must keep exactly one configuration per application and environment, found %d", rows)
	}
}

func TestSaveWithAStaleRevisionIsRejectedAndWritesNothing(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	configs, _ := configService(t, e)

	view, err := configs.SelectEnvironment(ctx, "shop-app", "staging")
	if err != nil {
		t.Fatal(err)
	}
	stale := draftOf(t, view, "", "", "kind-local")
	if _, err := configs.SaveEnvironmentConfiguration(ctx, "shop-app", "staging", draftOf(t, view, "", "", "kind-local")); err != nil {
		t.Fatal(err)
	}
	before := e.uc02Snapshot(t)

	_, err = configs.SaveEnvironmentConfiguration(ctx, "shop-app", "staging", stale)

	if !domain.HasCode(err, domain.CodeDraftConflict) {
		t.Fatalf("a second save from the same base must be DRAFT_CONFLICT, got %v", err)
	}
	assertUnchanged(t, before, e.uc02Snapshot(t))
}

func TestSaveWithAStaleApplicationVersionIsRejected(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	configs, _ := configService(t, e)

	view, err := configs.SelectEnvironment(ctx, "shop-app", "staging")
	if err != nil {
		t.Fatal(err)
	}
	draft := draftOf(t, view, "", "", "kind-local")
	draft.BaseApplicationDefinitionVersion = view.BaseApplicationDefinitionVersion - 1
	before := e.uc02Snapshot(t)

	_, err = configs.SaveEnvironmentConfiguration(ctx, "shop-app", "staging", draft)

	if !domain.HasCode(err, domain.CodeDraftConflict) {
		t.Fatalf("a draft based on an older Application Definition version must be DRAFT_CONFLICT, got %v", err)
	}
	assertUnchanged(t, before, e.uc02Snapshot(t))
}

func TestInvalidOutputReferenceIsRejectedAndWritesNothing(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	configs, _ := configService(t, e)

	view, err := configs.SelectEnvironment(ctx, "shop-app", "staging")
	if err != nil {
		t.Fatal(err)
	}
	draft := draftOf(t, view, "", "", "kind-local")
	bound := false
	for i := range draft.Variables {
		if draft.Variables[i].Source == domain.SourceResourceOutput {
			draft.Variables[i].OutputName = "reader_host" // no kind-local definition exposes it
			bound = true
			break
		}
	}
	if !bound {
		t.Skip("the fixture configuration binds no resource output")
	}
	before := e.uc02Snapshot(t)

	_, err = configs.SaveEnvironmentConfiguration(ctx, "shop-app", "staging", draft)

	if !domain.HasCode(err, domain.CodeInvalidOutputReference) {
		t.Fatalf("an output the resolved definition does not expose must be INVALID_OUTPUT_REFERENCE, got %v", err)
	}
	assertUnchanged(t, before, e.uc02Snapshot(t))
}

func TestStagedSecretKeepsPlaintextOutOfTheDatabase(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	configs, secrets := configService(t, e)

	view, err := configs.SelectEnvironment(ctx, "shop-app", "staging")
	if err != nil {
		t.Fatal(err)
	}
	var workloadID, definitionID string
	for _, w := range view.Workloads {
		if len(w.Secrets) > 0 {
			workloadID, definitionID = w.ID, w.Secrets[0].ID
			break
		}
	}
	if workloadID == "" {
		t.Skip("no workload of the fixture declares a secret")
	}

	ref, err := configs.StageSecret(ctx, "shop-app", "staging", workloadID, definitionID, "hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(ref, "idpsecret://") {
		t.Fatalf("staging must return an opaque reference, got %q", ref)
	}
	value, err := secrets.Get(ctx, ref)
	if err != nil || string(value) != "hunter2" {
		t.Fatalf("the Secret Store must hold the value: %q, %v", value, err)
	}

	draft := draftOf(t, view, "", "", "kind-local")
	for i := range draft.Secrets {
		if draft.Secrets[i].WorkloadID == workloadID && draft.Secrets[i].DefinitionID == definitionID {
			draft.Secrets[i] = service.BindingDraft{WorkloadID: workloadID, DefinitionID: definitionID,
				Name: draft.Secrets[i].Name, Source: domain.SourceSecretRef, SecretRef: ref}
		}
	}
	if _, err := configs.SaveEnvironmentConfiguration(ctx, "shop-app", "staging", draft); err != nil {
		t.Fatalf("saving a staged secret reference must succeed: %v", err)
	}

	var stored int
	if err := e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM secret WHERE secret_ref LIKE '%hunter2%'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 0 {
		t.Fatal("the plaintext secret must never reach the configuration database")
	}
	if err := e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM secret WHERE secret_ref = $1`, ref).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 1 {
		t.Fatalf("only the opaque reference must be persisted, found %d rows", stored)
	}
}

func TestResourceOutputsComeFromTheDefinitionTheTargetResolves(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	configs, _ := configService(t, e)

	view, err := configs.SelectEnvironment(ctx, "shop-app", "staging")
	if err != nil {
		t.Fatal(err)
	}
	var workloadID, resourceID string
	for _, w := range view.Workloads {
		if len(w.DependsOnResources) > 0 {
			workloadID, resourceID = w.ID, w.DependsOnResources[0]
			break
		}
	}
	if workloadID == "" {
		t.Skip("no workload of the fixture depends on a resource")
	}

	outputs, err := configs.ResourceOutputs(ctx, "shop-app", "staging", workloadID, resourceID, "", "kind-local")
	if err != nil {
		t.Fatal(err)
	}
	if outputs.DefinitionName == "" || len(outputs.Outputs) == 0 {
		t.Fatalf("the resolved definition and its outputs must be returned, got %+v", outputs)
	}

	// A workload that does not depend on the resource may not query it.
	var other string
	for _, w := range view.Workloads {
		if w.ID != workloadID && !containsString(w.DependsOnResources, resourceID) {
			other = w.ID
			break
		}
	}
	if other == "" {
		return
	}
	if _, err := configs.ResourceOutputs(ctx, "shop-app", "staging", other, resourceID, "", "kind-local"); err == nil {
		t.Fatal("a workload that does not depend on the resource must not receive its outputs")
	}
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
