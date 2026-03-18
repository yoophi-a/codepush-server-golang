CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    github_id TEXT,
    microsoft_id TEXT,
    azure_ad_id TEXT
);

CREATE TABLE IF NOT EXISTS access_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    friendly_name TEXT NOT NULL,
    description TEXT,
    created_by TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    expires BIGINT NOT NULL,
    is_session BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at BIGINT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_apps_name_created_at ON apps (name, created_at, id);

CREATE TABLE IF NOT EXISTS app_collaborators (
    app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    permission TEXT NOT NULL,
    PRIMARY KEY (app_id, account_id)
);

CREATE TABLE IF NOT EXISTS deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    deployment_key TEXT NOT NULL UNIQUE,
    created_at BIGINT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_deployments_app_name ON deployments (app_id, name);

CREATE TABLE IF NOT EXISTS packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    ordinal INTEGER NOT NULL,
    label TEXT NOT NULL,
    app_version TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_disabled BOOLEAN NOT NULL DEFAULT FALSE,
    is_mandatory BOOLEAN NOT NULL DEFAULT FALSE,
    package_hash TEXT NOT NULL DEFAULT '',
    blob_url TEXT NOT NULL DEFAULT '',
    manifest_blob_url TEXT NOT NULL DEFAULT '',
    rollout INTEGER,
    size BIGINT NOT NULL DEFAULT 0,
    upload_time BIGINT NOT NULL,
    release_method TEXT NOT NULL DEFAULT 'Upload',
    original_label TEXT,
    original_deployment TEXT,
    released_by TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_packages_deployment_ordinal ON packages (deployment_id, ordinal);
CREATE UNIQUE INDEX IF NOT EXISTS idx_packages_deployment_label ON packages (deployment_id, label);
