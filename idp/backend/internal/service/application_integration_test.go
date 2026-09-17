//go:build integration

package service_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"idp/internal/domain"
	"idp/internal/persistence"
	"idp/internal/service"
)

func appService(e *env) *service.ApplicationService {
	return &service.ApplicationService{Apps: e.repo.Apps, Specs: &persistence.SpecificationRepository{DB: e.db}}
}

var uc01Tables = []string{"application_definition", "application_definition_version", "application_component", "workload",
	"resource_requirement", "environment_variable_definition", "secret_definition", "dependency", "application_specification"}

func (e *env) snapshot(t *testing.T) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range uc01Tables {
		out[table] = e.count(t, table)
	}
	return out
}

func assertUnchanged(t *testing.T, before, after map[string]int) {
	t.Helper()
	for table, n := range before {
		if after[table] != n {
			t.Errorf("%s changed from %d to %d rows", table, n, after[table])
		}
	}
}

func newShopDraft(name string) service.ApplicationDefinitionDraft {
	port := 8080
	backend, frontend, db := uuid.NewString(), uuid.NewString(), uuid.NewString()
	return service.ApplicationDefinitionDraft{
		Name: name, Description: "UC-01 example",
		Workloads: []service.WorkloadDraft{
			{ID: backend, Name: "backend", Type: "Backend Service", ImageRepository: "registry.company.local/shop-backend", Port: &port,
				Outputs:   []string{"endpoint"},
				Variables: []service.ConfigRequirementDraft{{ID: uuid.NewString(), Name: "DB_HOST", Required: true}},
				Secrets:   []service.ConfigRequirementDraft{{ID: uuid.NewString(), Name: "DB_PASSWORD", Required: true}}},
			{ID: frontend, Name: "frontend", Type: "Frontend", ImageRepository: "registry.company.local/shop-frontend"},
		},
		Resources: []service.ResourceDraft{{ID: db, Name: "postgresql", Type: "PostgreSQL"}},
		Dependencies: []service.DependencyDraft{
			{ID: "d1", SourceID: frontend, TargetID: backend},
			{ID: "d2", SourceID: backend, TargetID: db},
		},
	}
}

