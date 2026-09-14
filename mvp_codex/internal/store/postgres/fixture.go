package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/fixture"
)

func (s *Store) SeedFixture(ctx context.Context, data fixture.Data) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	app := data.Snapshot.ApplicationDefinition
	_, err = tx.Exec(ctx, `INSERT INTO application_definition(application_id,name,description) VALUES($1,$2,$3)
		ON CONFLICT(application_id) DO UPDATE SET name=excluded.name, description=excluded.description, updated_at=now()`, app.ID, app.Name, "UC3 fixture")
	if err != nil {
		return fmt.Errorf("seed application: %w", err)
	}
	for _, workload := range app.Workloads {
		outputs, err := json.Marshal(workload.ExposedOutputs)
		if err != nil {
			return err
		}
		var port any
		if workload.Port > 0 {
			port = workload.Port
		}
		_, err = tx.Exec(ctx, `INSERT INTO workload(workload_id,application_id,name,type,image_repository,port,exposed_outputs)
			VALUES($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT(workload_id) DO UPDATE SET name=excluded.name,type=excluded.type,image_repository=excluded.image_repository,port=excluded.port,exposed_outputs=excluded.exposed_outputs,retired_at=NULL`,
			workload.ID, app.ID, workload.Name, workload.Type, workload.ImageRepository, port, outputs)
		if err != nil {
			return fmt.Errorf("seed workload %s: %w", workload.ID, err)
		}
	}
	for _, requirement := range app.ResourceRequirements {
		_, err = tx.Exec(ctx, `INSERT INTO resource_requirement(resource_requirement_id,application_id,name,resource_type) VALUES($1,$2,$3,$4)
			ON CONFLICT(resource_requirement_id) DO UPDATE SET name=excluded.name,resource_type=excluded.resource_type,retired_at=NULL`,
			requirement.ID, app.ID, requirement.Name, requirement.ResourceType)
		if err != nil {
			return fmt.Errorf("seed requirement %s: %w", requirement.ID, err)
		}
	}
	for _, dependency := range app.Dependencies {
		var targetWorkload, targetResource any
		if dependency.TargetWorkloadID != "" {
			targetWorkload = dependency.TargetWorkloadID
		}
		if dependency.TargetResourceRequirementID != "" {
			targetResource = dependency.TargetResourceRequirementID
		}
		_, err = tx.Exec(ctx, `INSERT INTO dependency(dependency_id,application_id,source_workload_id,target_type,target_workload_id,target_resource_requirement_id)
			VALUES($1,$2,$3,$4,$5,$6)
			ON CONFLICT(dependency_id) DO UPDATE SET source_workload_id=excluded.source_workload_id,target_type=excluded.target_type,target_workload_id=excluded.target_workload_id,target_resource_requirement_id=excluded.target_resource_requirement_id`,
			dependency.ID, app.ID, dependency.SourceWorkloadID, dependency.TargetType, targetWorkload, targetResource)
		if err != nil {
			return fmt.Errorf("seed dependency %s: %w", dependency.ID, err)
		}
	}

	config := data.Snapshot.EnvironmentConfiguration
	_, err = tx.Exec(ctx, `INSERT INTO environment_configuration(environment_configuration_id,application_id,environment) VALUES($1,$2,$3)
		ON CONFLICT(environment_configuration_id) DO UPDATE SET environment=excluded.environment,updated_at=now()`, config.ID, app.ID, config.Environment)
	if err != nil {
		return fmt.Errorf("seed configuration: %w", err)
	}
	for _, value := range config.Values {
		args := nullableConfiguration(value)
		_, err = tx.Exec(ctx, `INSERT INTO configuration_value(configuration_value_id,environment_configuration_id,workload_id,name,value_source,direct_value,resource_requirement_id,resource_output_name,referenced_workload_id,workload_output_name)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT(configuration_value_id) DO UPDATE SET workload_id=excluded.workload_id,name=excluded.name,value_source=excluded.value_source,direct_value=excluded.direct_value,resource_requirement_id=excluded.resource_requirement_id,resource_output_name=excluded.resource_output_name,referenced_workload_id=excluded.referenced_workload_id,workload_output_name=excluded.workload_output_name`,
			value.ID, config.ID, value.WorkloadID, value.Name, value.Source, args[0], args[1], args[2], args[3], args[4])
		if err != nil {
			return fmt.Errorf("seed configuration value %s: %w", value.ID, err)
		}
	}
	for _, secret := range config.Secrets {
		source := secret.Source
		if source == "" {
			source = domain.SecretKubernetes
		}
		_, err = tx.Exec(ctx, `INSERT INTO secret_reference(secret_id,environment_configuration_id,workload_id,name,source_type,deployment_target,namespace,secret_name,secret_uid,secret_key,resource_requirement_id,resource_output_name,remote_property)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),$10,NULLIF($11,'')::uuid,NULLIF($12,''),NULLIF($13,''))
			ON CONFLICT(secret_id) DO UPDATE SET workload_id=excluded.workload_id,name=excluded.name,source_type=excluded.source_type,deployment_target=excluded.deployment_target,namespace=excluded.namespace,secret_name=excluded.secret_name,secret_uid=excluded.secret_uid,secret_key=excluded.secret_key,resource_requirement_id=excluded.resource_requirement_id,resource_output_name=excluded.resource_output_name,remote_property=excluded.remote_property`,
			secret.ID, config.ID, secret.WorkloadID, secret.Name, source, secret.Target, secret.Namespace, secret.SecretName, secret.UID, secret.Key, secret.ResourceRequirementID, secret.ResourceOutputName, secret.RemoteProperty)
		if err != nil {
			return fmt.Errorf("seed secret reference %s: %w", secret.ID, err)
		}
	}
	for _, definition := range data.Definitions {
		contexts, err := json.Marshal(definition.SupportedContexts)
		if err != nil {
			return err
		}
		parameters, err := json.Marshal(definition.DefaultParameters)
		if err != nil {
			return err
		}
		overrides, _ := json.Marshal(definition.AllowedOverrides)
		outputs, _ := json.Marshal(definition.ExposedOutputs)
		sensitive, _ := json.Marshal(definition.SensitiveOutputs)
		_, err = tx.Exec(ctx, `INSERT INTO resource_definition(resource_definition_id,name,resource_type,provisioner_reference,supported_contexts,default_parameters,allowed_overrides,exposed_outputs,sensitive_outputs,retired_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULL)
			ON CONFLICT(resource_definition_id) DO UPDATE SET name=excluded.name,resource_type=excluded.resource_type,provisioner_reference=excluded.provisioner_reference,supported_contexts=excluded.supported_contexts,default_parameters=excluded.default_parameters,allowed_overrides=excluded.allowed_overrides,exposed_outputs=excluded.exposed_outputs,sensitive_outputs=excluded.sensitive_outputs,retired_at=NULL`,
			definition.ID, definition.Name, definition.ResourceType, definition.ProvisionerReference, contexts, parameters, overrides, outputs, sensitive)
		if err != nil {
			return fmt.Errorf("seed resource definition %s: %w", definition.ID, err)
		}
	}
	return tx.Commit(ctx)
}

func nullableConfiguration(value domain.ConfigurationValue) []any {
	result := make([]any, 5)
	if value.Source == domain.ValueDirect {
		result[0] = value.DirectValue
	}
	if value.ResourceRequirementID != "" {
		result[1] = value.ResourceRequirementID
	}
	if value.ResourceOutputName != "" {
		result[2] = value.ResourceOutputName
	}
	if value.ReferencedWorkloadID != "" {
		result[3] = value.ReferencedWorkloadID
	}
	if value.WorkloadOutputName != "" {
		result[4] = value.WorkloadOutputName
	}
	return result
}
