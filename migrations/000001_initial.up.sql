-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create projects table
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create api_keys table
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    name TEXT,
    key_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP
);

-- Create knowledge table
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

-- Create knowledge_versions table
CREATE TABLE knowledge_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_id UUID NOT NULL REFERENCES knowledge(id),
    version_number INT NOT NULL,
    title TEXT,
    summary TEXT,
    body_md TEXT,
    embedding vector(1536),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create assets table
CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    project_id UUID REFERENCES projects(id),
    filename TEXT NOT NULL,
    mime_type TEXT,
    sha256 TEXT,
    storage_key TEXT,
    keywords TEXT[],
    description TEXT,
    embedding vector(1536),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create knowledge_assets junction table
CREATE TABLE knowledge_assets (
    knowledge_id UUID NOT NULL REFERENCES knowledge(id),
    asset_id UUID NOT NULL REFERENCES assets(id),
    PRIMARY KEY (knowledge_id, asset_id)
);

-- Create embedding_jobs table
CREATE TABLE embedding_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_id UUID REFERENCES knowledge(id),
    asset_id UUID REFERENCES assets(id),
    status TEXT,
    retries INT,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP,
    CONSTRAINT chk_embedding_job_target 
        CHECK ((knowledge_id IS NOT NULL AND asset_id IS NULL) OR (knowledge_id IS NULL AND asset_id IS NOT NULL))
);

-- Create index for asset embeddings vector similarity search
CREATE INDEX idx_assets_embedding ON assets USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
