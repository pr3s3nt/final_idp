-- Design issue 13: every application keeps its desired state in its own
-- delivery repository. The IDP creates the repository on the application's
-- first deployment and generates a key pair for that application only. This
-- table holds where the repository is and secret references to the key pair;
-- the keys themselves and the Git hosting credential stay in the Secret Store.

CREATE TABLE delivery_repository (
    delivery_repository_id UUID PRIMARY KEY,
    application_id UUID NOT NULL UNIQUE REFERENCES application_definition ON DELETE RESTRICT,
    repository_url VARCHAR(2048) NOT NULL,
    branch VARCHAR(255) NOT NULL,
    write_key_reference VARCHAR(2048) NOT NULL,
    read_key_reference VARCHAR(2048) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
