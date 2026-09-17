package appspec

import (
	"strings"
	"testing"

	"idp/internal/domain"
)

func TestGenerateScoreDocumentsWithoutVersionsOrValues(t *testing.T) {
	p := 8080
	v := &domain.ApplicationVersion{ApplicationName: "shop-app", VersionNumber: 3,
		Workloads: []domain.Workload{
			{ID: "w2", Name: "frontend", Type: "Frontend", ImageRepository: "registry.company.local/shop-frontend"},
			{ID: "w1", Name: "backend", Type: "Backend Service", ImageRepository: "registry.company.local/shop-backend", Port: &p,
				ExposedOutputs: []string{"endpoint"},
				Variables:      []domain.ConfigDefinition{{Name: "DB_HOST", Required: true}},
				Secrets:        []domain.ConfigDefinition{{Name: "DB_PASSWORD", Required: true}}},
		},
		Resources: []domain.ResourceRequirement{{ID: "r1", Name: "postgresql", ResourceType: "PostgreSQL"}},
		Dependencies: []domain.Dependency{
			{SourceWorkloadID: "w2", TargetType: domain.TargetWorkload, TargetID: "w1"},
			{SourceWorkloadID: "w1", TargetType: domain.TargetResource, TargetID: "r1"},
		},
	}
	spec, err := Generate(v)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Format != "score-yaml" || spec.Version != "v3" {
		t.Fatalf("format/version: %+v", spec)
	}
	docs := strings.Split(spec.Content, "---\n")
	if len(docs) != 2 || !strings.Contains(docs[0], "name: backend") || !strings.Contains(docs[1], "name: frontend") {
		t.Fatalf("expected backend then frontend documents:\n%s", spec.Content)
	}
	for _, want := range []string{
		"image: registry.company.local/shop-backend\n",
		`DB_HOST: ""`,
		"idp.dev/secrets: DB_PASSWORD",
		"idp.dev/depends-on: postgresql",
		"type: PostgreSQL",
		"port: 8080",
		"idp.dev/depends-on: backend",
	} {
		if !strings.Contains(spec.Content, want) {
			t.Errorf("missing %q in:\n%s", want, spec.Content)
		}
	}
	again, _ := Generate(v)
	if again.Content != spec.Content {
		t.Error("generation is not deterministic")
	}
}
