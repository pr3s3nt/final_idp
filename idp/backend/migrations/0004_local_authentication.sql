CREATE TYPE user_account_status AS ENUM ('ACTIVE', 'DISABLED');
CREATE TYPE login_attempt_scope AS ENUM ('ACCOUNT', 'SOURCE');

CREATE TABLE user_account (
    user_id UUID PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    status user_account_status NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK (username = lower(btrim(username))),
    CHECK (char_length(username) BETWEEN 3 AND 64),
    CHECK (username ~ '^[a-z0-9][a-z0-9._-]{2,63}$'),
    CHECK (display_name = btrim(display_name)),
    CHECK (char_length(display_name) BETWEEN 1 AND 255),
    CHECK (display_name !~ '[[:cntrl:]]')
);

CREATE TABLE local_credential (
    user_id UUID PRIMARY KEY REFERENCES user_account (user_id) ON DELETE RESTRICT,
    password_hash TEXT NOT NULL,
    password_changed_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE auth_session (
    session_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES user_account (user_id) ON DELETE RESTRICT,
    token_hash BYTEA NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    csrf_token_hash BYTEA NOT NULL CHECK (octet_length(csrf_token_hash) = 32),
    created_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    CHECK (expires_at = created_at + INTERVAL '8 hours'),
    CHECK (last_seen_at >= created_at),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);
CREATE INDEX auth_session_user_revoked_idx ON auth_session (user_id, revoked_at);
CREATE INDEX auth_session_expires_idx ON auth_session (expires_at);

CREATE TABLE login_attempt (
    login_attempt_id UUID PRIMARY KEY,
    scope login_attempt_scope NOT NULL,
    key_hash BYTEA NOT NULL CHECK (octet_length(key_hash) = 32),
    window_started_at TIMESTAMPTZ NOT NULL,
    failure_count INT NOT NULL CHECK (failure_count >= 0),
    blocked_until TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE (scope, key_hash)
);
CREATE INDEX login_attempt_expires_idx ON login_attempt (expires_at);
