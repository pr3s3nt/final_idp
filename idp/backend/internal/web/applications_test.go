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

// fakeApps records calls and returns scripted results; it has no state
// between requests, like the real service.
type fakeApps struct {
	saveErr   error
	loadErr   error
	savedID   string
	savedBody service.ApplicationDefinitionDraft
	calls     int
}

func (f *fakeApps) ListApplications(context.Context) ([]service.ApplicationListItem, error) {
	return []service.ApplicationListItem{{ApplicationID: "a1", Name: "shop-app", LatestVersion: 2}}, nil
}

func (f *fakeApps) UpdateApplication(_ context.Context, id string) (*service.ApplicationDefinitionDraft, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	v := 2
	return &service.ApplicationDefinitionDraft{ApplicationID: id, BaseVersion: &v, Name: "shop-app"}, nil
}

func (f *fakeApps) SaveApplicationDefinition(_ context.Context, id string, d service.ApplicationDefinitionDraft) (*service.ApplicationDefinitionDraft, error) {
	f.calls++
	f.savedID, f.savedBody = id, d
	if f.saveErr != nil {
		return nil, f.saveErr
	}
	v := 3
	d.ApplicationID, d.BaseVersion = "a1", &v
	return &d, nil
}

func do(t *testing.T, apps *fakeApps, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	h := (&Server{Apps: apps, disableAuthenticationForTests: true}).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s %s: response is not a JSON object: %q", method, path, rec.Body.String())
	}
	return rec, out
}

func TestApplicationDefinitionAPI(t *testing.T) {
	apps := &fakeApps{}
	rec, out := do(t, apps, "GET", "/api/application-definitions", "")
	if rec.Code != 200 || out["applications"].([]any)[0].(map[string]any)["latestVersion"] != float64(2) {
		t.Fatalf("list: %d %v", rec.Code, out)
	}

	rec, out = do(t, apps, "GET", "/api/application-definitions/a1", "")
	if rec.Code != 200 || out["baseVersion"] != float64(2) || out["applicationId"] != "a1" {
		t.Fatalf("load: %d %v", rec.Code, out)
	}

	rec, out = do(t, apps, "POST", "/api/application-definitions", `{"name":"shop-app","workloads":[],"resources":[],"dependencies":[]}`)
	if rec.Code != 201 || apps.savedID != "" || out["baseVersion"] != float64(3) {
		t.Fatalf("create: %d %v (id %q)", rec.Code, out, apps.savedID)
	}

	rec, _ = do(t, apps, "POST", "/api/application-definitions/a1/versions", `{"baseVersion":2,"name":"shop-app"}`)
	if rec.Code != 201 || apps.savedID != "a1" || *apps.savedBody.BaseVersion != 2 {
		t.Fatalf("new version: %d (id %q)", rec.Code, apps.savedID)
	}
}

func TestApplicationDefinitionAPIErrors(t *testing.T) {
	notFound := &fakeApps{loadErr: domain.Reject(domain.CodeNotFound, "application %q not found", "x")}
	if rec, out := do(t, notFound, "GET", "/api/application-definitions/x", ""); rec.Code != 404 || problemCode(out) != "NOT_FOUND" {
		t.Fatalf("not found: %d %v", rec.Code, out)
	}

	conflict := &fakeApps{saveErr: domain.Reject(domain.CodeDraftConflict, "stale")}
	if rec, out := do(t, conflict, "POST", "/api/application-definitions/a1/versions", `{"baseVersion":1}`); rec.Code != 409 || problemCode(out) != "DRAFT_CONFLICT" {
		t.Fatalf("conflict: %d %v", rec.Code, out)
	}

	invalid := &fakeApps{saveErr: func() error {
		v := &domain.ValidationError{}
		v.AddField("workloads.w1.name", domain.CodeDuplicateName, "duplicate")
		return v
	}()}
	rec, out := do(t, invalid, "POST", "/api/application-definitions", `{"name":"x"}`)
	problem := out["problems"].([]any)[0].(map[string]any)
	if rec.Code != 422 || problem["code"] != "DUPLICATE_NAME" || problem["field"] != "workloads.w1.name" {
		t.Fatalf("validation: %d %v", rec.Code, out)
	}

	apps := &fakeApps{}
	for _, body := range []string{
		`{"name":"x","workloads":[{"id":"w","secrets":[{"id":"s","name":"DB_PASSWORD","value":"hunter2"}]}]}`,
		`{"name":"x","workloads":[{"id":"w","imageVersion":"v1"}]}`,
		`not json`,
		`{"name":"x"}{"name":"y"}`,
	} {
		if rec, out := do(t, apps, "POST", "/api/application-definitions", body); rec.Code != 422 || problemCode(out) != "INVALID_INPUT" {
			t.Errorf("%s: %d %v", body, rec.Code, out)
		}
	}
	if apps.calls != 0 {
		t.Fatalf("rejected requests reached the service %d times", apps.calls)
	}
}

func problemCode(out map[string]any) string {
	problems, _ := out["problems"].([]any)
	if len(problems) == 0 {
		return ""
	}
	return problems[0].(map[string]any)["code"].(string)
}
