package provisioner

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
)

type ApplyRequest struct {
	Item       domain.InfrastructurePlanItem
	Context    domain.DeploymentContext
	SecretRefs []domain.SecretReference
}

type ApplyResult struct {
	InfrastructureReference string
	Outputs                 map[string]string
}

type InspectorResult struct {
	Exists     bool
	Compatible bool
	Outputs    map[string]string
}

type Provisioner interface {
	Apply(context.Context, ApplyRequest) (ApplyResult, error)
	Inspect(context.Context, ApplyRequest) (InspectorResult, error)
	Outputs(context.Context, domain.InfrastructurePlanItem) (map[string]string, error)
}

type Reference struct {
	Kind    string
	Module  string
	Version string
}

func ParseReference(raw string) (Reference, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Reference{}, fmt.Errorf("INVALID_PROVISIONER_REFERENCE: %q", raw)
	}
	module := strings.TrimPrefix(parsed.Host+parsed.Path, "/")
	parts := strings.Split(module, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Reference{}, fmt.Errorf("INVALID_PROVISIONER_REFERENCE: %q must pin a module version", raw)
	}
	return Reference{Kind: parsed.Scheme, Module: parts[0], Version: parts[1]}, nil
}

type Registry struct{ providers map[string]Provisioner }

func NewRegistry() *Registry { return &Registry{providers: map[string]Provisioner{}} }

func (r *Registry) Register(kind string, provider Provisioner) error {
	if kind == "" || provider == nil {
		return fmt.Errorf("INVALID_PROVISIONER: kind and implementation are required")
	}
	if _, exists := r.providers[kind]; exists {
		return fmt.Errorf("PROVISIONER_ALREADY_REGISTERED: %s", kind)
	}
	r.providers[kind] = provider
	return nil
}

func (r *Registry) Resolve(rawReference string) (Provisioner, Reference, error) {
	reference, err := ParseReference(rawReference)
	if err != nil {
		return nil, Reference{}, err
	}
	provider, ok := r.providers[reference.Kind]
	if !ok {
		return nil, Reference{}, fmt.Errorf("PROVISIONER_NOT_REGISTERED: %s", reference.Kind)
	}
	return provider, reference, nil
}
