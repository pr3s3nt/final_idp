package manifest_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"sdp/internal/domain"
	"sdp/internal/domain/manifest"
)

// Runs the real score-k8s binary; skipped when it is not installed.
func TestPipelineRendersAdaptsAndMaterializesWithoutSecretValues(t *testing.T) {
	if _, err := exec.LookPath("score-k8s"); err != nil {
		t.Skip("score-k8s not installed")
	}
	port := 8080
	w := &domain.Workload{ID: "w1", Name: "backend", Port: &port}
	wd := domain.WorkloadDeployment{WorkloadID: "0000-backend", ImageRepository: "registry.company.local/shop-backend", ImageVersion: "v1"}
	rc := &manifest.ResolvedConfiguration{
		Variables: map[string]string{"DB_HOST": "postgres.res.svc", "PRICE": "costs $5"},
		Secrets:   map[string]string{"DB_PASSWORD": "super-secret-value"},
	}
	ctx := context.Background()
	render := func(rc *manifest.ResolvedConfiguration) (string, []byte) {
		spec, err := manifest.GenerateResolvedApplicationSpecification(w, wd, rc)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(spec.Score), "super-secret-value") {
			t.Fatal("secrets must never enter the Score specification")
		}
		objs, err := (&manifest.ScoreRenderer{}).GenerateKubernetesManifest(ctx, spec, "shop-app-staging")
		if err != nil {
			t.Fatal(err)
		}
		target := manifest.TargetInfo{Application: "shop-app", Environment: "STAGING", Namespace: "shop-app-staging", RegistryMirror: "localhost:5055"}
		objs, _ = manifest.AdaptManifestForTarget(objs, w, wd, target)
		objs, _ = manifest.MaterializeEnvironmentConfiguration(objs, w, wd, target, []byte("k"))
		objs, secrets, _ := manifest.MaterializeSecretConfiguration(objs, w, wd, rc, []byte("k"))
		body, err := manifest.Encode(objs)
		if err != nil {
			t.Fatal(err)
		}
		if len(secrets) != 1 || string(secrets[0].Data["DB_PASSWORD"]) != rc.Secrets["DB_PASSWORD"] {
			t.Fatalf("secret object not produced: %+v", secrets)
		}
		return string(body), body
	}

	first, _ := render(rc)
	for _, want := range []string{
		"image: localhost:5055/shop-backend:v1", "namespace: shop-app-staging", "idp.dev/workload-id: 0000-backend",
		"configMapKeyRef", "secretKeyRef", "kind: ConfigMap", "PRICE: costs $5", "tcpSocket",
	} {
		if !strings.Contains(first, want) {
			t.Errorf("manifest missing %q:\n%s", want, first)
		}
	}
	for _, forbidden := range []string{"super-secret-value", "score-k8s", "instance: backend-"} {
		if strings.Contains(first, forbidden) {
			t.Errorf("manifest must not contain %q", forbidden)
		}
	}
	second, _ := render(rc)
	if first != second {
		t.Fatal("rendering the same input twice must give identical manifests (stable selectors, no random labels)")
	}
	rotated := &manifest.ResolvedConfiguration{Variables: rc.Variables, Secrets: map[string]string{"DB_PASSWORD": "rotated"}}
	third, _ := render(rotated)
	if third == first {
		t.Fatal("a changed secret must change the pod template (hash annotation) so pods roll")
	}
}
