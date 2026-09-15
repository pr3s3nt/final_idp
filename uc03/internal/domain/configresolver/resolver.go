// Package configresolver is the Environment Configuration Resolver: it turns
// the value sources of one workload into resolved values for one execution.
package configresolver

import (
	"context"
	"fmt"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/manifest"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/secretstore"
)

// ResolveEnvironmentConfiguration resolves direct values, Resource Outputs,
// Workload Outputs and secret references for one workload of the deployed
// version. Only bindings of definitions present in that version are used.
func ResolveEnvironmentConfiguration(ctx context.Context, cfg *domain.EnvironmentConfiguration, w *domain.Workload,
	resourceOutputs, workloadOutputs map[string]domain.Outputs, secrets secretstore.Store) (*manifest.ResolvedConfiguration, error) {

	rc := &manifest.ResolvedConfiguration{Variables: map[string]string{}, Secrets: map[string]string{}}
	varDefs, secretDefs := map[string]string{}, map[string]string{}
	for _, d := range w.Variables {
		varDefs[d.ID] = d.Name
	}
	for _, d := range w.Secrets {
		secretDefs[d.ID] = d.Name
	}
	output := func(source map[string]domain.Outputs, refID, output, bindingName string) (domain.Output, error) {
		outs, ok := source[refID]
		if !ok {
			return domain.Output{}, fmt.Errorf("%s.%s: outputs of the referenced component are not available", w.Name, bindingName)
		}
		o, ok := outs[output]
		if !ok {
			return domain.Output{}, fmt.Errorf("%s.%s: output %q is not provided", w.Name, bindingName, output)
		}
		return o, nil
	}

	for _, b := range cfg.Variables {
		name, ok := varDefs[b.DefinitionID]
		if !ok || b.WorkloadID != w.ID {
			continue
		}
		switch b.Source {
		case domain.SourceDirect:
			rc.Variables[name] = b.DirectValue
		case domain.SourceResourceOutput:
			o, err := output(resourceOutputs, b.RefID, b.OutputName, name)
			if err != nil {
				return nil, err
			}
			rc.Variables[name] = o.Value
		case domain.SourceWorkloadOutput:
			o, err := output(workloadOutputs, b.RefID, b.OutputName, name)
			if err != nil {
				return nil, err
			}
			rc.Variables[name] = o.Value
		}
	}
	for _, b := range cfg.Secrets {
		name, ok := secretDefs[b.DefinitionID]
		if !ok || b.WorkloadID != w.ID {
			continue
		}
		switch b.Source {
		case domain.SourceSecretRef:
			v, err := secrets.Get(ctx, b.SecretRef)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: secret reference cannot be read", w.Name, name)
			}
			rc.Secrets[name] = string(v)
		case domain.SourceResourceOutput:
			o, err := output(resourceOutputs, b.RefID, b.OutputName, name)
			if err != nil {
				return nil, err
			}
			rc.Secrets[name] = o.Value
		}
	}
	return rc, nil
}
