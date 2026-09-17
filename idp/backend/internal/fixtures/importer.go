// Package fixtures imports UC-01 (application versions), UC-02 (environment
// configuration) and platform catalog input from YAML, standing in for the
// editors of those use cases.
package fixtures

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"sdp/internal/domain"
	"sdp/internal/integration/secretstore"
	"sdp/internal/persistence"
)

type Importer struct {
	Apps    *persistence.ApplicationRepository
	Configs *persistence.EnvironmentConfigurationRepository
	Catalog *persistence.ResourceDefinitionCatalog
	Secrets secretstore.Store
	Log     func(format string, args ...any)
}

// catalogFile is one immutable catalog version (fixtures/catalog/v<N>.yaml).
type catalogFile struct {
	Version     int `yaml:"version"`
	Definitions []struct {
		Name                      string                         `yaml:"name"`
		ResourceType              string                         `yaml:"resourceType"`
		ManagementMode            string                         `yaml:"managementMode"`
		ProvisionerReference      string                         `yaml:"provisionerReference"`
		ExistingResourceReference string                         `yaml:"existingResourceReference"`
		SupportedContexts         []map[string]string            `yaml:"supportedContexts"`
		ApplicabilityConditions   map[string][]string            `yaml:"applicabilityConditions"`
		DefaultParameters         map[string]any                 `yaml:"defaultParameters"`
		AllowedOverrides          map[string]domain.OverrideRule `yaml:"allowedOverrides"`
		ExposedOutputs            []string                       `yaml:"exposedOutputs"`
		SensitiveOutputs          []string                       `yaml:"sensitiveOutputs"`
		Requires                  []string                       `yaml:"requires"`
	} `yaml:"definitions"`
}

type applicationFile struct {
	Application string `yaml:"application"`
	Description string `yaml:"description"`
	Versions    []struct {
		Resources []struct {
			Name string `yaml:"name"`
			Type string `yaml:"type"`
		} `yaml:"resources"`
		Workloads []struct {
			Name            string                    `yaml:"name"`
			Type            string                    `yaml:"type"`
			ImageRepository string                    `yaml:"imageRepository"`
			Port            *int                      `yaml:"port"`
			Outputs         []string                  `yaml:"outputs"`
			Variables       []domain.ConfigDefinition `yaml:"variables"`
			Secrets         []domain.ConfigDefinition `yaml:"secrets"`
		} `yaml:"workloads"`
		Dependencies []persistence.DependencyInput `yaml:"dependencies"`
	} `yaml:"versions"`
}

type binding struct {
	Direct         *string `yaml:"direct"`
	ResourceOutput string  `yaml:"resourceOutput"`
	WorkloadOutput string  `yaml:"workloadOutput"`
	Generate       bool    `yaml:"generate"`
}

type configurationFile struct {
	Configurations []struct {
		Application string                        `yaml:"application"`
		Environment string                        `yaml:"environment"`
		Variables   map[string]map[string]binding `yaml:"variables"`
		Secrets     map[string]map[string]binding `yaml:"secrets"`
	} `yaml:"configurations"`
}

// ImportDirectory imports catalog/*.yaml, applications/*.yaml and
// configurations/*.yaml from dir. Catalog versions and application versions
// are only added, never changed, so re-running is safe.
func (im *Importer) ImportDirectory(ctx context.Context, dir string) error {
	catalogs, _ := filepath.Glob(filepath.Join(dir, "catalog", "*.yaml"))
	sort.Strings(catalogs)
	for _, f := range catalogs {
		if err := im.importCatalog(ctx, f); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
	}
	apps, _ := filepath.Glob(filepath.Join(dir, "applications", "*.yaml"))
	sort.Strings(apps)
	for _, f := range apps {
		if err := im.importApplication(ctx, f); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
	}
	configs, _ := filepath.Glob(filepath.Join(dir, "configurations", "*.yaml"))
	sort.Strings(configs)
	for _, f := range configs {
		if err := im.importConfigurations(ctx, f); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
	}
	return nil
}

func readYAML(path string, v any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal([]byte(os.ExpandEnv(string(raw))), v)
}

