package deliveryrepo

import "testing"

func TestRepositoryName(t *testing.T) {
	for _, tc := range []struct {
		pattern     string
		application string
		owner, name string
		wantErr     bool
	}{
		{pattern: "pr3s3nt/idp-<app>-gitops", application: "shop-app", owner: "pr3s3nt", name: "idp-shop-app-gitops"},
		{pattern: "acme/<app>", application: "Reporting App", owner: "acme", name: "reporting-app"},
		{pattern: "acme/<app>-state", application: "a/b", owner: "acme", name: "a-b-state"},
		{pattern: "idp-<app>-gitops", application: "shop-app", wantErr: true},
		{pattern: "owner/group/<app>", application: "shop-app", wantErr: true},
	} {
		owner, name, err := RepositoryName(tc.pattern, tc.application)
		if tc.wantErr {
			if err == nil {
				t.Errorf("pattern %q: want an error, got %s/%s", tc.pattern, owner, name)
			}
			continue
		}
		if err != nil {
			t.Errorf("pattern %q: %v", tc.pattern, err)
			continue
		}
		if owner != tc.owner || name != tc.name {
			t.Errorf("pattern %q with %q: got %s/%s, want %s/%s", tc.pattern, tc.application, owner, name, tc.owner, tc.name)
		}
	}
}

// Two applications must never be given the same delivery repository.
func TestRepositoryNameSeparatesApplications(t *testing.T) {
	a, b := "shop-app", "shop-app-2"
	_, first, err := RepositoryName("pr3s3nt/idp-<app>-gitops", a)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := RepositoryName("pr3s3nt/idp-<app>-gitops", b)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("%s and %s share the repository name %s", a, b, first)
	}
}
