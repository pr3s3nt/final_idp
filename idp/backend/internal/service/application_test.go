package service

import (
	"testing"

	"idp/internal/domain"
)

func validDraft() ApplicationDefinitionDraft {
	return ApplicationDefinitionDraft{Name: "shop-app", Workloads: []WorkloadDraft{{
		ID: "11111111-1111-4111-8111-111111111111", Name: "backend", Type: "Backend Service",
		ImageRepository: "registry.company.local/shop-backend",
	}}}
}

func TestValidateDraftBaseVersionRules(t *testing.T) {
	one := 1
	zero := 0

	d := validDraft()
	if _, err := ValidateDraft("", d); err != nil {
		t.Fatalf("create rejected: %v", err)
	}
	d.BaseVersion = &one
	if _, err := ValidateDraft("", d); !domain.HasCode(err, domain.CodeInvalidInput) {
		t.Fatalf("create with baseVersion accepted: %v", err)
	}

	d = validDraft()
	appID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	if _, err := ValidateDraft(appID, d); !domain.HasCode(err, domain.CodeInvalidInput) {
		t.Fatalf("edit without baseVersion accepted: %v", err)
	}
	d.BaseVersion = &zero
	if _, err := ValidateDraft(appID, d); !domain.HasCode(err, domain.CodeInvalidInput) {
		t.Fatalf("edit with baseVersion 0 accepted: %v", err)
	}
	d.BaseVersion = &one
	d.ApplicationID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	if _, err := ValidateDraft(appID, d); !domain.HasCode(err, domain.CodeInvalidInput) {
		t.Fatalf("mismatched applicationId accepted: %v", err)
	}
	d.ApplicationID = appID
	in, err := ValidateDraft(appID, d)
	if err != nil || in.BaseVersion != 1 || in.ApplicationID != appID {
		t.Fatalf("edit rejected: %v %+v", err, in)
	}
}
