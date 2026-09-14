CREATE TYPE dependency_target_type AS ENUM ('WORKLOAD', 'RESOURCE');
CREATE TYPE resource_status AS ENUM ('PLANNED', 'PROVISIONING', 'READY', 'FAILED', 'RETIRED');
CREATE TYPE deployment_status AS ENUM ('AWAITING_CONFIRMATION', 'QUEUED', 'RUNNING', 'SUBMITTED', 'FAILED');
CREATE TYPE delivery_status AS ENUM ('NOT_PUBLISHED', 'ACCEPTED', 'SYNCING', 'SYNCED', 'OUT_OF_SYNC', 'DEGRADED', 'FAILED', 'UNKNOWN');
CREATE TYPE step_name AS ENUM ('INFRASTRUCTURE_READY', 'CONFIGURATION_RESOLVED', 'MANIFEST_GENERATED');
CREATE TYPE step_status AS ENUM ('PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'SKIPPED');
CREATE TYPE job_status AS ENUM ('QUEUED', 'CLAIMED', 'SUCCEEDED', 'FAILED');
CREATE TYPE job_phase AS ENUM ('INFRASTRUCTURE', 'CONFIGURATION', 'MANIFEST', 'PUBLISH', 'COMPLETE');
CREATE TYPE guard_status AS ENUM ('IDLE', 'EXECUTING', 'RECOVERY_REQUIRED');

CREATE TABLE application_definition (
    application_id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    retired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workload (
    workload_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition(application_id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    image_repository TEXT NOT NULL,
    port INTEGER CHECK (port BETWEEN 1 AND 65535),
    exposed_outputs JSONB NOT NULL DEFAULT '[]'::jsonb,
    retired_at TIMESTAMPTZ,
    UNIQUE (application_id, name)
);

CREATE TABLE resource_requirement (
    resource_requirement_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition(application_id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    retired_at TIMESTAMPTZ,
    UNIQUE (application_id, name)
);

CREATE TABLE dependency (
    dependency_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition(application_id) ON DELETE RESTRICT,
    source_workload_id UUID NOT NULL REFERENCES workload(workload_id) ON DELETE RESTRICT,
    target_type dependency_target_type NOT NULL,
    target_workload_id UUID REFERENCES workload(workload_id) ON DELETE RESTRICT,
    target_resource_requirement_id UUID REFERENCES resource_requirement(resource_requirement_id) ON DELETE RESTRICT,
    CHECK (
        (target_type = 'WORKLOAD' AND target_workload_id IS NOT NULL AND target_resource_requirement_id IS NULL)
        OR (target_type = 'RESOURCE' AND target_workload_id IS NULL AND target_resource_requirement_id IS NOT NULL)
    )
);

CREATE TABLE environment_configuration (
    environment_configuration_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition(application_id) ON DELETE RESTRICT,
    environment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (application_id, environment)
);

CREATE TABLE configuration_value (
    configuration_value_id UUID PRIMARY KEY,
    environment_configuration_id UUID NOT NULL REFERENCES environment_configuration(environment_configuration_id) ON DELETE RESTRICT,
    workload_id UUID NOT NULL REFERENCES workload(workload_id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    value_source TEXT NOT NULL CHECK (value_source IN ('DIRECT', 'RESOURCE_OUTPUT', 'WORKLOAD_OUTPUT')),
    direct_value TEXT,
    resource_requirement_id UUID REFERENCES resource_requirement(resource_requirement_id) ON DELETE RESTRICT,
    resource_output_name TEXT,
    referenced_workload_id UUID REFERENCES workload(workload_id) ON DELETE RESTRICT,
    workload_output_name TEXT,
    UNIQUE (environment_configuration_id, workload_id, name),
    CHECK (
        (value_source = 'DIRECT' AND direct_value IS NOT NULL AND resource_requirement_id IS NULL AND resource_output_name IS NULL AND referenced_workload_id IS NULL AND workload_output_name IS NULL)
        OR (value_source = 'RESOURCE_OUTPUT' AND direct_value IS NULL AND resource_requirement_id IS NOT NULL AND resource_output_name IS NOT NULL AND referenced_workload_id IS NULL AND workload_output_name IS NULL)
        OR (value_source = 'WORKLOAD_OUTPUT' AND direct_value IS NULL AND resource_requirement_id IS NULL AND resource_output_name IS NULL AND referenced_workload_id IS NOT NULL AND workload_output_name IS NOT NULL)
    )
);

CREATE TABLE secret_reference (
    secret_id UUID PRIMARY KEY,
    environment_configuration_id UUID NOT NULL REFERENCES environment_configuration(environment_configuration_id) ON DELETE RESTRICT,
    workload_id UUID NOT NULL REFERENCES workload(workload_id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    deployment_target TEXT NOT NULL,
    namespace TEXT NOT NULL,
    secret_name TEXT NOT NULL,
    secret_uid TEXT NOT NULL,
    secret_key TEXT NOT NULL,
    UNIQUE (environment_configuration_id, workload_id, name)
);

CREATE TABLE resource_definition (
    resource_definition_id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    resource_type TEXT NOT NULL,
    provisioner_reference TEXT NOT NULL,
    supported_contexts JSONB NOT NULL,
    default_parameters JSONB NOT NULL,
    allowed_overrides JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (allowed_overrides = '{}'::jsonb),
    exposed_outputs JSONB NOT NULL,
    sensitive_outputs JSONB NOT NULL DEFAULT '[]'::jsonb,
    retired_at TIMESTAMPTZ
);

CREATE TABLE resource_instance (
    resource_instance_id UUID PRIMARY KEY,
    resource_definition_id UUID NOT NULL REFERENCES resource_definition(resource_definition_id) ON DELETE RESTRICT,
    owner_application_id UUID NOT NULL REFERENCES application_definition(application_id) ON DELETE RESTRICT,
    environment TEXT NOT NULL,
    resource_requirement_id UUID NOT NULL REFERENCES resource_requirement(resource_requirement_id) ON DELETE RESTRICT,
    deployment_target TEXT NOT NULL,
    provider_state_reference TEXT NOT NULL UNIQUE,
    infrastructure_reference TEXT UNIQUE,
    definition_fingerprint CHAR(64) NOT NULL,
    parameters_fingerprint CHAR(64) NOT NULL,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    recovery_verified BOOLEAN NOT NULL DEFAULT false,
    status resource_status NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (status <> 'READY' OR infrastructure_reference IS NOT NULL)
);

CREATE TABLE resource_instance_binding (
    resource_instance_binding_id UUID PRIMARY KEY,
    resource_instance_id UUID NOT NULL REFERENCES resource_instance(resource_instance_id) ON DELETE RESTRICT,
    application_id UUID NOT NULL REFERENCES application_definition(application_id) ON DELETE RESTRICT,
    environment TEXT NOT NULL,
    resource_requirement_id UUID NOT NULL REFERENCES resource_requirement(resource_requirement_id) ON DELETE RESTRICT,
    deployment_target TEXT NOT NULL,
    binding_role TEXT NOT NULL CHECK (binding_role = 'OWNER'),
    retired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX resource_instance_binding_active_scope
ON resource_instance_binding(application_id, environment, resource_requirement_id, deployment_target)
WHERE retired_at IS NULL;

CREATE TABLE deployment (
    deployment_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition(application_id) ON DELETE RESTRICT,
    environment_configuration_id UUID NOT NULL REFERENCES environment_configuration(environment_configuration_id) ON DELETE RESTRICT,
    environment TEXT NOT NULL,
    deployment_target TEXT NOT NULL,
    plan_fingerprint CHAR(64) NOT NULL,
    plan_fingerprint_algo TEXT NOT NULL CHECK (plan_fingerprint_algo = 'sha256-mvp-v1'),
    status deployment_status NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (deployment_id, application_id, environment, deployment_target)
);

CREATE TABLE deployment_input_snapshot (
    deployment_id UUID PRIMARY KEY REFERENCES deployment(deployment_id) ON DELETE RESTRICT,
    schema_version TEXT NOT NULL CHECK (schema_version = 'mvp-source-v1'),
    application_definition JSONB NOT NULL,
    environment_configuration JSONB NOT NULL,
    render_context JSONB NOT NULL,
    input_fingerprint CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workload_deployment (
    workload_deployment_id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL REFERENCES deployment(deployment_id) ON DELETE RESTRICT,
    workload_id UUID NOT NULL REFERENCES workload(workload_id) ON DELETE RESTRICT,
    image_repository TEXT NOT NULL,
    image_version TEXT NOT NULL,
    image_digest TEXT NOT NULL CHECK (image_digest ~ '^sha256:[0-9a-f]{64}$'),
    UNIQUE (deployment_id, workload_id)
);

CREATE TABLE deployment_context (
    deployment_context_id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL UNIQUE REFERENCES deployment(deployment_id) ON DELETE RESTRICT,
    cloud_provider TEXT NOT NULL CHECK (lower(cloud_provider) = 'aws'),
    region TEXT NOT NULL,
    target_specific_input JSONB NOT NULL
);

CREATE TABLE deployment_scope_guard (
    application_id UUID NOT NULL REFERENCES application_definition(application_id) ON DELETE RESTRICT,
    environment TEXT NOT NULL,
    deployment_target TEXT NOT NULL,
    deployment_id UUID REFERENCES deployment(deployment_id) ON DELETE RESTRICT,
    status guard_status NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (application_id, environment, deployment_target),
    CHECK ((status = 'IDLE' AND deployment_id IS NULL) OR (status <> 'IDLE' AND deployment_id IS NOT NULL))
);

CREATE TABLE deployment_record (
    deployment_record_id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL UNIQUE REFERENCES deployment(deployment_id) ON DELETE RESTRICT,
    environment TEXT NOT NULL,
    deployment_target TEXT NOT NULL,
    delivery_reference TEXT,
    artifact_uri TEXT,
    artifact_digest TEXT CHECK (artifact_digest IS NULL OR artifact_digest ~ '^sha256:[0-9a-f]{64}$'),
    expected_application_name TEXT,
    status deployment_status NOT NULL,
    delivery_status delivery_status NOT NULL DEFAULT 'NOT_PUBLISHED',
    error_summary TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE deployment_step (
    deployment_step_id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL REFERENCES deployment(deployment_id) ON DELETE RESTRICT,
    deployment_record_id UUID NOT NULL REFERENCES deployment_record(deployment_record_id) ON DELETE RESTRICT,
    sequence_number INTEGER NOT NULL CHECK (sequence_number BETWEEN 1 AND 3),
    step_name step_name NOT NULL,
    status step_status NOT NULL,
    related_component_reference TEXT,
    error_summary TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    UNIQUE (deployment_record_id, sequence_number),
    UNIQUE (deployment_id, step_name)
);

CREATE TABLE deployment_execution_job (
    job_id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL UNIQUE REFERENCES deployment(deployment_id) ON DELETE RESTRICT,
    idempotency_key TEXT NOT NULL,
    request_fingerprint CHAR(64) NOT NULL,
    override_values JSONB NOT NULL CHECK (override_values = '{}'::jsonb),
    accepted_plan_fingerprint CHAR(64) NOT NULL,
    status job_status NOT NULL,
    phase job_phase,
    worker_run_id UUID,
    failure_code TEXT,
    last_error TEXT,
    started_at TIMESTAMPTZ,
    heartbeat_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (deployment_id, idempotency_key),
    CHECK (status <> 'CLAIMED' OR (worker_run_id IS NOT NULL AND phase IS NOT NULL AND started_at IS NOT NULL)),
    CHECK (status <> 'SUCCEEDED' OR (phase = 'COMPLETE' AND completed_at IS NOT NULL)),
    CHECK (status <> 'FAILED' OR (failure_code IS NOT NULL AND completed_at IS NOT NULL))
);

CREATE TABLE deployment_record_resource_instance (
    deployment_record_resource_instance_id UUID PRIMARY KEY,
    deployment_record_id UUID NOT NULL REFERENCES deployment_record(deployment_record_id) ON DELETE RESTRICT,
    resource_instance_id UUID NOT NULL REFERENCES resource_instance(resource_instance_id) ON DELETE RESTRICT,
    UNIQUE (deployment_record_id, resource_instance_id)
);

CREATE OR REPLACE FUNCTION reject_immutable_deployment_input()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'deployment input is immutable';
END;
$$;

CREATE TRIGGER deployment_snapshot_immutable
BEFORE UPDATE OR DELETE ON deployment_input_snapshot
FOR EACH ROW EXECUTE FUNCTION reject_immutable_deployment_input();

CREATE TRIGGER workload_deployment_immutable
BEFORE UPDATE OR DELETE ON workload_deployment
FOR EACH ROW EXECUTE FUNCTION reject_immutable_deployment_input();

CREATE TRIGGER deployment_context_immutable
BEFORE UPDATE OR DELETE ON deployment_context
FOR EACH ROW EXECUTE FUNCTION reject_immutable_deployment_input();
