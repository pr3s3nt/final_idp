-- Design issue 12: the Resource Definition catalog is versioned. A catalog
-- change creates a new immutable catalog_version with all its definitions; a
-- deployment records the catalog version it resolved with.

CREATE TABLE catalog_version (
    catalog_version_id UUID PRIMARY KEY,
    version_number INT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL
);

-- Definitions imported before versioning become catalog version 1.
INSERT INTO catalog_version (catalog_version_id, version_number, created_at)
SELECT gen_random_uuid(), 1, now() WHERE EXISTS (SELECT 1 FROM resource_definition);

ALTER TABLE resource_definition
    ADD COLUMN catalog_version_id UUID REFERENCES catalog_version ON DELETE RESTRICT;
UPDATE resource_definition
   SET catalog_version_id = (SELECT catalog_version_id FROM catalog_version WHERE version_number = 1);
ALTER TABLE resource_definition ALTER COLUMN catalog_version_id SET NOT NULL;
ALTER TABLE resource_definition DROP CONSTRAINT resource_definition_name_key;
ALTER TABLE resource_definition
    ADD CONSTRAINT resource_definition_version_name_uq UNIQUE (catalog_version_id, name);

ALTER TABLE deployment
    ADD COLUMN catalog_version_id UUID REFERENCES catalog_version ON DELETE RESTRICT;
UPDATE deployment
   SET catalog_version_id = (SELECT catalog_version_id FROM catalog_version WHERE version_number = 1);
ALTER TABLE deployment ALTER COLUMN catalog_version_id SET NOT NULL;
