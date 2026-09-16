// Package deliveryrepo is the Delivery Repository Provider abstraction: it
// makes sure an application has its own place to keep desired state, with a key
// pair that belongs to that application only. Nothing outside this package
// knows which Git hosting is used.
package deliveryrepo

import (
	"context"
	"fmt"
	"strings"
)

// Application is the application a delivery repository is prepared for.
type Application struct {
	ID   string
	Name string
}

// Repository is a prepared delivery repository. The keys are not returned:
// only the references under which they were stored in the Secret Store.
type Repository struct {
	URL               string // git@host:owner/name.git
	Branch            string
	WriteKeyReference string // used by the IDP to push desired state
	ReadKeyReference  string // handed to the CD system to read it
}

// Provider prepares the delivery repository of an application. It is called
// once per application, on its first deployment.
type Provider interface {
	EnsureDeliveryRepository(ctx context.Context, app Application) (Repository, error)
}

// RepositoryName expands the platform naming pattern for one application, for
// example "pr3s3nt/idp-<app>-gitops" with application "shop-app" becomes
// "pr3s3nt/idp-shop-app-gitops".
func RepositoryName(pattern string, application string) (owner, name string, err error) {
	full := strings.ReplaceAll(pattern, "<app>", slug(application))
	owner, name, ok := strings.Cut(full, "/")
	if !ok || owner == "" || name == "" {
		return "", "", fmt.Errorf("delivery repository pattern %q must look like owner/name and contain <app>", pattern)
	}
	if strings.Contains(name, "/") {
		return "", "", fmt.Errorf("delivery repository pattern %q must name exactly one repository", pattern)
	}
	return owner, name, nil
}

// slug keeps the application recognisable in the repository name while staying
// inside what Git hosting accepts.
func slug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
