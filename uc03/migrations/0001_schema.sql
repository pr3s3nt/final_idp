-- UC-03 schema. Follows docs/architecture/database/schema.md; extensions are marked
-- "DEVIATION" and listed in implementation_plan.md.

CREATE TYPE component_type AS ENUM (
    'WORKLOAD', 'RESOURCE_REQUIREMENT', 'ENVIRONMENT_VARIABLE_DEFINITION', 'SECRET_DEFINITION',
    'PLATFORM_REQUIREMENT' -- DEVIATION: implicit target infrastructure (k8s-cluster, network)
);
CREATE TYPE dependency_target_type AS ENUM ('WORKLOAD', 'RESOURCE');
CREATE TYPE environment_name AS ENUM ('STAGING', 'PRODUCTION');
CREATE TYPE configuration_value_source AS ENUM ('DIRECT', 'RESOURCE_OUTPUT', 'WORKLOAD_OUTPUT');
CREATE TYPE secret_value_source AS ENUM ('SECRET_REF', 'RESOURCE_OUTPUT');
CREATE TYPE management_mode AS ENUM ('MANAGED', 'EXISTING');
CREATE TYPE resource_instance_status AS ENUM ('PLANNED', 'PROVISIONING', 'READY', 'FAILED', 'DESTROYED', 'UNLINKED');
CREATE TYPE workload_instance_status AS ENUM ('DEPLOYING', 'HEALTHY', 'FAILED', 'REMOVED');
CREATE TYPE deployment_status AS ENUM ('AWAITING_CONFIRMATION', 'CONFIRMED', 'DEPLOYING', 'SUCCEEDED', 'FAILED');
CREATE TYPE deployment_kind AS ENUM ('DEPLOY', 'TEARDOWN'); -- DEVIATION: teardown operation
CREATE TYPE inclusion_reason AS ENUM ('SELECTED', 'CASCADED');
CREATE TYPE job_status AS ENUM ('QUEUED', 'RUNNING', 'COMPLETED', 'FAILED'); -- D6 proposed values
CREATE TYPE step_status AS ENUM ('PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'SKIPPED'); -- D4 proposed values
CREATE TYPE step_name AS ENUM (
    'PLAN_VERIFIED', -- DEVIATION: worker re-checks the confirmed plan before side effects
    'INFRASTRUCTURE_READY', 'CONFIGURATION_RESOLVED', 'MANIFEST_GENERATED', 'CD_SYNCED',
    'APPLICATION_READY', 'REMOVED', 'DESTROYED', 'UNLINKED'
); -- D4 proposed values

-- Application Repository --------------------------------------------------

CREATE TABLE application_definition (
    application_id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE application_definition_version (
    application_definition_version_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition ON DELETE RESTRICT,
    version_number INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (application_id, version_number)
);

CREATE TABLE application_component (
    component_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition ON DELETE RESTRICT,
    component_type component_type NOT NULL,
    platform_requirement_type VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (component_id, component_type),
    CHECK ((component_type = 'PLATFORM_REQUIREMENT') = (platform_requirement_type IS NOT NULL))
);
CREATE UNIQUE INDEX application_component_platform_uq
    ON application_component (application_id, platform_requirement_type)
    WHERE component_type = 'PLATFORM_REQUIREMENT';

CREATE TABLE workload (
    application_definition_version_id UUID NOT NULL
        REFERENCES application_definition_version ON DELETE RESTRICT,
    workload_id UUID NOT NULL,
    component_type component_type NOT NULL DEFAULT 'WORKLOAD' CHECK (component_type = 'WORKLOAD'),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    image_repository VARCHAR(1024) NOT NULL,
    port INT CHECK (port BETWEEN 1 AND 65535),
    exposed_outputs JSONB NOT NULL,
    PRIMARY KEY (application_definition_version_id, workload_id),
    UNIQUE (application_definition_version_id, name),
    FOREIGN KEY (workload_id, component_type) REFERENCES application_component (component_id, component_type)
);

CREATE TABLE resource_requirement (
    application_definition_version_id UUID NOT NULL
        REFERENCES application_definition_version ON DELETE RESTRICT,
    resource_requirement_id UUID NOT NULL,
    component_type component_type NOT NULL DEFAULT 'RESOURCE_REQUIREMENT'
        CHECK (component_type = 'RESOURCE_REQUIREMENT'),
    name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    PRIMARY KEY (application_definition_version_id, resource_requirement_id),
    UNIQUE (application_definition_version_id, name),
    FOREIGN KEY (resource_requirement_id, component_type)
        REFERENCES application_component (component_id, component_type)
);

CREATE TABLE environment_variable_definition (
    application_definition_version_id UUID NOT NULL,
    variable_definition_id UUID NOT NULL,
    component_type component_type NOT NULL DEFAULT 'ENVIRONMENT_VARIABLE_DEFINITION'
        CHECK (component_type = 'ENVIRONMENT_VARIABLE_DEFINITION'),
    workload_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    required BOOLEAN NOT NULL,
    PRIMARY KEY (application_definition_version_id, variable_definition_id),
    UNIQUE (application_definition_version_id, workload_id, name),
    FOREIGN KEY (application_definition_version_id, workload_id)
        REFERENCES workload (application_definition_version_id, workload_id) ON DELETE RESTRICT,
    FOREIGN KEY (variable_definition_id, component_type)
        REFERENCES application_component (component_id, component_type)
);

CREATE TABLE secret_definition (
    application_definition_version_id UUID NOT NULL,
    secret_definition_id UUID NOT NULL,
    component_type component_type NOT NULL DEFAULT 'SECRET_DEFINITION'
        CHECK (component_type = 'SECRET_DEFINITION'),
    workload_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    required BOOLEAN NOT NULL,
    PRIMARY KEY (application_definition_version_id, secret_definition_id),
    UNIQUE (application_definition_version_id, workload_id, name),
    FOREIGN KEY (application_definition_version_id, workload_id)
        REFERENCES workload (application_definition_version_id, workload_id) ON DELETE RESTRICT,
    FOREIGN KEY (secret_definition_id, component_type)
        REFERENCES application_component (component_id, component_type)
);

CREATE TABLE dependency (
    dependency_id UUID PRIMARY KEY,
    application_definition_version_id UUID NOT NULL
        REFERENCES application_definition_version ON DELETE RESTRICT,
    source_workload_id UUID NOT NULL,
    target_type dependency_target_type NOT NULL,
    target_workload_id UUID,
    target_resource_requirement_id UUID,
    FOREIGN KEY (application_definition_version_id, source_workload_id)
        REFERENCES workload (application_definition_version_id, workload_id) ON DELETE RESTRICT,
    FOREIGN KEY (application_definition_version_id, target_workload_id)
        REFERENCES workload (application_definition_version_id, workload_id) ON DELETE RESTRICT,
    FOREIGN KEY (application_definition_version_id, target_resource_requirement_id)
        REFERENCES resource_requirement (application_definition_version_id, resource_requirement_id)
        ON DELETE RESTRICT,
    CHECK (
        (target_type = 'WORKLOAD' AND target_workload_id IS NOT NULL AND target_resource_requirement_id IS NULL)
        OR (target_type = 'RESOURCE' AND target_resource_requirement_id IS NOT NULL AND target_workload_id IS NULL)
    )
);
CREATE UNIQUE INDEX dependency_workload_uq ON dependency
    (application_definition_version_id, source_workload_id, target_workload_id) WHERE target_type = 'WORKLOAD';
CREATE UNIQUE INDEX dependency_resource_uq ON dependency
    (application_definition_version_id, source_workload_id, target_resource_requirement_id) WHERE target_type = 'RESOURCE';

-- Specification Repository -------------------------------------------------

CREATE TABLE application_specification (
    specification_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition ON DELETE RESTRICT,
    application_definition_version_id UUID NOT NULL UNIQUE
        REFERENCES application_definition_version ON DELETE RESTRICT,
    format VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    version VARCHAR(100) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Environment Configuration Repository --------------------------------------

CREATE TABLE environment_configuration (
    environment_configuration_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition ON DELETE RESTRICT,
    environment environment_name NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (application_id, environment)
);

CREATE TABLE environment_variable (
    environment_variable_id UUID PRIMARY KEY,
    environment_configuration_id UUID NOT NULL REFERENCES environment_configuration ON DELETE CASCADE,
    workload_id UUID NOT NULL REFERENCES application_component (component_id),
    variable_definition_id UUID NOT NULL REFERENCES application_component (component_id),
    variable_name VARCHAR(255) NOT NULL,
    UNIQUE (environment_configuration_id, variable_definition_id),
    UNIQUE (environment_configuration_id, environment_variable_id)
);

CREATE TABLE configuration_value (
    configuration_value_id UUID PRIMARY KEY,
    environment_configuration_id UUID NOT NULL,
    environment_variable_id UUID NOT NULL UNIQUE,
    value_source configuration_value_source NOT NULL,
    direct_value TEXT,
    resource_requirement_id UUID REFERENCES application_component (component_id),
    resource_output_name VARCHAR(255),
    workload_id UUID REFERENCES application_component (component_id),
    workload_output_name VARCHAR(255),
    FOREIGN KEY (environment_configuration_id, environment_variable_id)
        REFERENCES environment_variable (environment_configuration_id, environment_variable_id) ON DELETE CASCADE,
    CHECK (
        (value_source = 'DIRECT' AND direct_value IS NOT NULL AND resource_requirement_id IS NULL
            AND resource_output_name IS NULL AND workload_id IS NULL AND workload_output_name IS NULL)
        OR (value_source = 'RESOURCE_OUTPUT' AND direct_value IS NULL AND resource_requirement_id IS NOT NULL
            AND resource_output_name IS NOT NULL AND workload_id IS NULL AND workload_output_name IS NULL)
        OR (value_source = 'WORKLOAD_OUTPUT' AND direct_value IS NULL AND resource_requirement_id IS NULL
            AND resource_output_name IS NULL AND workload_id IS NOT NULL AND workload_output_name IS NOT NULL)
    )
);

CREATE TABLE secret (
    secret_id UUID PRIMARY KEY,
    environment_configuration_id UUID NOT NULL REFERENCES environment_configuration ON DELETE CASCADE,
    workload_id UUID NOT NULL REFERENCES application_component (component_id),
    secret_definition_id UUID NOT NULL REFERENCES application_component (component_id),
    secret_name VARCHAR(255) NOT NULL,
    value_source secret_value_source NOT NULL,
    secret_ref VARCHAR(2048),
    resource_requirement_id UUID REFERENCES application_component (component_id),
    resource_output_name VARCHAR(255),
    UNIQUE (environment_configuration_id, secret_definition_id),
    CHECK (
        (value_source = 'SECRET_REF' AND secret_ref IS NOT NULL
            AND resource_requirement_id IS NULL AND resource_output_name IS NULL)
        OR (value_source = 'RESOURCE_OUTPUT' AND secret_ref IS NULL
            AND resource_requirement_id IS NOT NULL AND resource_output_name IS NOT NULL)
    )
);

-- Platform Resource Definition Catalog ----------------------------------------

CREATE TABLE resource_definition (
    resource_definition_id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    resource_type VARCHAR(100) NOT NULL,
    provisioner_reference VARCHAR(2048) NOT NULL,
    supported_contexts JSONB NOT NULL,
    default_parameters JSONB NOT NULL,
    allowed_overrides JSONB NOT NULL,
    exposed_outputs JSONB NOT NULL,
    sensitive_outputs JSONB NOT NULL,
    management_mode management_mode NOT NULL,
    applicability_conditions JSONB,
    existing_resource_reference VARCHAR(2048),
    requires JSONB NOT NULL DEFAULT '[]', -- DEVIATION: resource types this definition depends on
    CHECK (management_mode <> 'EXISTING' OR existing_resource_reference IS NOT NULL)
);

-- Resource Instance Repository -----------------------------------------------

CREATE TABLE resource_instance (
    resource_instance_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition,
    environment environment_name NOT NULL,
    resource_requirement_id UUID NOT NULL REFERENCES application_component (component_id),
    resource_definition_id UUID NOT NULL REFERENCES resource_definition,
    deployment_target VARCHAR(255) NOT NULL,
    infrastructure_reference VARCHAR(2048) NOT NULL,
    provider_state_reference VARCHAR(2048),
    status resource_instance_status NOT NULL,
    output_fingerprint CHAR(64),
    applied_overrides JSONB NOT NULL DEFAULT '{}', -- DEVIATION: override baseline
    applied_input_fingerprint CHAR(64),            -- DEVIATION: UPDATE vs REUSE decision
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX resource_instance_owner_uq ON resource_instance
    (application_id, environment, resource_requirement_id, deployment_target)
    WHERE status NOT IN ('DESTROYED', 'UNLINKED');

-- Deployment Repository ----------------------------------------------------------

CREATE TABLE deployment (
    deployment_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition,
    application_definition_version_id UUID NOT NULL
        REFERENCES application_definition_version ON DELETE RESTRICT,
    environment_configuration_id UUID NOT NULL REFERENCES environment_configuration,
    environment environment_name NOT NULL,
    deployment_target VARCHAR(255) NOT NULL,
    kind deployment_kind NOT NULL DEFAULT 'DEPLOY',
    plan_fingerprint CHAR(64) NOT NULL,
    plan_fingerprint_algo VARCHAR(32) NOT NULL,
    status deployment_status NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX deployment_owner_idx ON deployment (application_id, environment, deployment_target, status);

CREATE TABLE workload_deployment (
    workload_deployment_id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL REFERENCES deployment ON DELETE CASCADE,
    workload_id UUID NOT NULL REFERENCES application_component (component_id),
    image_repository VARCHAR(1024) NOT NULL,
    image_version VARCHAR(255) NOT NULL,
    inclusion_reason inclusion_reason NOT NULL,
    wave_number INT NOT NULL CHECK (wave_number >= 0),
    UNIQUE (deployment_id, workload_id)
);

CREATE TABLE workload_instance (
    workload_instance_id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES application_definition,
    workload_id UUID NOT NULL REFERENCES application_component (component_id),
    environment environment_name NOT NULL,
    deployment_target VARCHAR(255) NOT NULL,
    current_workload_deployment_id UUID NOT NULL REFERENCES workload_deployment,
    status workload_instance_status NOT NULL,
    output_fingerprint CHAR(64),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
-- DEVIATION: partial UNIQUE so a REMOVED workload can be deployed again (R8a).
CREATE UNIQUE INDEX workload_instance_owner_uq ON workload_instance
    (workload_id, environment, deployment_target) WHERE status <> 'REMOVED';

CREATE TABLE deployment_execution_job (
    job_id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL UNIQUE REFERENCES deployment ON DELETE CASCADE,
    selected_overrides JSONB NOT NULL,
    status job_status NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE deployment_context (
    deployment_context_id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL UNIQUE REFERENCES deployment ON DELETE CASCADE,
    cloud_provider VARCHAR(100),
    region VARCHAR(100),
    target_specific_input JSONB NOT NULL
);

CREATE TABLE deployment_record (
    deployment_record_id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL UNIQUE REFERENCES deployment ON DELETE CASCADE,
    environment environment_name NOT NULL,
    deployment_target VARCHAR(255) NOT NULL,
    delivery_reference VARCHAR(2048),
    removed_components JSONB NOT NULL,
    status deployment_status NOT NULL, -- D5: kept, always written together with deployment.status
    error_summary TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (deployment_record_id, deployment_id)
);

CREATE TABLE deployment_step (
    deployment_step_id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL,
    deployment_record_id UUID NOT NULL,
    sequence_number INT NOT NULL,
    wave_number INT NOT NULL CHECK (wave_number >= 0),
    step_name step_name NOT NULL,
    status step_status NOT NULL,
    related_component_reference VARCHAR(2048),
    error_summary TEXT,
    detail JSONB NOT NULL DEFAULT '{}', -- DEVIATION: per-wave delivery reference and similar
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    UNIQUE (deployment_record_id, sequence_number),
    FOREIGN KEY (deployment_record_id, deployment_id)
        REFERENCES deployment_record (deployment_record_id, deployment_id) ON DELETE CASCADE
);

CREATE TABLE deployment_record_resource_instance (
    deployment_record_resource_instance_id UUID PRIMARY KEY,
    deployment_record_id UUID NOT NULL REFERENCES deployment_record ON DELETE CASCADE,
    resource_instance_id UUID NOT NULL REFERENCES resource_instance,
    UNIQUE (deployment_record_id, resource_instance_id)
);
