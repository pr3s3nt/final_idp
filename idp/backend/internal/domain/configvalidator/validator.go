// Package configvalidator is the Environment Configuration Validator of UC-02.
// It checks a complete client-owned draft against the latest Application
// Definition version and the Resource Definitions resolved from the Catalog
// Version and deployment target the Developer selected (ADR-020).
package configvalidator

import (
	"idp/internal/domain"
)

// Input is everything the validator needs. It holds no state of its own, so
// the same draft always produces the same problems.
type Input struct {
	// Version is the latest Application Definition version; every reference is
	// checked against it.
	Version *domain.ApplicationVersion
	// Configuration is the complete draft translated into bindings.
	Configuration *domain.EnvironmentConfiguration
	// Definitions maps a Resource Requirement ID to the single Resource
	// Definition resolved for it. A requirement that could not be resolved is
	// absent, which is itself a problem when a binding references it.
	Definitions map[string]*domain.ResourceDefinition
}

// Validate reports every problem in one pass so the Developer can fix them
// together (A1). It returns nil when the draft may be saved.
func Validate(in Input) error {
	p := &domain.ValidationError{}
	v, cfg := in.Version, in.Configuration
	if v == nil || cfg == nil {
		p.Add(domain.CodeInvalidInput, "the draft has no application version or no configuration")
		return p.OrNil()
	}
	if !cfg.Environment.Valid() {
		p.Add(domain.CodeInvalidInput, "environment %q must be STAGING or PRODUCTION", string(cfg.Environment))
	}

	// Index the declared requirements of the latest version by workload and
	// definition ID, so a binding can be matched to what declares it.
	type requirement struct {
		workload *domain.Workload
		def      domain.ConfigDefinition
	}
	variables, secrets := map[string]requirement{}, map[string]requirement{}
	for i := range v.Workloads {
		w := &v.Workloads[i]
		for _, def := range w.Variables {
			variables[w.ID+"/"+def.ID] = requirement{w, def}
		}
		for _, def := range w.Secrets {
			secrets[w.ID+"/"+def.ID] = requirement{w, def}
		}
	}

	bound := map[string]bool{}
	check := func(b domain.ConfiguredValue, secret bool) {
		declared, kind := variables, "variable"
		if secret {
			declared, kind = secrets, "secret"
		}
		key := b.WorkloadID + "/" + b.DefinitionID
		req, ok := declared[key]
		if !ok {
			p.Add(domain.CodeConfigurationMismatch, "a %s binding references %s, which version %d does not declare for that workload",
				kind, b.DefinitionID, v.VersionNumber)
			return
		}
		if bound[kind+":"+key] {
			p.Add(domain.CodeInvalidInput, "%s %s.%s is configured more than once", kind, req.workload.Name, req.def.Name)
			return
		}
		bound[kind+":"+key] = true
		checkSource(p, in, req.workload, req.def.Name, b, secret)
	}
	for _, b := range cfg.Variables {
		check(b, false)
	}
	for _, b := range cfg.Secrets {
		check(b, true)
	}

	// Every requirement the version marks as required needs a binding.
	for i := range v.Workloads {
		w := &v.Workloads[i]
		for _, def := range w.Variables {
			if def.Required && !bound["variable:"+w.ID+"/"+def.ID] {
				p.Add(domain.CodeMissingConfiguration, "%s.%s is required by version %d but has no value", w.Name, def.Name, v.VersionNumber)
			}
		}
		for _, def := range w.Secrets {
			if def.Required && !bound["secret:"+w.ID+"/"+def.ID] {
				p.Add(domain.CodeMissingConfiguration, "secret %s.%s is required by version %d but has no value", w.Name, def.Name, v.VersionNumber)
			}
		}
	}
	return p.OrNil()
}

// checkSource validates the one value source of a single binding.
func checkSource(p *domain.ValidationError, in Input, w *domain.Workload, name string, b domain.ConfiguredValue, secret bool) {
	v := in.Version
	switch b.Source {
	case domain.SourceDirect:
		if secret {
			p.Add(domain.CodeInvalidOutputReference, "secret %s.%s cannot be a plaintext direct value", w.Name, name)
		} else if b.DirectValue == "" {
			p.Add(domain.CodeMissingConfiguration, "%s.%s has an empty direct value", w.Name, name)
		}

	case domain.SourceSecretRef:
		switch {
		case !secret:
			p.Add(domain.CodeInvalidOutputReference, "%s.%s is a variable but uses a secret reference", w.Name, name)
		case b.SecretRef == "":
			p.Add(domain.CodeMissingConfiguration, "secret %s.%s has no secret reference", w.Name, name)
		}

	case domain.SourceResourceOutput:
		resource := v.Resource(b.RefID)
		switch {
		case resource == nil:
			p.Add(domain.CodeConfigurationMismatch, "%s.%s references a resource that is not in version %d", w.Name, name, v.VersionNumber)
			return
		case !v.DependsOn(w.ID, b.RefID):
			p.Add(domain.CodeConfigurationMismatch, "%s.%s references %s, which %s does not depend on in version %d",
				w.Name, name, resource.Name, w.Name, v.VersionNumber)
			return
		}
		d := in.Definitions[b.RefID]
		if d == nil {
			p.Add(domain.CodeNoResourceDefinition, "%s.%s references %s, which the selected catalog version and deployment target do not resolve to a resource definition",
				w.Name, name, resource.Name)
			return
		}
		switch {
		case !d.HasOutput(b.OutputName):
			p.Add(domain.CodeInvalidOutputReference, "%s.%s references output %q, which definition %s does not expose",
				w.Name, name, b.OutputName, d.Name)
		case secret && !d.IsSensitiveOutput(b.OutputName):
			p.Add(domain.CodeInvalidOutputReference, "secret %s.%s must use a sensitive output; %q is not sensitive", w.Name, name, b.OutputName)
		case !secret && d.IsSensitiveOutput(b.OutputName):
			p.Add(domain.CodeInvalidOutputReference, "%s.%s uses sensitive output %q; bind it as a secret", w.Name, name, b.OutputName)
		}

	case domain.SourceWorkloadOutput:
		target := v.Workload(b.RefID)
		switch {
		case secret:
			p.Add(domain.CodeInvalidOutputReference, "secret %s.%s cannot come from a workload output", w.Name, name)
		case target == nil:
			p.Add(domain.CodeConfigurationMismatch, "%s.%s references a workload that is not in version %d", w.Name, name, v.VersionNumber)
		case !v.DependsOn(w.ID, b.RefID):
			p.Add(domain.CodeConfigurationMismatch, "%s.%s references %s, which %s does not depend on in version %d",
				w.Name, name, target.Name, w.Name, v.VersionNumber)
		case !contains(target.ExposedOutputs, b.OutputName):
			p.Add(domain.CodeInvalidOutputReference, "%s.%s references output %q, which %s does not expose", w.Name, name, b.OutputName, target.Name)
		}

	default:
		p.Add(domain.CodeInvalidInput, "%s.%s has no valid value source", w.Name, name)
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
