-- +goose Up
CREATE TABLE admin_users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         text        NOT NULL UNIQUE,
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Only a SHA-256 of the session token is stored, so a database leak does not expose live sessions.
CREATE TABLE admin_sessions (
    token_hash bytea       PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES admin_users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX admin_sessions_expires_idx ON admin_sessions (expires_at);

CREATE INDEX orders_status_idx ON orders (status);

-- +goose Down
DROP INDEX orders_status_idx;
DROP TABLE admin_sessions;
DROP TABLE admin_users;