func TestUC01CreateApplication(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	apps := appService(e)
	draft := newShopDraft("orders-app")

	saved, err := apps.SaveApplicationDefinition(ctx, "", draft)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if *saved.BaseVersion != 1 || saved.Name != "orders-app" || saved.Description != "UC-01 example" {
		t.Fatalf("saved: %+v", saved)
	}
	backend := findWorkload(t, saved, "backend")
	if backend.ID != draft.Workloads[0].ID || backend.Secrets[0].ID != draft.Workloads[0].Secrets[0].ID {
		t.Fatalf("submitted IDs of new components were not kept: %+v", backend)
	}
	if len(saved.Dependencies) != 2 {
		t.Fatalf("dependencies: %+v", saved.Dependencies)
	}

	loaded, err := apps.UpdateApplication(ctx, saved.ApplicationID)
	if err != nil || *loaded.BaseVersion != 1 || len(loaded.Workloads) != 2 || len(loaded.Resources) != 1 {
		t.Fatalf("load for edit: %v %+v", err, loaded)
	}

	format, content, err := (&persistence.SpecificationRepository{DB: e.db}).Find(ctx, versionID(t, e, saved.ApplicationID, 1))
	if err != nil || format != "score-yaml" || !strings.Contains(content, "idp.dev/secrets: DB_PASSWORD") {
		t.Fatalf("specification: %v %s\n%s", err, format, content)
	}

	list, err := apps.ListApplications(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range list {
		found = found || (a.ApplicationID == saved.ApplicationID && a.LatestVersion == 1)
	}
	if !found {
		t.Fatalf("created application missing from list: %+v", list)
	}
}

func TestUC01EditCreatesNewVersionWithStableIDs(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	apps := appService(e)
	v1, err := apps.SaveApplicationDefinition(ctx, "", newShopDraft("orders-app"))
	if err != nil {
		t.Fatal(err)
	}
	backendID := findWorkload(t, v1, "backend").ID

	edit, err := apps.UpdateApplication(ctx, v1.ApplicationID)
	if err != nil {
		t.Fatal(err)
	}
	// Rename backend, remove the resource and its dependency, add a variable.
	for i := range edit.Workloads {
		if edit.Workloads[i].ID == backendID {
			edit.Workloads[i].Name = "api"
			edit.Workloads[i].Variables = append(edit.Workloads[i].Variables, service.ConfigRequirementDraft{ID: uuid.NewString(), Name: "LOG_LEVEL", Required: false})
		}
	}
	edit.Resources = nil
	var deps []service.DependencyDraft
	for _, d := range edit.Dependencies {
		if d.SourceID != backendID {
			deps = append(deps, d)
		}
	}
	edit.Dependencies = deps
	edit.Name = "orders"

	v2, err := apps.SaveApplicationDefinition(ctx, v1.ApplicationID, *edit)
	if err != nil {
		t.Fatalf("save v2: %v", err)
	}
	if *v2.BaseVersion != 2 || v2.ApplicationID != v1.ApplicationID || v2.Name != "orders" {
		t.Fatalf("v2: %+v", v2)
	}
	api := findWorkload(t, v2, "api")
	if api.ID != backendID || len(api.Variables) != 2 || len(v2.Resources) != 0 || len(v2.Dependencies) != 1 {
		t.Fatalf("renamed workload lost its ID or edits: %+v", v2)
	}

	old, err := e.repo.Apps.FindVersion(ctx, v1.ApplicationID, "1")
	if err != nil {
		t.Fatal(err)
	}
	if old.Workload(backendID).Name != "backend" || len(old.Resources) != 1 || len(old.Dependencies) != 2 {
		t.Fatalf("version 1 was modified: %+v", old)
	}
	if e.count(t, "application_specification") != 2 {
		t.Fatal("each version needs its own specification")
	}
}

func TestUC01InvalidDraftWritesNothing(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	apps := appService(e)
	v1, err := apps.SaveApplicationDefinition(ctx, "", newShopDraft("orders-app"))
	if err != nil {
		t.Fatal(err)
	}
	before := e.snapshot(t)

	cases := map[string]func(d *service.ApplicationDefinitionDraft){
		"duplicate name": func(d *service.ApplicationDefinitionDraft) { d.Resources[0].Name = "backend" },
		"missing reference": func(d *service.ApplicationDefinitionDraft) {
			d.Dependencies[1].TargetID = uuid.NewString()
		},
		"cycle": func(d *service.ApplicationDefinitionDraft) {
			d.Dependencies = append(d.Dependencies, service.DependencyDraft{SourceID: d.Dependencies[0].TargetID, TargetID: d.Dependencies[0].SourceID})
		},
		"image tag": func(d *service.ApplicationDefinitionDraft) { d.Workloads[0].ImageRepository += ":v1" },
	}
	for name, mutate := range cases {
		edit, err := apps.UpdateApplication(ctx, v1.ApplicationID)
		if err != nil {
			t.Fatal(err)
		}
		mutate(edit)
		if _, err := apps.SaveApplicationDefinition(ctx, v1.ApplicationID, *edit); err == nil {
			t.Errorf("%s: invalid draft saved", name)
		}
	}
	assertUnchanged(t, before, e.snapshot(t))
}

func TestUC01StaleBaseVersionIsDraftConflict(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	apps := appService(e)
	v1, err := apps.SaveApplicationDefinition(ctx, "", newShopDraft("orders-app"))
	if err != nil {
		t.Fatal(err)
	}
	first, _ := apps.UpdateApplication(ctx, v1.ApplicationID)
	second, _ := apps.UpdateApplication(ctx, v1.ApplicationID)
	first.Description = "saved first"
	if _, err := apps.SaveApplicationDefinition(ctx, v1.ApplicationID, *first); err != nil {
		t.Fatal(err)
	}
	before := e.snapshot(t)
	second.Description = "saved second"
	_, err = apps.SaveApplicationDefinition(ctx, v1.ApplicationID, *second)
	if !domain.HasCode(err, domain.CodeDraftConflict) {
		t.Fatalf("stale draft: want DRAFT_CONFLICT, got %v", err)
	}
	assertUnchanged(t, before, e.snapshot(t))
	latest, _ := apps.UpdateApplication(ctx, v1.ApplicationID)
	if latest.Description != "saved first" || *latest.BaseVersion != 2 {
		t.Fatalf("stale draft changed the application: %+v", latest)
	}
}

func TestUC01ConcurrentSavesFromSameBase(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	apps := appService(e)
	v1, err := apps.SaveApplicationDefinition(ctx, "", newShopDraft("orders-app"))
	if err != nil {
		t.Fatal(err)
	}
	const writers = 8
	var wg sync.WaitGroup
	errs := make([]error, writers)
	for i := 0; i < writers; i++ {
		draft, err := apps.UpdateApplication(ctx, v1.ApplicationID)
		if err != nil {
			t.Fatal(err)
		}
		draft.Description = uuid.NewString()
		wg.Add(1)
		go func(i int, d service.ApplicationDefinitionDraft) {
			defer wg.Done()
			_, errs[i] = apps.SaveApplicationDefinition(ctx, v1.ApplicationID, d)
		}(i, *draft)
	}
	wg.Wait()
	wins := 0
	for _, err := range errs {
		switch {
		case err == nil:
			wins++
		case !domain.HasCode(err, domain.CodeDraftConflict):
			t.Errorf("loser failed with %v, want DRAFT_CONFLICT", err)
		}
	}
	if wins != 1 {
		t.Fatalf("%d saves from the same base succeeded, want 1", wins)
	}
	var versions int
	e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM application_definition_version WHERE application_id = $1`, v1.ApplicationID).Scan(&versions)
	if versions != 2 {
		t.Fatalf("versions: %d, want 2", versions)
	}
}

func TestUC01RejectsForeignComponentIDAndTakenName(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	apps := appService(e)
	shop, err := e.repo.Apps.FindApplication(ctx, "shop-app")
	if err != nil {
		t.Fatal(err)
	}
	shopLatest, err := apps.UpdateApplication(ctx, shop.ID)
	if err != nil {
		t.Fatal(err)
	}
	before := e.snapshot(t)

	draft := newShopDraft("orders-app")
	draft.Workloads[1].ID = shopLatest.Workloads[0].ID
	draft.Dependencies[0].SourceID = draft.Workloads[1].ID
	if _, err := apps.SaveApplicationDefinition(ctx, "", draft); !domain.HasCode(err, domain.CodeInvalidComponentID) {
		t.Fatalf("foreign component ID: want INVALID_COMPONENT_ID, got %v", err)
	}
	if _, err := apps.SaveApplicationDefinition(ctx, "", newShopDraft("shop-app")); !domain.HasCode(err, domain.CodeApplicationNameTaken) {
		t.Fatalf("taken name: want APPLICATION_NAME_TAKEN, got %v", err)
	}
	stale := *shopLatest
	stale.ApplicationID = ""
	if _, err := apps.SaveApplicationDefinition(ctx, uuid.NewString(), stale); !domain.HasCode(err, domain.CodeNotFound) {
		t.Fatalf("unknown application: want NOT_FOUND, got %v", err)
	}
	assertUnchanged(t, before, e.snapshot(t))
}

func findWorkload(t *testing.T, d *service.ApplicationDefinitionDraft, name string) service.WorkloadDraft {
	t.Helper()
	for _, w := range d.Workloads {
		if w.Name == name {
			return w
		}
	}
	t.Fatalf("workload %q not in %+v", name, d.Workloads)
	return service.WorkloadDraft{}
}

func versionID(t *testing.T, e *env, applicationID string, number int) string {
	t.Helper()
	var id string
	if err := e.db.Pool.QueryRow(context.Background(), `SELECT application_definition_version_id FROM application_definition_version
		WHERE application_id = $1 AND version_number = $2`, applicationID, number).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
