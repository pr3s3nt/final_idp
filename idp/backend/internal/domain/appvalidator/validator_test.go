package appvalidator

import (
	"strings"
	"testing"

	"idp/internal/domain"
)

const (
	backendID  = "11111111-1111-4111-8111-111111111111"
	frontendID = "22222222-2222-4222-8222-222222222222"
	dbID       = "33333333-3333-4333-8333-333333333333"
	logLevelID = "44444444-4444-4444-8444-444444444444"
	dbPassID   = "55555555-5555-4555-8555-555555555555"
)

func port(p int) *int { return &p }

// shop is the UC-01 specification example: frontend → backend → postgresql.
func shop() Definition {
	return Definition{
		Name: "shop-app",
		Workloads: []domain.Workload{
			{ID: backendID, Name: "backend", Type: "Backend Service", ImageRepository: "registry.company.local/shop-backend",
				Port: port(8080), ExposedOutputs: []string{"endpoint"},
				Variables: []domain.ConfigDefinition{{ID: logLevelID, Name: "LOG_LEVEL", Required: true}},
				Secrets:   []domain.ConfigDefinition{{ID: dbPassID, Name: "DB_PASSWORD", Required: true}}},
			{ID: frontendID, Name: "frontend", Type: "Frontend", ImageRepository: "registry.company.local/shop-frontend", Port: port(3000)},
		},
		Resources: []domain.ResourceRequirement{{ID: dbID, Name: "postgresql", ResourceType: "PostgreSQL"}},
		Dependencies: []Dependency{
			{Key: "d1", SourceID: frontendID, TargetID: backendID},
			{Key: "d2", SourceID: backendID, TargetID: dbID},
		},
	}
}

func TestValidDefinitionResolvesDependencyTargets(t *testing.T) {
	deps, err := Validate(shop())
	if err != nil {
		t.Fatalf("valid definition rejected: %v", err)
	}
	if len(deps) != 2 || deps[0].TargetType != domain.TargetWorkload || deps[1].TargetType != domain.TargetResource {
		t.Fatalf("unexpected dependencies: %+v", deps)
	}
}

