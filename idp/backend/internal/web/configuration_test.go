package web

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"idp/internal/domain"
	"idp/internal/service"
)

// fakeConfigs records what the API passed on and returns scripted results. It
// keeps no state between requests, like the real service.
type fakeConfigs struct {
	saveErr      error
	stagedValue  string
	savedDraft   service.EnvironmentConfigurationDraft
	savedEnv     string
	outputsQuery [4]string
	calls        int
}

func (f *fakeConfigs) SelectEnvironment(_ context.Context, applicationID, environment string) (*service.ConfigurationRequirements, error) {
	return &service.ConfigurationRequirements{ApplicationID: applicationID, Environment: strings.ToUpper(environment),
		BaseApplicationDefinitionVersion: 2, BaseConfigurationRevision: "rev-1"}, nil
}

func (f *fakeConfigs) ResourceOutputs(_ context.Context, _, _, workloadID, resourceID, catalogVersion, target string) (*service.ResourceOutputsView, error) {
	f.outputsQuery = [4]string{workloadID, resourceID, catalogVersion, target}
	return &service.ResourceOutputsView{ResourceID: resourceID, ResourceName: "postgresql", DefinitionName: "postgres-k8s",
		Outputs: []service.OutputOption{{Name: "host"}, {Name: "password", Sensitive: true}}}, nil
}

func (f *fakeConfigs) WorkloadOutputs(_ context.Context, _, _, workloadID string) (*service.WorkloadOutputsView, error) {
	return &service.WorkloadOutputsView{WorkloadID: workloadID, WorkloadName: "backend",
		Outputs: []service.OutputOption{{Name: "endpoint"}}}, nil
}

func (f *fakeConfigs) StageSecret(_ context.Context, _, _, _, _, value string) (string, error) {
	f.stagedValue = value
	return "idpsecret://app/staging/secret", nil
}

func (f *fakeConfigs) SaveEnvironmentConfiguration(_ context.Context, _, environment string,
	d service.EnvironmentConfigurationDraft) (*service.EnvironmentConfigurationDraft, error) {
	f.calls++
	f.savedDraft, f.savedEnv = d, environment
	if f.saveErr != nil {
		return nil, f.saveErr
	}
	d.BaseConfigurationRevision = "rev-2"
	return &d, nil
}

func call(t *testing.T, configs *fakeConfigs, method, path, body string) (int, map[string]any) {
	t.Helper()
	h := (&Server{Configs: configs, disableAuthenticationForTests: true}).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s %s: response is not a JSON object: %q", method, path, rec.Body.String())
	}
	return rec.Code, out
}

const base = "/api/environment-configurations/a1/staging"

func TestSelectEnvironmentReturnsBothConcurrencyBases(t *testing.T) {
	status, body := call(t, &fakeConfigs{}, "GET", base, "")

	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if body["baseApplicationDefinitionVersion"] != float64(2) || body["baseConfigurationRevision"] != "rev-1" {
		t.Fatalf("both bases must reach the browser, got %v", body)
	}
}

func TestResourceOutputsPassTheSelectedCatalogVersionAndTarget(t *testing.T) {
	configs := &fakeConfigs{}
	status, body := call(t, configs, "GET", base+"/resources/r1/outputs?workloadId=w1&catalogVersion=2&target=kind-local", "")

	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if configs.outputsQuery != [4]string{"w1", "r1", "2", "kind-local"} {
		t.Fatalf("the query must reach the service unchanged, got %v", configs.outputsQuery)
	}
	if body["definitionName"] != "postgres-k8s" {
		t.Fatalf("the resolved definition must be named in the response, got %v", body)
	}
}

func TestWorkloadOutputsAreServed(t *testing.T) {
	status, body := call(t, &fakeConfigs{}, "GET", base+"/workloads/w2/outputs?workloadId=w3", "")

	if status != 200 || body["workloadName"] != "backend" {
		t.Fatalf("status = %d, body = %v", status, body)
	}
}

func TestStagingASecretReturnsOnlyAnOpaqueReference(t *testing.T) {
	configs := &fakeConfigs{}
	status, body := call(t, configs, "POST", base+"/secrets",
		`{"workloadId":"w1","definitionId":"d1","value":"hunter2"}`)

	if status != 201 {
		t.Fatalf("status = %d, want 201", status)
	}
	if configs.stagedValue != "hunter2" {
		t.Fatalf("the plaintext value must reach the Secret Store, got %q", configs.stagedValue)
	}
	if body["secretRef"] != "idpsecret://app/staging/secret" {
		t.Fatalf("the response must carry the opaque reference, got %v", body)
	}
	if strings.Contains(strings.ToLower(jsonOf(t, body)), "hunter2") {
		t.Fatalf("the response must not echo the plaintext secret: %v", body)
	}
}

func TestSaveSendsTheCompleteDraftIncludingTheCatalogSelection(t *testing.T) {
	configs := &fakeConfigs{}
	status, body := call(t, configs, "PUT", base, `{
		"applicationId":"a1","environment":"STAGING",
		"baseApplicationDefinitionVersion":2,"baseConfigurationRevision":"rev-1",
		"catalogVersion":"2","deploymentTarget":"kind-local",
		"variables":[{"workloadId":"w1","definitionId":"d1","name":"DB_HOST","source":"RESOURCE_OUTPUT","refId":"r1","outputName":"host"}],
		"secrets":[]}`)

	if status != 200 {
		t.Fatalf("status = %d, want 200: %v", status, body)
	}
	if configs.savedEnv != "staging" || configs.savedDraft.CatalogVersion != "2" || configs.savedDraft.DeploymentTarget != "kind-local" {
		t.Fatalf("the save must carry the environment and the catalog selection, got %q %+v", configs.savedEnv, configs.savedDraft)
	}
	if len(configs.savedDraft.Variables) != 1 || configs.savedDraft.Variables[0].OutputName != "host" {
		t.Fatalf("the complete draft must reach the service, got %+v", configs.savedDraft.Variables)
	}
	if body["baseConfigurationRevision"] != "rev-2" {
		t.Fatalf("the new revision must return to the browser, got %v", body)
	}
}

func TestSaveRejectsAnUnknownFieldSuchAsAPlaintextSecret(t *testing.T) {
	configs := &fakeConfigs{}
	status, _ := call(t, configs, "PUT", base, `{"applicationId":"a1","secretValue":"hunter2"}`)

	if status != 422 {
		t.Fatalf("status = %d, want 422", status)
	}
	if configs.calls != 0 {
		t.Fatal("a rejected request must not reach the service")
	}
}

func TestStaleDraftIsAConflict(t *testing.T) {
	configs := &fakeConfigs{saveErr: domain.Reject(domain.CodeDraftConflict, "the configuration changed")}
	status, body := call(t, configs, "PUT", base, `{"applicationId":"a1","baseApplicationDefinitionVersion":2}`)

	if status != 409 {
		t.Fatalf("status = %d, want 409: %v", status, body)
	}
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
