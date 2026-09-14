package catalog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/fingerprint"
)

type Resolver struct{}

func NewResolver() *Resolver { return &Resolver{} }

func (r *Resolver) Resolve(snapshot domain.DeploymentInputSnapshot, graph domain.DeploymentGraph, definitions []domain.ResourceDefinition) ([]domain.ResourceResolution, error) {
	graphResources := make(map[string]struct{})
	for _, node := range graph.Nodes {
		if node.Kind == domain.GraphNodeResource {
			graphResources[node.ID] = struct{}{}
		}
	}
	resolutions := make([]domain.ResourceResolution, 0, len(graphResources))
	for _, requirement := range snapshot.ApplicationDefinition.ResourceRequirements {
		if _, ok := graphResources[requirement.ID]; !ok {
			continue
		}
		matches := make([]domain.ResourceDefinition, 0, 1)
		for _, definition := range definitions {
			if definition.Retired || definition.ResourceType != requirement.ResourceType {
				continue
			}
			if supports(definition.SupportedContexts, snapshot.RenderContext) {
				matches = append(matches, definition)
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("RESOURCE_DEFINITION_NOT_FOUND: requirement %s type %s has no definition for %s/%s", requirement.ID, requirement.ResourceType, snapshot.RenderContext.CloudProvider, snapshot.RenderContext.Region)
		}
		if len(matches) > 1 {
			ids := make([]string, 0, len(matches))
			for _, definition := range matches {
				ids = append(ids, definition.ID)
			}
			sort.Strings(ids)
			return nil, fmt.Errorf("RESOURCE_DEFINITION_AMBIGUOUS: requirement %s matches %s", requirement.ID, strings.Join(ids, ","))
		}
		definition := matches[0]
		if definition.ID == "" || definition.ProvisionerReference == "" {
			return nil, fmt.Errorf("INVALID_RESOURCE_DEFINITION: %s has no identity or provisioner reference", definition.Name)
		}
		if len(definition.AllowedOverrides) != 0 {
			return nil, fmt.Errorf("UNSUPPORTED_MVP_OPERATION: definition %s enables overrides", definition.ID)
		}
		if _, err := DefinitionFingerprint(definition); err != nil {
			return nil, fmt.Errorf("INVALID_RESOURCE_DEFINITION: %s: %w", definition.ID, err)
		}
		resolutions = append(resolutions, domain.ResourceResolution{Requirement: requirement, Definition: definition})
	}
	if err := validateReferencedOutputs(snapshot, resolutions); err != nil {
		return nil, err
	}
	sort.Slice(resolutions, func(i, j int) bool { return resolutions[i].Requirement.ID < resolutions[j].Requirement.ID })
	return resolutions, nil
}

func supports(selectors []domain.ContextSelector, context domain.DeploymentContext) bool {
	for _, selector := range selectors {
		if !strings.EqualFold(selector.CloudProvider, context.CloudProvider) {
			continue
		}
		if selector.Region == "" || selector.Region == context.Region {
			return true
		}
	}
	return false
}

func DefinitionFingerprint(definition domain.ResourceDefinition) (string, error) {
	selectors := append([]domain.ContextSelector(nil), definition.SupportedContexts...)
	sort.Slice(selectors, func(i, j int) bool {
		if selectors[i].CloudProvider != selectors[j].CloudProvider {
			return selectors[i].CloudProvider < selectors[j].CloudProvider
		}
		return selectors[i].Region < selectors[j].Region
	})
	payload := struct {
		ID                   string                   `json:"id"`
		ResourceType         string                   `json:"resourceType"`
		ProvisionerReference string                   `json:"provisionerReference"`
		SupportedContexts    []domain.ContextSelector `json:"supportedContexts"`
		DefaultParameters    map[string]any           `json:"defaultParameters"`
		AllowedOverrides     map[string]any           `json:"allowedOverrides"`
		ExposedOutputs       []string                 `json:"exposedOutputs"`
		SensitiveOutputs     []string                 `json:"sensitiveOutputs"`
	}{definition.ID, definition.ResourceType, definition.ProvisionerReference, selectors, definition.DefaultParameters, definition.AllowedOverrides, fingerprint.SortedStrings(definition.ExposedOutputs), fingerprint.SortedStrings(definition.SensitiveOutputs)}
	return fingerprint.Sum(payload)
}

func validateReferencedOutputs(snapshot domain.DeploymentInputSnapshot, resolutions []domain.ResourceResolution) error {
	byRequirement := make(map[string]domain.ResourceDefinition, len(resolutions))
	for _, resolution := range resolutions {
		byRequirement[resolution.Requirement.ID] = resolution.Definition
	}
	for _, value := range snapshot.EnvironmentConfiguration.Values {
		if value.Source != domain.ValueResourceOutput {
			continue
		}
		definition, ok := byRequirement[value.ResourceRequirementID]
		if !ok {
			return fmt.Errorf("RESOURCE_DEFINITION_NOT_FOUND: configuration %s references unresolved requirement", value.ID)
		}
		found := false
		for _, output := range definition.ExposedOutputs {
			if output == value.ResourceOutputName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("RESOURCE_OUTPUT_NOT_EXPOSED: definition %s does not expose %s", definition.ID, value.ResourceOutputName)
		}
	}
	for _, secret := range snapshot.EnvironmentConfiguration.Secrets {
		if secret.Source != domain.SecretResource {
			continue
		}
		definition, ok := byRequirement[secret.ResourceRequirementID]
		if !ok {
			return fmt.Errorf("RESOURCE_DEFINITION_NOT_FOUND: secret %s references unresolved requirement", secret.ID)
		}
		if !contains(definition.ExposedOutputs, secret.ResourceOutputName) {
			return fmt.Errorf("RESOURCE_OUTPUT_NOT_EXPOSED: definition %s does not expose %s", definition.ID, secret.ResourceOutputName)
		}
	}
	return nil
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
