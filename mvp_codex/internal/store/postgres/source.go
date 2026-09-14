package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
)

type CreateInput struct {
	ApplicationID string
	Environment   string
	Context       domain.DeploymentContext
	Images        []domain.WorkloadImage
}

func (s *Store) LoadSource(ctx context.Context, input CreateInput) (domain.DeploymentInputSnapshot, []domain.ResourceDefinition, map[string]domain.ResourceInstance, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return domain.DeploymentInputSnapshot{}, nil, nil, err
	}
	defer tx.Rollback(ctx)
	snapshot, err := loadSourceTx(ctx, tx, input.ApplicationID, input.Environment)
	if err != nil {
		return domain.DeploymentInputSnapshot{}, nil, nil, err
	}
	snapshot.RenderContext = input.Context
	snapshot.Images = input.Images
	definitions, err := loadDefinitionsTx(ctx, tx)
	if err != nil {
		return domain.DeploymentInputSnapshot{}, nil, nil, err
	}
	instances, err := loadInstancesTx(ctx, tx, snapshot)
	if err != nil {
		return domain.DeploymentInputSnapshot{}, nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.DeploymentInputSnapshot{}, nil, nil, err
	}
	return snapshot, definitions, instances, nil
}

func loadSourceTx(ctx context.Context, tx pgx.Tx, applicationID, environment string) (domain.DeploymentInputSnapshot, error) {
	var app domain.ApplicationDefinition
	err := tx.QueryRow(ctx, `SELECT application_id::text,name FROM application_definition WHERE application_id=$1 AND retired_at IS NULL`, applicationID).Scan(&app.ID, &app.Name)
	if err != nil {
		return domain.DeploymentInputSnapshot{}, fmt.Errorf("load application: %w", err)
	}

	rows, err := tx.Query(ctx, `SELECT workload_id::text,name,type,image_repository,COALESCE(port,0),exposed_outputs FROM workload WHERE application_id=$1 AND retired_at IS NULL ORDER BY workload_id`, applicationID)
	if err != nil {
		return domain.DeploymentInputSnapshot{}, err
	}
	for rows.Next() {
		var workload domain.Workload
		var outputs []byte
		if err := rows.Scan(&workload.ID, &workload.Name, &workload.Type, &workload.ImageRepository, &workload.Port, &outputs); err != nil {
			rows.Close()
			return domain.DeploymentInputSnapshot{}, err
		}
		if err := decodeJSON(outputs, &workload.ExposedOutputs); err != nil {
			rows.Close()
			return domain.DeploymentInputSnapshot{}, err
		}
		app.Workloads = append(app.Workloads, workload)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return domain.DeploymentInputSnapshot{}, err
	}

	rows, err = tx.Query(ctx, `SELECT resource_requirement_id::text,name,resource_type FROM resource_requirement WHERE application_id=$1 AND retired_at IS NULL ORDER BY resource_requirement_id`, applicationID)
	if err != nil {
		return domain.DeploymentInputSnapshot{}, err
	}
	for rows.Next() {
		var requirement domain.ResourceRequirement
		if err := rows.Scan(&requirement.ID, &requirement.Name, &requirement.ResourceType); err != nil {
			rows.Close()
			return domain.DeploymentInputSnapshot{}, err
		}
		app.ResourceRequirements = append(app.ResourceRequirements, requirement)
	}
	rows.Close()

	rows, err = tx.Query(ctx, `SELECT dependency_id::text,source_workload_id::text,target_type,COALESCE(target_workload_id::text,''),COALESCE(target_resource_requirement_id::text,'') FROM dependency WHERE application_id=$1 ORDER BY dependency_id`, applicationID)
	if err != nil {
		return domain.DeploymentInputSnapshot{}, err
	}
	for rows.Next() {
		var dependency domain.Dependency
		if err := rows.Scan(&dependency.ID, &dependency.SourceWorkloadID, &dependency.TargetType, &dependency.TargetWorkloadID, &dependency.TargetResourceRequirementID); err != nil {
			rows.Close()
			return domain.DeploymentInputSnapshot{}, err
		}
		app.Dependencies = append(app.Dependencies, dependency)
	}
	rows.Close()

	var config domain.EnvironmentConfiguration
	err = tx.QueryRow(ctx, `SELECT environment_configuration_id::text,environment FROM environment_configuration WHERE application_id=$1 AND environment=$2`, applicationID, environment).Scan(&config.ID, &config.Environment)
	if err != nil {
		return domain.DeploymentInputSnapshot{}, fmt.Errorf("load configuration: %w", err)
	}
	rows, err = tx.Query(ctx, `SELECT configuration_value_id::text,workload_id::text,name,value_source,COALESCE(direct_value,''),COALESCE(resource_requirement_id::text,''),COALESCE(resource_output_name,''),COALESCE(referenced_workload_id::text,''),COALESCE(workload_output_name,'') FROM configuration_value WHERE environment_configuration_id=$1 ORDER BY configuration_value_id`, config.ID)
	if err != nil {
		return domain.DeploymentInputSnapshot{}, err
	}
	for rows.Next() {
		var value domain.ConfigurationValue
		if err := rows.Scan(&value.ID, &value.WorkloadID, &value.Name, &value.Source, &value.DirectValue, &value.ResourceRequirementID, &value.ResourceOutputName, &value.ReferencedWorkloadID, &value.WorkloadOutputName); err != nil {
			rows.Close()
			return domain.DeploymentInputSnapshot{}, err
		}
		config.Values = append(config.Values, value)
	}
	rows.Close()
	rows, err = tx.Query(ctx, `SELECT secret_id::text,workload_id::text,name,source_type,deployment_target,namespace,secret_name,COALESCE(secret_uid,''),secret_key,COALESCE(resource_requirement_id::text,''),COALESCE(resource_output_name,''),COALESCE(remote_property,'') FROM secret_reference WHERE environment_configuration_id=$1 ORDER BY secret_id`, config.ID)
	if err != nil {
		return domain.DeploymentInputSnapshot{}, err
	}
	for rows.Next() {
		var secret domain.SecretReference
		if err := rows.Scan(&secret.ID, &secret.WorkloadID, &secret.Name, &secret.Source, &secret.Target, &secret.Namespace, &secret.SecretName, &secret.UID, &secret.Key, &secret.ResourceRequirementID, &secret.ResourceOutputName, &secret.RemoteProperty); err != nil {
			rows.Close()
			return domain.DeploymentInputSnapshot{}, err
		}
		config.Secrets = append(config.Secrets, secret)
	}
	rows.Close()
	return domain.DeploymentInputSnapshot{SchemaVersion: domain.SnapshotSchemaVersion, ApplicationDefinition: app, EnvironmentConfiguration: config, CreatedAt: time.Now().UTC()}, nil
}

