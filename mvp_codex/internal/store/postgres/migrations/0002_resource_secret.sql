ALTER TABLE secret_reference
    ADD COLUMN source_type TEXT NOT NULL DEFAULT 'KUBERNETES_SECRET',
    ADD COLUMN resource_requirement_id UUID REFERENCES resource_requirement(resource_requirement_id) ON DELETE RESTRICT,
    ADD COLUMN resource_output_name TEXT,
    ADD COLUMN remote_property TEXT,
    ALTER COLUMN secret_uid DROP NOT NULL;

ALTER TABLE secret_reference
    ADD CONSTRAINT secret_reference_source_check CHECK (
        (
            source_type = 'KUBERNETES_SECRET'
            AND secret_uid IS NOT NULL
            AND resource_requirement_id IS NULL
            AND resource_output_name IS NULL
            AND remote_property IS NULL
        )
        OR
        (
            source_type = 'RESOURCE_SECRET'
            AND secret_uid IS NULL
            AND resource_requirement_id IS NOT NULL
            AND resource_output_name IS NOT NULL
            AND remote_property IS NOT NULL
        )
    );
