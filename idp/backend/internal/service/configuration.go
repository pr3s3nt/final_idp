package service

import (
	"context"
	"strings"

	"idp/internal/domain"
	"idp/internal/domain/configvalidator"
	"idp/internal/domain/resourceresolver"
	"idp/internal/integration/secretstore"
	"idp/internal/persistence"
)

// EnvironmentConfigurationDraft is the client-owned DTO of UC-02 (ADR-016).
// The browser owns it between edits; the backend keeps no draft and receives it
// complete on Save. It never carries a plaintext Secret: a Secret value is
// staged first and only its opaque reference travels in the draft.
type EnvironmentConfigurationDraft struct {
	ApplicationID string `json:"applicationId"`
	Environment   string `json:"environment"`
	// BaseApplicationDefinitionVersion is the latest version the requirements
	// were loaded from.
	BaseApplicationDefinitionVersion int `json:"baseApplicationDefinitionVersion"`
	// BaseConfigurationRevision is empty when the environment has no
	// configuration yet.
	BaseConfigurationRevision string `json:"baseConfigurationRevision"`
	// CatalogVersion and DeploymentTarget select which Resource Definition
	// applies to each Resource Requirement (ADR-020). Neither is persisted.
	CatalogVersion   string         `json:"catalogVersion"`
	DeploymentTarget string         `json:"deploymentTarget"`
	Variables        []BindingDraft `json:"variables"`
	Secrets          []BindingDraft `json:"secrets"`
}

// BindingDraft is one Environment Variable or Secret with exactly one value
// source.
type BindingDraft struct {
	WorkloadID   string `json:"workloadId"`
	DefinitionID string `json:"definitionId"`
	Name         string `json:"name"`
	// Source is DIRECT, RESOURCE_OUTPUT, WORKLOAD_OUTPUT or SECRET_REF.
	Source string `json:"source"`
	// Value is the direct value of a non-secret variable.
	Value string `json:"value,omitempty"`
	// SecretRef is the opaque reference the Secret Store returned.
	SecretRef string `json:"secretRef,omitempty"`
	// RefID is the referenced Resource Requirement ID or Workload ID.
	RefID      string `json:"refId,omitempty"`
	OutputName string `json:"outputName,omitempty"`
}

// ConfigurationRequirements is what selectEnvironment() returns: what UC-01
// declared, what is configured today, and the lists the Developer chooses a
// Catalog Version and deployment target from.
type ConfigurationRequirements struct {
	ApplicationID                    string                          `json:"applicationId"`
	ApplicationName                  string                          `json:"applicationName"`
	Environment                      string                          `json:"environment"`
	BaseApplicationDefinitionVersion int                             `json:"baseApplicationDefinitionVersion"`
	BaseConfigurationRevision        string                          `json:"baseConfigurationRevision"`
	Workloads                        []ConfigurationWorkload         `json:"workloads"`
	Resources                        []ConfigurationResource         `json:"resources"`
	CatalogVersions                  []domain.CatalogVersion         `json:"catalogVersions"`
	Targets                          []resourceresolver.TargetOption `json:"targets"`
	// Configuration holds the bindings already stored for this environment, or
	// nil when nothing is configured yet.
	Configuration *EnvironmentConfigurationDraft `json:"configuration"`
}

// ConfigurationWorkload lists what one workload needs and what it may
// reference. Only components in DependsOnResources/DependsOnWorkloads may
// supply an output for this workload.
type ConfigurationWorkload struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	Variables          []ConfigRequirementDraft `json:"variables"`
	Secrets            []ConfigRequirementDraft `json:"secrets"`
	DependsOnResources []string                 `json:"dependsOnResources"`
	DependsOnWorkloads []string                 `json:"dependsOnWorkloads"`
}

type ConfigurationResource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// OutputOption is one output a Developer may bind to.
type OutputOption struct {
	Name      string `json:"name"`
	Sensitive bool   `json:"sensitive"`
}

