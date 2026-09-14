ALTER TABLE deployment_context
    DROP CONSTRAINT deployment_context_cloud_provider_check;

ALTER TABLE deployment_context
    ADD CONSTRAINT deployment_context_cloud_provider_check
    CHECK (lower(cloud_provider) IN ('aws', 'kind', 'local'));
