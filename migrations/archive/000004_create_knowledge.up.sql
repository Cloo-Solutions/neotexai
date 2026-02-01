CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE knowledge (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    project_id UUID REFERENCES projects(id),
    type TEXT,
    status TEXT,
    title TEXT,
    summary TEXT,
    body_md TEXT,
    embedding vector(1536),
    scope_path TEXT,
    scope_file TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