func (im *Importer) importCatalog(ctx context.Context, path string) error {
	var f catalogFile
	if err := readYAML(path, &f); err != nil {
		return err
	}
	if f.Version < 1 {
		return fmt.Errorf("catalog file needs version >= 1")
	}
	var defs []domain.ResourceDefinition
	for _, d := range f.Definitions {
		defs = append(defs, domain.ResourceDefinition{
			Name: d.Name, ResourceType: d.ResourceType, ManagementMode: domain.ManagementMode(d.ManagementMode),
			ProvisionerReference: d.ProvisionerReference, ExistingResourceReference: d.ExistingResourceReference,
			SupportedContexts: d.SupportedContexts, ApplicabilityConditions: d.ApplicabilityConditions,
			DefaultParameters: d.DefaultParameters, AllowedOverrides: d.AllowedOverrides,
			ExposedOutputs: d.ExposedOutputs, SensitiveOutputs: d.SensitiveOutputs, Requires: d.Requires,
		})
	}
	created, err := im.Catalog.CreateVersion(ctx, f.Version, defs)
	if err != nil {
		return err
	}
	if created {
		im.Log("catalog: version %d created (%d definitions)", f.Version, len(defs))
	} else {
		im.Log("catalog: version %d already exists; catalog versions are immutable, file ignored", f.Version)
	}
	return nil
}

func (im *Importer) importApplication(ctx context.Context, path string) error {
	var f applicationFile
	if err := readYAML(path, &f); err != nil {
		return err
	}
	existing := 0
	if app, err := im.Apps.FindApplication(ctx, f.Application); err == nil {
		existing = app.LatestVersion
	}
	for i, v := range f.Versions {
		if i < existing {
			continue
		}
		in := persistence.VersionInput{ApplicationName: f.Application, Description: f.Description, Dependencies: v.Dependencies}
		for _, r := range v.Resources {
			in.Resources = append(in.Resources, domain.ResourceRequirement{Name: r.Name, ResourceType: r.Type})
		}
		for _, w := range v.Workloads {
			in.Workloads = append(in.Workloads, domain.Workload{Name: w.Name, Type: w.Type, ImageRepository: w.ImageRepository,
				Port: w.Port, ExposedOutputs: w.Outputs, Variables: w.Variables, Secrets: w.Secrets})
		}
		saved, err := im.Apps.SaveNewVersion(ctx, in)
		if err != nil {
			return err
		}
		im.Log("application: %s version %d (%d workloads, %d resources)", f.Application, saved.VersionNumber, len(saved.Workloads), len(saved.Resources))
	}
	return nil
}

// componentIndex resolves component names to stable IDs across all versions.
type componentIndex struct {
	workloads map[string]string
	resources map[string]string
	variables map[string]string // workloadID/name
	secrets   map[string]string
}

