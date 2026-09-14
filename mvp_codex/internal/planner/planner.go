package planner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/catalog"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/fingerprint"
)

type Planner struct{ StateRoot string }

func New(stateRoot string) *Planner { return &Planner{StateRoot: strings.TrimSuffix(stateRoot, "/")} }

func (p *Planner) Plan(snapshot domain.DeploymentInputSnapshot, resolutions []domain.ResourceResolution, instances map[string]domain.ResourceInstance) (domain.InfrastructurePlan, error) {
	if snapshot.InputFingerprint == "" {
		return domain.InfrastructurePlan{}, fmt.Errorf("INVALID_SNAPSHOT: input fingerprint is required")
	}
	items := make([]domain.InfrastructurePlanItem, 0, len(resolutions))
	for _, resolution := range resolutions {
		definitionFP, err := catalog.DefinitionFingerprint(resolution.Definition)
		if err != nil {
			return domain.InfrastructurePlan{}, err
		}
		parameters := cloneMap(resolution.Definition.DefaultParameters)
		parametersFP, err := fingerprint.Sum(parameters)
		if err != nil {
			return domain.InfrastructurePlan{}, fmt.Errorf("INVALID_RESOURCE_PARAMETERS: %s: %w", resolution.Requirement.ID, err)
		}
		instance, exists := instances[resolution.Requirement.ID]
		action := domain.PlanCreate
		instanceID := deterministicInstanceID(snapshot, resolution.Requirement.ID)
		version := int64(0)
		stateReference := p.stateReference(snapshot, resolution.Requirement.ID)
		if exists {
			if err := validateScope(instance, snapshot, resolution.Requirement.ID); err != nil {
				return domain.InfrastructurePlan{}, err
			}
			instanceID, version, stateReference = instance.ID, instance.Version, instance.ProviderStateReference
			switch instance.Status {
			case domain.ResourceReady:
				if instance.DefinitionFingerprint != definitionFP || instance.ParametersFingerprint != parametersFP || instance.ResourceDefinitionID != resolution.Definition.ID {
					return domain.InfrastructurePlan{}, fmt.Errorf("RESOURCE_CHANGE_UNSUPPORTED: ready instance %s is incompatible", instance.ID)
				}
				action = domain.PlanReuse
			case domain.ResourcePlanned:
				if !instance.RecoveryVerified {
					return domain.InfrastructurePlan{}, fmt.Errorf("RESOURCE_RECOVERY_REQUIRED: planned instance %s has not been inspected", instance.ID)
				}
				if instance.DefinitionFingerprint != definitionFP || instance.ParametersFingerprint != parametersFP {
					return domain.InfrastructurePlan{}, fmt.Errorf("RESOURCE_CHANGE_UNSUPPORTED: retained instance %s is incompatible", instance.ID)
				}
				action = domain.PlanCreate
			case domain.ResourceProvisioning, domain.ResourceFailed:
				return domain.InfrastructurePlan{}, fmt.Errorf("RESOURCE_RECOVERY_REQUIRED: instance %s is %s", instance.ID, instance.Status)
			default:
				return domain.InfrastructurePlan{}, fmt.Errorf("RESOURCE_CHANGE_UNSUPPORTED: instance %s is %s", instance.ID, instance.Status)
			}
		}
		if stateReference == "" {
			return domain.InfrastructurePlan{}, fmt.Errorf("INVALID_PROVIDER_STATE_REFERENCE: requirement %s", resolution.Requirement.ID)
		}
		items = append(items, domain.InfrastructurePlanItem{
			ResourceRequirementID:        resolution.Requirement.ID,
			ResourceDefinitionID:         resolution.Definition.ID,
			ProvisionerReference:         resolution.Definition.ProvisionerReference,
			Action:                       action,
			ResolvedParameters:           parameters,
			ExposedOutputs:               fingerprint.SortedStrings(resolution.Definition.ExposedOutputs),
			ReferencedResourceInstanceID: instanceID,
			ReferencedInstanceVersion:    version,
			ProviderStateReference:       stateReference,
			DefinitionFingerprint:        definitionFP,
			ParametersFingerprint:        parametersFP,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ResourceRequirementID < items[j].ResourceRequirementID })
	plan := domain.InfrastructurePlan{ApplicationID: snapshot.ApplicationDefinition.ID, Environment: snapshot.EnvironmentConfiguration.Environment, DeploymentTarget: snapshot.RenderContext.TargetID, InputFingerprint: snapshot.InputFingerprint, Items: items, Algorithm: domain.PlanFingerprintAlgo}
	fp, err := planFingerprint(plan)
	if err != nil {
		return domain.InfrastructurePlan{}, err
	}
	plan.Fingerprint = fp
	return plan, nil
}

func planFingerprint(plan domain.InfrastructurePlan) (string, error) {
	plan.Fingerprint = ""
	return fingerprint.Sum(plan)
}

func deterministicInstanceID(snapshot domain.DeploymentInputSnapshot, requirementID string) string {
	sum, _ := fingerprint.Sum([]string{snapshot.ApplicationDefinition.ID, snapshot.EnvironmentConfiguration.Environment, requirementID, snapshot.RenderContext.TargetID})
	// A deterministic UUID-shaped identity keeps resource reservations stable
	// across reviewed attempts while remaining compatible with the SQL model.
	return sum[:8] + "-" + sum[8:12] + "-5" + sum[13:16] + "-8" + sum[17:20] + "-" + sum[20:32]
}

func (p *Planner) stateReference(snapshot domain.DeploymentInputSnapshot, requirementID string) string {
	id := deterministicInstanceID(snapshot, requirementID)
	return p.StateRoot + "/" + id + "/terraform.tfstate"
}

func validateScope(instance domain.ResourceInstance, snapshot domain.DeploymentInputSnapshot, requirementID string) error {
	if instance.OwnerApplicationID != snapshot.ApplicationDefinition.ID || instance.Environment != snapshot.EnvironmentConfiguration.Environment || instance.ResourceRequirementID != requirementID || instance.DeploymentTarget != snapshot.RenderContext.TargetID {
		return fmt.Errorf("INVALID_RESOURCE_BINDING: instance %s does not match exact scope", instance.ID)
	}
	return nil
}

func cloneMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for k, v := range input {
		result[k] = v
	}
	return result
}
