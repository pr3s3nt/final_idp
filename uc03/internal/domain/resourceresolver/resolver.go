// Package resourceresolver is the Resource Definition Resolver: it selects the
// catalog definition for a resource by type, deployment context and
// applicability, and fails on ambiguity instead of picking arbitrarily.
package resourceresolver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
)

// Resolver resolves against the definitions of one catalog version. All holds
// the definitions of every catalog version, so instances created with another
// version can still be looked up by definition ID.
type Resolver struct {
	Definitions []domain.ResourceDefinition
	All         []domain.ResourceDefinition
}

// Scope identifies who the resource is resolved for.
type Scope struct {
	ApplicationID   string
	ApplicationName string
	Environment     domain.Environment
	Context         domain.DeploymentContext
}

// Resolve returns the single most specific definition of resourceType that
// supports the context and whose applicability conditions all match.
func (r *Resolver) Resolve(requirementID, resourceType string, s Scope) (*domain.ResourceResolution, *domain.Problem) {
	type candidate struct {
		def         *domain.ResourceDefinition
		specificity int
	}
	var matches []candidate
	for i := range r.Definitions {
		d := &r.Definitions[i]
		if d.ResourceType != resourceType || !SupportsContext(d, s.Context) {
			continue
		}
		spec, ok := applicable(d, s)
		if !ok {
			continue
		}
		matches = append(matches, candidate{d, spec})
	}
	if len(matches) == 0 {
		return nil, &domain.Problem{Code: domain.CodeNoResourceDefinition, Message: fmt.Sprintf(
			"no resource definition of type %q supports target %q (%s) for %s/%s",
			resourceType, s.Context.Target, contextString(s.Context), s.ApplicationName, s.Environment)}
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].specificity > matches[j].specificity })
	if len(matches) > 1 && matches[0].specificity == matches[1].specificity {
		var names []string
		for _, m := range matches {
			if m.specificity == matches[0].specificity {
				names = append(names, m.def.Name)
			}
		}
		return nil, &domain.Problem{Code: domain.CodeAmbiguousDefinition, Message: fmt.Sprintf(
			"resource type %q matches several equally specific definitions: %s", resourceType, strings.Join(names, ", "))}
	}
	best := matches[0]
	reason := fmt.Sprintf("type %s, context %s", resourceType, contextString(s.Context))
	if best.specificity > 0 {
		reason += fmt.Sprintf(", %d applicability condition(s) matched", best.specificity)
	}
	return &domain.ResourceResolution{RequirementID: requirementID, Definition: best.def, Reason: reason}, nil
}

// SupportsContext reports whether any supported_contexts entry matches every
// one of its keys against the deployment context.
func SupportsContext(d *domain.ResourceDefinition, c domain.DeploymentContext) bool {
	values := c.Values()
	for _, entry := range d.SupportedContexts {
		ok := true
		for k, v := range entry {
			if values[k] != v {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func applicable(d *domain.ResourceDefinition, s Scope) (int, bool) {
	matched := 0
	for key, allowed := range d.ApplicabilityConditions {
		var actual []string
		switch key {
		case "application":
			actual = []string{s.ApplicationName, s.ApplicationID}
		case "environment":
			actual = []string{string(s.Environment)}
		default:
			return 0, false // unknown condition keys never match
		}
		if !anyIn(actual, allowed) {
			return 0, false
		}
		matched++
	}
	return matched, true
}

func anyIn(actual, allowed []string) bool {
	for _, a := range actual {
		for _, b := range allowed {
			if a == b {
				return true
			}
		}
	}
	return false
}

func contextString(c domain.DeploymentContext) string {
	parts := []string{"target=" + c.Target}
	if c.CloudProvider != "" {
		parts = append(parts, "cloudProvider="+c.CloudProvider)
	}
	if c.Region != "" {
		parts = append(parts, "region="+c.Region)
	}
	return strings.Join(parts, " ")
}

// TargetOption is a deployment target the catalog has a k8s-cluster definition
// for: a cloud target where the IDP builds the cluster, or an internal cluster.
type TargetOption struct {
	Target        string `json:"target"`
	CloudProvider string `json:"cloudProvider"`
	Region        string `json:"region,omitempty"`
}

// SupportedTargets lists the targets that have a k8s-cluster definition; a
// target is supported only if the catalog can build its cluster.
func (r *Resolver) SupportedTargets() []TargetOption {
	seen := map[TargetOption]bool{}
	var out []TargetOption
	for i := range r.Definitions {
		d := &r.Definitions[i]
		if d.ResourceType != domain.ResourceTypeCluster {
			continue
		}
		for _, e := range d.SupportedContexts {
			o := TargetOption{Target: e["target"], CloudProvider: e["cloudProvider"], Region: e["region"]}
			if o.Target != "" && !seen[o] {
				seen[o] = true
				out = append(out, o)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Target+out[i].Region < out[j].Target+out[j].Region })
	return out
}

// DefinitionByID finds a definition by ID in any catalog version.
func (r *Resolver) DefinitionByID(id string) *domain.ResourceDefinition {
	for _, list := range [][]domain.ResourceDefinition{r.Definitions, r.All} {
		for i := range list {
			if list[i].ID == id {
				return &list[i]
			}
		}
	}
	return nil
}
