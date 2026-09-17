// Package resourceoutput is the Resource Output Resolver / Collector: it reads
// the outputs of a ready Resource Instance from provider state (MANAGED) or
// from the shared resource's connection record (EXISTING).
package resourceoutput

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pr3s3nt/final_idp/idp/backend/internal/domain"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/integration/provisioner"
	"github.com/pr3s3nt/final_idp/idp/backend/internal/integration/secretstore"
)

type Collector struct {
	Provisioner provisioner.Provisioner
	Secrets     secretstore.Store
}

// CollectResourceOutputs returns the declared outputs of an instance, flagging
// sensitive ones. It fails when a declared output is missing.
func (c *Collector) CollectResourceOutputs(ctx context.Context, ri *domain.ResourceInstance, d *domain.ResourceDefinition) (domain.Outputs, error) {
	if ri.Status != domain.RIReady {
		return nil, fmt.Errorf("resource instance %s is %s, not READY", ri.ID, ri.Status)
	}
	var raw map[string]string
	var err error
	switch d.ManagementMode {
	case domain.Managed:
		raw, err = c.Provisioner.Outputs(ctx, ri.ID)
	case domain.Existing:
		var body []byte
		body, err = c.Secrets.Get(ctx, d.ExistingResourceReference)
		if err == nil {
			err = json.Unmarshal(body, &raw)
		}
	}
	if err != nil {
		return nil, err
	}
	out := domain.Outputs{}
	declared := append(append([]string{}, d.ExposedOutputs...), d.SensitiveOutputs...)
	for _, name := range declared {
		v, ok := raw[name]
		if !ok {
			return nil, fmt.Errorf("output %q declared by %s is not provided by the resource", name, d.Name)
		}
		out[name] = domain.Output{Value: v, Sensitive: d.IsSensitiveOutput(name)}
	}
	return out, nil
}
