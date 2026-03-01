CREATE TABLE users (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workflows (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    retry_policy JSONB NOT NULL DEFAULT '{"max_retries":3,"initial_delay_ms":1000,"max_delay_ms":30000,"multiplier":2.0}',
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workflow_steps (
    id          TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    step_index  INTEGER NOT NULL,
    action      TEXT NOT NULL CHECK (action IN ('http_call', 'delay', 'log', 'transform')),
    config      JSONB NOT NULL DEFAULT '{}',
    name        TEXT NOT NULL DEFAULT '',
    UNIQUE (workflow_id, step_index)
);

CREATE TABLE workflow_runs (
    id           TEXT PRIMARY KEY,
    workflow_id  TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    context      JSONB NOT NULL DEFAULT '{}',
    current_step INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE step_executions (
    id             TEXT PRIMARY KEY,
    run_id         TEXT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    step_index     INTEGER NOT NULL,
    attempt_id     TEXT NOT NULL UNIQUE,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    action         TEXT NOT NULL CHECK (action IN ('http_call', 'delay', 'log', 'transform')),
    status         TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    input          JSONB NOT NULL DEFAULT '{}',
    output         JSONB NOT NULL DEFAULT '{}',
    error_message  TEXT,
    duration_ms    INTEGER,
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ
);

-- Seed user for development (no auth yet)
INSERT INTO users (id, email, password_hash, display_name)
VALUES ('01JDEFAULT000000000000000', 'dev@flowforge.local', 'nologin', 'Dev User');
