// Package appspec is the UC-01 Application Specification Generator. It turns
// one saved Application Definition Version into an unresolved Score
// specification: one YAML document per workload.
package appspec

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"idp/internal/domain"
)

// Format is the value stored in application_specification.format.
const Format = "score-yaml"

// Specification is a generated, unresolved application specification.
type Specification struct {
	Format  string
	Content string
	Version string
}

// Generate builds the specification. It contains image repositories without
// versions, configuration requirement names without values, and the
// depends-on relations. It never contains a Secret value.
func Generate(v *domain.ApplicationVersion) (*Specification, error) {
	names := map[string]string{}
	for _, w := range v.Workloads {
		names[w.ID] = w.Name
	}
	for _, r := range v.Resources {
		names[r.ID] = r.Name
	}

	workloads := append([]domain.Workload(nil), v.Workloads...)
	sort.Slice(workloads, func(i, j int) bool { return workloads[i].Name < workloads[j].Name })

	var out bytes.Buffer
	for i, w := range workloads {
		container := map[string]any{"image": w.ImageRepository}
		if len(w.Variables) > 0 {
			// Values are configured per environment in UC-02; the empty
			// string marks a declared requirement.
			vars := map[string]string{}
			for _, d := range w.Variables {
				vars[d.Name] = ""
			}
			container["variables"] = vars
		}
		annotations := map[string]string{
			"idp.dev/application":   v.ApplicationName,
			"idp.dev/workload-type": w.Type,
		}
		if s := definitionNames(w.Secrets); s != "" {
			annotations["idp.dev/secrets"] = s
		}
		if len(w.ExposedOutputs) > 0 {
			outputs := append([]string(nil), w.ExposedOutputs...)
			sort.Strings(outputs)
			annotations["idp.dev/outputs"] = strings.Join(outputs, ",")
		}
		var dependsOn []string
		resources := map[string]any{}
		for _, d := range v.Dependencies {
			if d.SourceWorkloadID != w.ID {
				continue
			}
			dependsOn = append(dependsOn, names[d.TargetID])
			if d.TargetType == domain.TargetResource {
				if r := v.Resource(d.TargetID); r != nil {
					resources[r.Name] = map[string]string{"type": r.ResourceType}
				}
			}
		}
		if len(dependsOn) > 0 {
			sort.Strings(dependsOn)
			annotations["idp.dev/depends-on"] = strings.Join(dependsOn, ",")
		}
		doc := map[string]any{
			"apiVersion": "score.dev/v1b1",
			"metadata":   map[string]any{"name": w.Name, "annotations": annotations},
			"containers": map[string]any{"main": container},
		}
		if w.Port != nil {
			doc["service"] = map[string]any{"ports": map[string]any{"web": map[string]any{"port": *w.Port, "targetPort": *w.Port}}}
		}
		if len(resources) > 0 {
			doc["resources"] = resources
		}
		body, err := yaml.Marshal(doc)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			out.WriteString("---\n")
		}
		out.Write(body)
	}
	return &Specification{Format: Format, Content: out.String(), Version: fmt.Sprintf("v%d", v.VersionNumber)}, nil
}

func definitionNames(defs []domain.ConfigDefinition) string {
	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Name
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
