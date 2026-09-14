package provisioner_test

import (
	"testing"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/provisioner"
)

func TestParsePinnedTerraformReference(t *testing.T) {
	got, err := provisioner.ParseReference("terraform://modules/postgres@v1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "terraform" || got.Module != "modules/postgres" || got.Version != "v1" {
		t.Fatalf("reference = %#v", got)
	}
}