func loadDefinitionsTx(ctx context.Context, tx pgx.Tx) ([]domain.ResourceDefinition, error) {
	rows, err := tx.Query(ctx, `SELECT resource_definition_id::text,name,resource_type,provisioner_reference,supported_contexts,default_parameters,allowed_overrides,exposed_outputs,sensitive_outputs FROM resource_definition WHERE retired_at IS NULL ORDER BY resource_definition_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.ResourceDefinition
	for rows.Next() {
		var definition domain.ResourceDefinition
		var contexts, parameters, overrides, outputs, sensitive []byte
		if err := rows.Scan(&definition.ID, &definition.Name, &definition.ResourceType, &definition.ProvisionerReference, &contexts, &parameters, &overrides, &outputs, &sensitive); err != nil {
			return nil, err
		}
		if err := decodeJSON(contexts, &definition.SupportedContexts); err != nil {
			return nil, err
		}
		if err := decodeJSON(parameters, &definition.DefaultParameters); err != nil {
			return nil, err
		}
		if err := decodeJSON(overrides, &definition.AllowedOverrides); err != nil {
			return nil, err
		}
		if err := decodeJSON(outputs, &definition.ExposedOutputs); err != nil {
			return nil, err
		}
		if err := decodeJSON(sensitive, &definition.SensitiveOutputs); err != nil {
			return nil, err
		}
		result = append(result, definition)
	}
	return result, rows.Err()
}

func loadInstancesTx(ctx context.Context, tx pgx.Tx, snapshot domain.DeploymentInputSnapshot) (map[string]domain.ResourceInstance, error) {
	rows, err := tx.Query(ctx, `SELECT ri.resource_instance_id::text,ri.resource_definition_id::text,ri.owner_application_id::text,ri.environment,ri.resource_requirement_id::text,ri.deployment_target,ri.provider_state_reference,COALESCE(ri.infrastructure_reference,''),ri.definition_fingerprint,ri.parameters_fingerprint,ri.version,ri.recovery_verified,ri.status
		FROM resource_instance_binding b JOIN resource_instance ri ON ri.resource_instance_id=b.resource_instance_id
		WHERE b.application_id=$1 AND b.environment=$2 AND b.deployment_target=$3 AND b.retired_at IS NULL`, snapshot.ApplicationDefinition.ID, snapshot.EnvironmentConfiguration.Environment, snapshot.RenderContext.TargetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]domain.ResourceInstance{}
	for rows.Next() {
		var instance domain.ResourceInstance
		if err := rows.Scan(&instance.ID, &instance.ResourceDefinitionID, &instance.OwnerApplicationID, &instance.Environment, &instance.ResourceRequirementID, &instance.DeploymentTarget, &instance.ProviderStateReference, &instance.InfrastructureReference, &instance.DefinitionFingerprint, &instance.ParametersFingerprint, &instance.Version, &instance.RecoveryVerified, &instance.Status); err != nil {
			return nil, err
		}
		if _, exists := result[instance.ResourceRequirementID]; exists {
			return nil, fmt.Errorf("multiple active instances for requirement %s", instance.ResourceRequirementID)
		}
		result[instance.ResourceRequirementID] = instance
	}
	return result, rows.Err()
}

func decodeJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	return decoder.Decode(target)
}
