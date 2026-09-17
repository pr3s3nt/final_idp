package service

import (
	"context"
	"strings"

	"idp/internal/domain"
	"idp/internal/domain/appspec"
	"idp/internal/domain/appvalidator"
	"idp/internal/persistence"
)

// ApplicationDefinitionDraft is the client-owned DTO of UC-01 (ADR-016). The
// browser sends it complete on Save; the backend keeps no draft between
// requests. The same shape, with BaseVersion set to the latest version, is
// returned to start an edit.
type ApplicationDefinitionDraft struct {
	ApplicationID string            `json:"applicationId,omitempty"`
	BaseVersion   *int              `json:"baseVersion"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Workloads     []WorkloadDraft   `json:"workloads"`
	Resources     []ResourceDraft   `json:"resources"`
	Dependencies  []DependencyDraft `json:"dependencies"`
}

type WorkloadDraft struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Type            string                   `json:"type"`
	ImageRepository string                   `json:"imageRepository"`
	Port            *int                     `json:"port"`
	Outputs         []string                 `json:"outputs"`
	Variables       []ConfigRequirementDraft `json:"variables"`
	Secrets         []ConfigRequirementDraft `json:"secrets"`
}

// ConfigRequirementDraft declares an Environment Variable or Secret by name.
// It has no value field: UC-01 never receives a Secret value.
type ConfigRequirementDraft struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

type ResourceDraft struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type DependencyDraft struct {
	ID       string `json:"id"`
	SourceID string `json:"sourceId"`
	TargetID string `json:"targetId"`
}

// ApplicationListItem is one row of the Application Definition list.
type ApplicationListItem struct {
	ApplicationID string `json:"applicationId"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	LatestVersion int    `json:"latestVersion"`
}

// ApplicationService is the stateless UC-01 Application Service.
type ApplicationService struct {
	Apps  *persistence.ApplicationRepository
	Specs *persistence.SpecificationRepository
}

func (s *ApplicationService) ListApplications(ctx context.Context) ([]ApplicationListItem, error) {
	apps, err := s.Apps.ListApplications(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ApplicationListItem, len(apps))
	for i, a := range apps {
		out[i] = ApplicationListItem{ApplicationID: a.ID, Name: a.Name, Description: a.Description, LatestVersion: a.LatestVersion}
	}
	return out, nil
}

// UpdateApplication is updateApplication(): load the latest version as an
// editable draft whose BaseVersion is that version.
func (s *ApplicationService) UpdateApplication(ctx context.Context, applicationID string) (*ApplicationDefinitionDraft, error) {
	v, err := s.Apps.LatestVersion(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	return DraftOf(v), nil
}

// SaveApplicationDefinition is saveApplicationDefinition(): validate the
// complete draft, save it as a new version if its base still matches, then
// generateApplicationSpecification() for that version. applicationID is empty
// when the draft creates an application.
func (s *ApplicationService) SaveApplicationDefinition(ctx context.Context, applicationID string, draft ApplicationDefinitionDraft) (*ApplicationDefinitionDraft, error) {
	in, err := ValidateDraft(applicationID, draft)
	if err != nil {
		return nil, err
	}
	v, err := s.Apps.SaveNewVersionIfBaseMatches(ctx, *in)
	if err != nil {
		return nil, err
	}
	spec, err := appspec.Generate(v)
	if err != nil {
		return nil, err
	}
	if err := s.Specs.Save(ctx, v.ApplicationID, v.VersionID, spec.Format, spec.Content, spec.Version); err != nil {
		return nil, err
	}
	return DraftOf(v), nil
}

// ValidateDraft checks request-level rules and runs the Application
// Definition Validator. It needs no stored state.
func ValidateDraft(applicationID string, d ApplicationDefinitionDraft) (*persistence.SaveVersionInput, error) {
	if applicationID == "" {
		if d.BaseVersion != nil || d.ApplicationID != "" {
			return nil, domain.Reject(domain.CodeInvalidInput, "a new application has no applicationId or baseVersion")
		}
	} else {
		if d.BaseVersion == nil || *d.BaseVersion < 1 {
			return nil, domain.Reject(domain.CodeInvalidInput, "baseVersion is required when saving an existing application")
		}
		if d.ApplicationID != "" && !strings.EqualFold(d.ApplicationID, applicationID) {
			return nil, domain.Reject(domain.CodeInvalidInput, "applicationId in the draft does not match the request path")
		}
	}

	def := appvalidator.Definition{Name: d.Name, Description: d.Description}
	for _, w := range d.Workloads {
		dw := domain.Workload{ID: w.ID, Name: w.Name, Type: w.Type, ImageRepository: w.ImageRepository, Port: w.Port, ExposedOutputs: w.Outputs}
		for _, c := range w.Variables {
			dw.Variables = append(dw.Variables, domain.ConfigDefinition{ID: c.ID, Name: c.Name, Required: c.Required})
		}
		for _, c := range w.Secrets {
			dw.Secrets = append(dw.Secrets, domain.ConfigDefinition{ID: c.ID, Name: c.Name, Required: c.Required})
		}
		def.Workloads = append(def.Workloads, dw)
	}
	for _, r := range d.Resources {
		def.Resources = append(def.Resources, domain.ResourceRequirement{ID: r.ID, Name: r.Name, ResourceType: r.Type})
	}
	for _, dep := range d.Dependencies {
		def.Dependencies = append(def.Dependencies, appvalidator.Dependency{Key: dep.ID, SourceID: dep.SourceID, TargetID: dep.TargetID})
	}
	deps, err := appvalidator.Validate(def)
	if err != nil {
		return nil, err
	}
	in := &persistence.SaveVersionInput{ApplicationID: applicationID, Name: d.Name, Description: d.Description,
		Workloads: def.Workloads, Resources: def.Resources, Dependencies: deps}
	if d.BaseVersion != nil {
		in.BaseVersion = *d.BaseVersion
	}
	return in, nil
}

// DraftOf converts a stored version into the editable DTO.
func DraftOf(v *domain.ApplicationVersion) *ApplicationDefinitionDraft {
	base := v.VersionNumber
	d := &ApplicationDefinitionDraft{ApplicationID: v.ApplicationID, BaseVersion: &base, Name: v.ApplicationName,
		Description: v.Description, Workloads: []WorkloadDraft{}, Resources: []ResourceDraft{}, Dependencies: []DependencyDraft{}}
	for _, w := range v.Workloads {
		wd := WorkloadDraft{ID: w.ID, Name: w.Name, Type: w.Type, ImageRepository: w.ImageRepository, Port: w.Port,
			Outputs: append([]string{}, w.ExposedOutputs...), Variables: []ConfigRequirementDraft{}, Secrets: []ConfigRequirementDraft{}}
		for _, c := range w.Variables {
			wd.Variables = append(wd.Variables, ConfigRequirementDraft{ID: c.ID, Name: c.Name, Required: c.Required})
		}
		for _, c := range w.Secrets {
			wd.Secrets = append(wd.Secrets, ConfigRequirementDraft{ID: c.ID, Name: c.Name, Required: c.Required})
		}
		d.Workloads = append(d.Workloads, wd)
	}
	for _, r := range v.Resources {
		d.Resources = append(d.Resources, ResourceDraft{ID: r.ID, Name: r.Name, Type: r.ResourceType})
	}
	for _, dep := range v.Dependencies {
		d.Dependencies = append(d.Dependencies, DependencyDraft{ID: dep.ID, SourceID: dep.SourceWorkloadID, TargetID: dep.TargetID})
	}
	return d
}