func TestRejections(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(d *Definition)
		code   string
		field  string
	}{
		{"missing application name", func(d *Definition) { d.Name = "" }, domain.CodeMissingRequiredField, "name"},
		{"invalid application name", func(d *Definition) { d.Name = "Shop App" }, domain.CodeInvalidName, "name"},
		{"no workload", func(d *Definition) { d.Workloads = nil; d.Dependencies = nil }, domain.CodeNoWorkload, "workloads"},
		{"invalid workload name", func(d *Definition) { d.Workloads[0].Name = "api.v2" }, domain.CodeInvalidName, "workloads." + backendID + ".name"},
		{"workload name too long", func(d *Definition) { d.Workloads[0].Name = strings.Repeat("a", 64) }, domain.CodeInvalidName, "workloads." + backendID + ".name"},
		{"duplicate workload name", func(d *Definition) { d.Workloads[1].Name = "backend" }, domain.CodeDuplicateName, "workloads." + frontendID + ".name"},
		{"workload and resource share a name", func(d *Definition) { d.Resources[0].Name = "backend" }, domain.CodeDuplicateName, "resources." + dbID + ".name"},
		{"missing workload type", func(d *Definition) { d.Workloads[0].Type = " " }, domain.CodeMissingRequiredField, "workloads." + backendID + ".type"},
		{"missing image repository", func(d *Definition) { d.Workloads[0].ImageRepository = "" }, domain.CodeMissingRequiredField, "workloads." + backendID + ".imageRepository"},
		{"image repository with tag", func(d *Definition) { d.Workloads[0].ImageRepository = "registry.company.local/shop-backend:v1.4.2" }, domain.CodeInvalidImageRepository, "workloads." + backendID + ".imageRepository"},
		{"image repository with digest", func(d *Definition) { d.Workloads[0].ImageRepository = "shop@sha256:abc" }, domain.CodeInvalidImageRepository, "workloads." + backendID + ".imageRepository"},
		{"invalid image repository", func(d *Definition) { d.Workloads[0].ImageRepository = "Registry/Shop Backend" }, domain.CodeInvalidImageRepository, "workloads." + backendID + ".imageRepository"},
		{"port zero", func(d *Definition) { d.Workloads[0].Port = port(0) }, domain.CodeInvalidPort, "workloads." + backendID + ".port"},
		{"port too large", func(d *Definition) { d.Workloads[0].Port = port(65536) }, domain.CodeInvalidPort, "workloads." + backendID + ".port"},
		{"invalid output name", func(d *Definition) { d.Workloads[0].ExposedOutputs = []string{"End.Point"} }, domain.CodeInvalidName, "workloads." + backendID + ".outputs.0"},
		{"invalid variable name", func(d *Definition) { d.Workloads[0].Variables[0].Name = "LOG-LEVEL" }, domain.CodeInvalidName, "workloads." + backendID + ".variables." + logLevelID + ".name"},
		{"variable name starts with digit", func(d *Definition) { d.Workloads[0].Variables[0].Name = "1KEY" }, domain.CodeInvalidName, "workloads." + backendID + ".variables." + logLevelID + ".name"},
		{"variable and secret share a name", func(d *Definition) { d.Workloads[0].Secrets[0].Name = "LOG_LEVEL" }, domain.CodeDuplicateName, "workloads." + backendID + ".secrets." + dbPassID + ".name"},
		{"component without ID", func(d *Definition) { d.Resources[0].ID = ""; d.Dependencies = d.Dependencies[:1] }, domain.CodeInvalidComponentID, "resources..id"},
		{"component ID not a UUID", func(d *Definition) { d.Workloads[0].Secrets[0].ID = "db-pass" }, domain.CodeInvalidComponentID, "workloads." + backendID + ".secrets.db-pass.id"},
		{"component ID used twice", func(d *Definition) { d.Workloads[0].Secrets[0].ID = logLevelID }, domain.CodeInvalidComponentID, "workloads." + backendID + ".secrets." + logLevelID + ".id"},
		{"dependency target missing", func(d *Definition) { d.Dependencies[1].TargetID = "99999999-9999-4999-8999-999999999999" }, domain.CodeDependencyUnresolved, "dependencies.d2"},
		{"dependency source is a resource", func(d *Definition) { d.Dependencies[1] = Dependency{Key: "d2", SourceID: dbID, TargetID: backendID} }, domain.CodeInvalidDependency, "dependencies.d2"},
		{"dependency on itself", func(d *Definition) { d.Dependencies[1].TargetID = backendID }, domain.CodeDependencyCycle, "dependencies.d2"},
		{"duplicate dependency", func(d *Definition) {
			d.Dependencies = append(d.Dependencies, Dependency{Key: "d3", SourceID: frontendID, TargetID: backendID})
		}, domain.CodeDuplicateDependency, "dependencies.d3"},
		{"dependency cycle", func(d *Definition) {
			d.Dependencies = append(d.Dependencies, Dependency{Key: "d3", SourceID: backendID, TargetID: frontendID})
		}, domain.CodeDependencyCycle, "dependencies"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := shop()
			c.mutate(&d)
			_, err := Validate(d)
			if !hasProblem(err, c.code, c.field) {
				t.Fatalf("want %s at %q, got %v", c.code, c.field, err)
			}
		})
	}
}

func TestCycleMessageNamesWorkloads(t *testing.T) {
	d := shop()
	d.Dependencies = append(d.Dependencies, Dependency{SourceID: backendID, TargetID: frontendID})
	_, err := Validate(d)
	if err == nil || !strings.Contains(err.Error(), "frontend → backend → frontend") && !strings.Contains(err.Error(), "backend → frontend → backend") {
		t.Fatalf("cycle not named: %v", err)
	}
}

func TestAcceptsRegistryWithPortAndNestedPath(t *testing.T) {
	for _, repo := range []string{"localhost:5055/shop-backend", "123456789012.dkr.ecr.ap-southeast-1.amazonaws.com/team/shop_backend", "nginx"} {
		d := shop()
		d.Workloads[0].ImageRepository = repo
		if _, err := Validate(d); err != nil {
			t.Errorf("%s rejected: %v", repo, err)
		}
	}
}

func hasProblem(err error, code, field string) bool {
	v, ok := err.(*domain.ValidationError)
	if !ok {
		return false
	}
	for _, p := range v.Problems {
		if p.Code == code && p.Field == field {
			return true
		}
	}
	return false
}