// ResourceOutputsView is the answer to requestResourceOutputs(): the single
// Resource Definition resolved for that requirement and the outputs it exposes.
type ResourceOutputsView struct {
	ResourceID     string         `json:"resourceId"`
	ResourceName   string         `json:"resourceName"`
	DefinitionName string         `json:"definitionName"`
	Outputs        []OutputOption `json:"outputs"`
}

// WorkloadOutputsView is the answer to requestWorkloadOutputs().
type WorkloadOutputsView struct {
	WorkloadID   string         `json:"workloadId"`
	WorkloadName string         `json:"workloadName"`
	Outputs      []OutputOption `json:"outputs"`
}

// ConfigurationService is the stateless Environment Configuration Service of
// UC-02. It loads durable state, answers catalog queries, stages Secret values
// and performs Save; it holds no draft between requests.
type ConfigurationService struct {
	Apps    *persistence.ApplicationRepository
	Configs *persistence.EnvironmentConfigurationRepository
	Catalog *persistence.ResourceDefinitionCatalog
	Secrets secretstore.Store
}

// SelectEnvironment is selectEnvironment(): it loads the requirements of the
// latest Application Definition version, the current Environment Configuration
// and its opaque revision, plus the Catalog Versions and deployment targets the
// Developer chooses from.
func (s *ConfigurationService) SelectEnvironment(ctx context.Context, applicationID, environment string) (*ConfigurationRequirements, error) {
	env, err := parseEnvironment(environment)
	if err != nil {
		return nil, err
	}
	app, err := s.Apps.FindApplication(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	v, err := s.Apps.LatestVersion(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	out := &ConfigurationRequirements{ApplicationID: app.ID, ApplicationName: app.Name, Environment: string(env),
		BaseApplicationDefinitionVersion: v.VersionNumber, Workloads: []ConfigurationWorkload{},
		Resources: []ConfigurationResource{}, Targets: []resourceresolver.TargetOption{}}

	for i := range v.Workloads {
		w := &v.Workloads[i]
		cw := ConfigurationWorkload{ID: w.ID, Name: w.Name, Variables: []ConfigRequirementDraft{}, Secrets: []ConfigRequirementDraft{},
			DependsOnResources: []string{}, DependsOnWorkloads: []string{}}
		for _, d := range w.Variables {
			cw.Variables = append(cw.Variables, ConfigRequirementDraft{ID: d.ID, Name: d.Name, Required: d.Required})
		}
		for _, d := range w.Secrets {
			cw.Secrets = append(cw.Secrets, ConfigRequirementDraft{ID: d.ID, Name: d.Name, Required: d.Required})
		}
		for _, dep := range v.Dependencies {
			if dep.SourceWorkloadID != w.ID {
				continue
			}
			if dep.TargetType == domain.TargetResource {
				cw.DependsOnResources = append(cw.DependsOnResources, dep.TargetID)
			} else {
				cw.DependsOnWorkloads = append(cw.DependsOnWorkloads, dep.TargetID)
			}
		}
		out.Workloads = append(out.Workloads, cw)
	}
	for _, r := range v.Resources {
		out.Resources = append(out.Resources, ConfigurationResource{ID: r.ID, Name: r.Name, Type: r.ResourceType})
	}

	cfg, err := s.Configs.FindByApplicationAndEnvironment(ctx, app.ID, env)
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		revision, err := s.Configs.FindRevision(ctx, app.ID, env)
		if err != nil {
			return nil, err
		}
		out.BaseConfigurationRevision = revision
		// Only bindings the latest version still declares are offered for
		// editing. A binding of a component that a later version removed is not
		// shown and is dropped by the next Save, which replaces the binding set.
		out.Configuration = draftOfConfiguration(declaredIn(v, cfg), v.VersionNumber, revision)
	}

	versions, err := s.Catalog.ListVersions(ctx)
	if err != nil {
		return nil, err
	}
	out.CatalogVersions = versions
	all, err := s.Catalog.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	if len(versions) > 0 {
		newest := &resourceresolver.Resolver{All: all}
		for _, d := range all {
			if d.CatalogVersionID == versions[0].ID {
				newest.Definitions = append(newest.Definitions, d)
			}
		}
		out.Targets = newest.SupportedTargets()
	}
	return out, nil
}

// ResourceOutputs is requestResourceOutputs(): the outputs of the one Resource
// Definition that the selected Catalog Version and deployment target resolve
// for this Resource Requirement. Only a resource the workload depends on can be
// queried.
func (s *ConfigurationService) ResourceOutputs(ctx context.Context, applicationID, environment, workloadID, resourceID, catalogVersion, target string) (*ResourceOutputsView, error) {
	env, err := parseEnvironment(environment)
	if err != nil {
		return nil, err
	}
	app, err := s.Apps.FindApplication(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	v, err := s.Apps.LatestVersion(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	resource := v.Resource(resourceID)
	if resource == nil {
		return nil, domain.Reject(domain.CodeNotFound, "resource %s is not in version %d", resourceID, v.VersionNumber)
	}
	if workloadID != "" && !v.DependsOn(workloadID, resourceID) {
		w := v.Workload(workloadID)
		if w == nil {
			return nil, domain.Reject(domain.CodeNotFound, "workload %s is not in version %d", workloadID, v.VersionNumber)
		}
		return nil, domain.Reject(domain.CodeConfigurationMismatch, "%s does not depend on %s in version %d; add the dependency in UC-01 first",
			w.Name, resource.Name, v.VersionNumber)
	}
	definitions, err := s.resolveDefinitions(ctx, v, app.Name, env, catalogVersion, target)
	if err != nil {
		return nil, err
	}
	d := definitions[resourceID]
	if d == nil {
		return nil, domain.Reject(domain.CodeNoResourceDefinition,
			"the selected catalog version and deployment target do not resolve a resource definition for %s", resource.Name)
	}
	view := &ResourceOutputsView{ResourceID: resource.ID, ResourceName: resource.Name, DefinitionName: d.Name, Outputs: []OutputOption{}}
	for _, name := range d.ExposedOutputs {
		view.Outputs = append(view.Outputs, OutputOption{Name: name, Sensitive: false})
	}
	for _, name := range d.SensitiveOutputs {
		view.Outputs = append(view.Outputs, OutputOption{Name: name, Sensitive: true})
	}
	return view, nil
}

// WorkloadOutputs is requestWorkloadOutputs(): the outputs a depended-on
// workload exposes in the latest version.
func (s *ConfigurationService) WorkloadOutputs(ctx context.Context, applicationID, sourceWorkloadID, workloadID string) (*WorkloadOutputsView, error) {
	app, err := s.Apps.FindApplication(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	v, err := s.Apps.LatestVersion(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	target := v.Workload(workloadID)
	if target == nil {
		return nil, domain.Reject(domain.CodeNotFound, "workload %s is not in version %d", workloadID, v.VersionNumber)
	}
	if sourceWorkloadID != "" && !v.DependsOn(sourceWorkloadID, workloadID) {
		source := v.Workload(sourceWorkloadID)
		if source == nil {
			return nil, domain.Reject(domain.CodeNotFound, "workload %s is not in version %d", sourceWorkloadID, v.VersionNumber)
		}
		return nil, domain.Reject(domain.CodeConfigurationMismatch, "%s does not depend on %s in version %d; add the dependency in UC-01 first",
			source.Name, target.Name, v.VersionNumber)
	}
	view := &WorkloadOutputsView{WorkloadID: target.ID, WorkloadName: target.Name, Outputs: []OutputOption{}}
	for _, name := range target.ExposedOutputs {
		view.Outputs = append(view.Outputs, OutputOption{Name: name, Sensitive: false})
	}
	return view, nil
}

// StageSecret is the secure part of setDirectConfigurationValue(): the
// plaintext value goes to the Secret Store and only the opaque reference
// returns to the browser. Each staging writes its own reference, so a value
// that was already saved is never overwritten before Save; cleanup of
// references that Save never stores is owned by D07.
func (s *ConfigurationService) StageSecret(ctx context.Context, applicationID, environment, workloadID, definitionID, value string) (string, error) {
	env, err := parseEnvironment(environment)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", domain.Reject(domain.CodeMissingConfiguration, "the secret value is empty")
	}
	if s.Secrets == nil {
		return "", domain.Reject(domain.CodeInvalidInput, "no secret store is configured")
	}
	app, err := s.Apps.FindApplication(ctx, applicationID)
	if err != nil {
		return "", err
	}
	v, err := s.Apps.LatestVersion(ctx, app.ID)
	if err != nil {
		return "", err
	}
	w := v.Workload(workloadID)
	if w == nil {
		return "", domain.Reject(domain.CodeNotFound, "workload %s is not in version %d", workloadID, v.VersionNumber)
	}
	declared := false
	for _, d := range w.Secrets {
		if d.ID == definitionID {
			declared = true
			break
		}
	}
	if !declared {
		return "", domain.Reject(domain.CodeConfigurationMismatch, "%s does not declare that secret in version %d", w.Name, v.VersionNumber)
	}
	name := strings.Join([]string{app.ID, strings.ToLower(string(env)), workloadID, definitionID, secretstore.RandomValue(8)}, "/")
	return s.Secrets.Put(ctx, name, []byte(value))
}

// SaveEnvironmentConfiguration is saveEnvironmentConfiguration(): it validates
// the complete draft against the latest version and the resolved definitions,
// then writes it only when both concurrency bases still match (contract 3).
func (s *ConfigurationService) SaveEnvironmentConfiguration(ctx context.Context, applicationID, environment string,
	draft EnvironmentConfigurationDraft) (*EnvironmentConfigurationDraft, error) {

	env, err := parseEnvironment(environment)
	if err != nil {
		return nil, err
	}
	if draft.Environment != "" && !strings.EqualFold(draft.Environment, string(env)) {
		return nil, domain.Reject(domain.CodeInvalidInput, "environment in the draft does not match the request path")
	}
	app, err := s.Apps.FindApplication(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	if draft.ApplicationID != "" && !strings.EqualFold(draft.ApplicationID, app.ID) {
		return nil, domain.Reject(domain.CodeInvalidInput, "applicationId in the draft does not match the request path")
	}
	v, err := s.Apps.LatestVersion(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	definitions, err := s.resolveDefinitions(ctx, v, app.Name, env, draft.CatalogVersion, draft.DeploymentTarget)
	if err != nil {
		return nil, err
	}

	cfg := &domain.EnvironmentConfiguration{ApplicationID: app.ID, Environment: env}
	for _, b := range draft.Variables {
		cfg.Variables = append(cfg.Variables, configuredValueOf(b))
	}
	for _, b := range draft.Secrets {
		cfg.Secrets = append(cfg.Secrets, configuredValueOf(b))
	}
	if err := configvalidator.Validate(configvalidator.Input{Version: v, Configuration: cfg, Definitions: definitions}); err != nil {
		return nil, err
	}

	id, revision, err := s.Configs.SaveIfBasesMatch(ctx, persistence.SaveConfigurationInput{Configuration: cfg,
		BaseApplicationDefinitionVersion: draft.BaseApplicationDefinitionVersion,
		BaseConfigurationRevision:        draft.BaseConfigurationRevision})
	if err != nil {
		return nil, err
	}
	saved, err := s.Configs.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	out := draftOfConfiguration(saved, v.VersionNumber, revision)
	out.CatalogVersion, out.DeploymentTarget = draft.CatalogVersion, draft.DeploymentTarget
	return out, nil
}

// resolveDefinitions resolves every Resource Requirement of the version to the
// single Resource Definition selected by the Catalog Version and deployment
// target, using the rule UC-03 uses. A requirement that resolves to nothing is
// left out; the validator reports it when a binding needs it.
func (s *ConfigurationService) resolveDefinitions(ctx context.Context, v *domain.ApplicationVersion, applicationName string,
	env domain.Environment, catalogVersion, target string) (map[string]*domain.ResourceDefinition, error) {

	if target == "" {
		return nil, domain.Reject(domain.CodeUnsupportedTarget, "select a deployment target before choosing resource outputs")
	}
	version, err := s.catalogVersion(ctx, catalogVersion)
	if err != nil {
		return nil, err
	}
	all, err := s.Catalog.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	r := &resourceresolver.Resolver{All: all}
	for _, d := range all {
		if d.CatalogVersionID == version.ID {
			r.Definitions = append(r.Definitions, d)
		}
	}
	deploymentContext, err := resolveContext(r, target, "", "", nil)
	if err != nil {
		return nil, err
	}
	scope := resourceresolver.Scope{ApplicationID: v.ApplicationID, ApplicationName: applicationName, Environment: env, Context: deploymentContext}
	out := map[string]*domain.ResourceDefinition{}
	for _, req := range v.Resources {
		resolution, problem := r.Resolve(req.ID, req.ResourceType, scope)
		if problem != nil {
			continue // reported per binding by the validator
		}
		out[req.ID] = resolution.Definition
	}
	return out, nil
}

// catalogVersion returns the requested Catalog Version, or the newest one when
// the draft names none.
func (s *ConfigurationService) catalogVersion(ctx context.Context, key string) (*domain.CatalogVersion, error) {
	if key != "" {
		return s.Catalog.FindVersion(ctx, key)
	}
	versions, err := s.Catalog.ListVersions(ctx)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, domain.Reject(domain.CodeNotFound, "the catalog has no version")
	}
	return &versions[0], nil
}

// declaredIn returns the configuration without the bindings whose workload or
// definition is not part of the given version.
func declaredIn(v *domain.ApplicationVersion, c *domain.EnvironmentConfiguration) *domain.EnvironmentConfiguration {
	variables, secrets := map[string]bool{}, map[string]bool{}
	for i := range v.Workloads {
		w := &v.Workloads[i]
		for _, d := range w.Variables {
			variables[w.ID+"/"+d.ID] = true
		}
		for _, d := range w.Secrets {
			secrets[w.ID+"/"+d.ID] = true
		}
	}
	out := &domain.EnvironmentConfiguration{ID: c.ID, ApplicationID: c.ApplicationID, Environment: c.Environment}
	for _, b := range c.Variables {
		if variables[b.WorkloadID+"/"+b.DefinitionID] {
			out.Variables = append(out.Variables, b)
		}
	}
	for _, b := range c.Secrets {
		if secrets[b.WorkloadID+"/"+b.DefinitionID] {
			out.Secrets = append(out.Secrets, b)
		}
	}
	return out
}

func configuredValueOf(b BindingDraft) domain.ConfiguredValue {
	return domain.ConfiguredValue{WorkloadID: b.WorkloadID, DefinitionID: b.DefinitionID, Name: b.Name, Source: b.Source,
		DirectValue: b.Value, SecretRef: b.SecretRef, RefID: b.RefID, OutputName: b.OutputName}
}

// draftOfConfiguration turns a stored configuration into the editable DTO.
func draftOfConfiguration(c *domain.EnvironmentConfiguration, baseVersion int, revision string) *EnvironmentConfigurationDraft {
	d := &EnvironmentConfigurationDraft{ApplicationID: c.ApplicationID, Environment: string(c.Environment),
		BaseApplicationDefinitionVersion: baseVersion, BaseConfigurationRevision: revision,
		Variables: []BindingDraft{}, Secrets: []BindingDraft{}}
	for _, b := range c.Variables {
		d.Variables = append(d.Variables, bindingDraftOf(b))
	}
	for _, b := range c.Secrets {
		d.Secrets = append(d.Secrets, bindingDraftOf(b))
	}
	return d
}

func bindingDraftOf(b domain.ConfiguredValue) BindingDraft {
	return BindingDraft{WorkloadID: b.WorkloadID, DefinitionID: b.DefinitionID, Name: b.Name, Source: b.Source,
		Value: b.DirectValue, SecretRef: b.SecretRef, RefID: b.RefID, OutputName: b.OutputName}
}

func parseEnvironment(value string) (domain.Environment, error) {
	env := domain.Environment(strings.ToUpper(strings.TrimSpace(value)))
	if !env.Valid() {
		return "", domain.Reject(domain.CodeInvalidInput, "environment %q must be staging or production", value)
	}
	return env, nil
}
