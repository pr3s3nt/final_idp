package fingerprint_test

import (
	"strings"
	"testing"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/fingerprint"
)

func TestMapOrderDoesNotChangeFingerprint(t *testing.T) {
	a, err := fingerprint.Sum(map[string]any{"b": int64(2), "a": "one"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := fingerprint.Sum(map[string]any{"a": "one", "b": int64(2)})
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

func TestFloatIsRejected(t *testing.T) {
	_, err := fingerprint.Sum(map[string]any{"bad": 1.5})
	if err == nil || !strings.Contains(err.Error(), "floating point") {
		t.Fatalf("err = %v", err)
	}
}