func (im *Importer) index(ctx context.Context, applicationID string) (*componentIndex, error) {
	versions, err := im.Apps.ListVersions(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	idx := &componentIndex{map[string]string{}, map[string]string{}, map[string]string{}, map[string]string{}}
	for _, vs := range versions {
		v, err := im.Apps.FindVersion(ctx, applicationID, vs.ID)
		if err != nil {
			return nil, err
		}
		for _, r := range v.Resources {
			idx.resources[r.Name] = r.ID
		}
		for _, w := range v.Workloads {
			idx.workloads[w.Name] = w.ID
			for _, d := range w.Variables {
				idx.variables[w.ID+"/"+d.Name] = d.ID
			}
			for _, d := range w.Secrets {
				idx.secrets[w.ID+"/"+d.Name] = d.ID
			}
		}
	}
	return idx, nil
}

func (im *Importer) importConfigurations(ctx context.Context, path string) error {
	var f configurationFile
	if err := readYAML(path, &f); err != nil {
		return err
	}
	for _, c := range f.Configurations {
		app, err := im.Apps.FindApplication(ctx, c.Application)
		if err != nil {
			return err
		}
		idx, err := im.index(ctx, app.ID)
		if err != nil {
			return err
		}
		env := domain.Environment(c.Environment)
		if !env.Valid() {
			return fmt.Errorf("environment %q must be STAGING or PRODUCTION", c.Environment)
		}
		// Keep existing generated secret references so re-imports do not rotate them.
		previous, err := im.Configs.FindByApplicationAndEnvironment(ctx, app.ID, env)
		if err != nil {
			return err
		}
		cfg := &domain.EnvironmentConfiguration{ApplicationID: app.ID, Environment: env}
		for _, wname := range sortedKeys(c.Variables) {
			wid, ok := idx.workloads[wname]
			if !ok {
				return fmt.Errorf("workload %q not found", wname)
			}
			for _, name := range sortedKeys(c.Variables[wname]) {
				defID, ok := idx.variables[wid+"/"+name]
				if !ok {
					return fmt.Errorf("variable %s.%s is not declared in any version", wname, name)
				}
				v, err := im.value(idx, c.Variables[wname][name])
				if err != nil {
					return fmt.Errorf("%s.%s: %w", wname, name, err)
				}
				v.WorkloadID, v.DefinitionID, v.Name = wid, defID, name
				cfg.Variables = append(cfg.Variables, v)
			}
		}
		for _, wname := range sortedKeys(c.Secrets) {
			wid, ok := idx.workloads[wname]
			if !ok {
				return fmt.Errorf("workload %q not found", wname)
			}
			for _, name := range sortedKeys(c.Secrets[wname]) {
				defID, ok := idx.secrets[wid+"/"+name]
				if !ok {
					return fmt.Errorf("secret %s.%s is not declared in any version", wname, name)
				}
				b := c.Secrets[wname][name]
				var v domain.ConfiguredValue
				switch {
				case b.Generate:
					v.Source = domain.SourceSecretRef
					if previous != nil {
						for _, ps := range previous.Secrets {
							if ps.DefinitionID == defID && ps.SecretRef != "" {
								v.SecretRef = ps.SecretRef
							}
						}
					}
					if v.SecretRef == "" {
						name := fmt.Sprintf("%s/%s/%s/%s", c.Application, strings.ToLower(c.Environment), wname, strings.ToLower(name))
						ref, err := im.Secrets.Put(ctx, name, []byte(secretstore.RandomValue(24)))
						if err != nil {
							return err
						}
						v.SecretRef = ref
					}
				case b.ResourceOutput != "":
					v, err = im.value(idx, b)
					if err != nil {
						return fmt.Errorf("%s.%s: %w", wname, name, err)
					}
				default:
					return fmt.Errorf("secret %s.%s needs generate or resourceOutput; plaintext secrets are not accepted", wname, name)
				}
				v.WorkloadID, v.DefinitionID, v.Name = wid, defID, name
				cfg.Secrets = append(cfg.Secrets, v)
			}
		}
		if _, err := im.Configs.Save(ctx, cfg); err != nil {
			return err
		}
		im.Log("configuration: %s %s (%d variables, %d secrets)", c.Application, env, len(cfg.Variables), len(cfg.Secrets))
	}
	return nil
}

func (im *Importer) value(idx *componentIndex, b binding) (domain.ConfiguredValue, error) {
	var v domain.ConfiguredValue
	switch {
	case b.Direct != nil:
		v.Source, v.DirectValue = domain.SourceDirect, *b.Direct
	case b.ResourceOutput != "":
		comp, out, ok := strings.Cut(b.ResourceOutput, ".")
		rid, found := idx.resources[comp]
		if !ok || !found {
			return v, fmt.Errorf("resource output %q does not name a resource", b.ResourceOutput)
		}
		v.Source, v.RefID, v.OutputName = domain.SourceResourceOutput, rid, out
	case b.WorkloadOutput != "":
		comp, out, ok := strings.Cut(b.WorkloadOutput, ".")
		wid, found := idx.workloads[comp]
		if !ok || !found {
			return v, fmt.Errorf("workload output %q does not name a workload", b.WorkloadOutput)
		}
		v.Source, v.RefID, v.OutputName = domain.SourceWorkloadOutput, wid, out
	default:
		return v, fmt.Errorf("binding needs direct, resourceOutput or workloadOutput")
	}
	return v, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
