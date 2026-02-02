-- Hybrid search + chunking support

-- Add weighted tsvector columns for lexical search
ALTER TABLE knowledge
    ADD COLUMN search_tsv tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english'::regconfig, coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english'::regconfig, coalesce(summary, '')), 'B') ||
        setweight(to_tsvector('english'::regconfig, coalesce(body_md, '')), 'C') ||
        setweight(to_tsvector('simple'::regconfig, coalesce(scope_path, '')), 'D')
    ) STORED;

ALTER TABLE assets
    ADD COLUMN search_tsv tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple'::regconfig, coalesce(filename, '')), 'A') ||
        setweight(to_tsvector('english'::regconfig, coalesce(description, '')), 'B')
    ) STORED;

-- Chunked knowledge table for improved recall/snippets
CREATE TABLE knowledge_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_id UUID NOT NULL REFERENCES knowledge(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id),
    project_id UUID REFERENCES projects(id),
    type TEXT,
    status TEXT,
    title TEXT,
    summary TEXT,
    scope_path TEXT,
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),
    search_tsv tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english'::regconfig, coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english'::regconfig, coalesce(summary, '')), 'B') ||
        setweight(to_tsvector('english'::regconfig, coalesce(content, '')), 'C') ||
        setweight(to_tsvector('simple'::regconfig, coalesce(scope_path, '')), 'D')
    ) STORED,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Vector indexes
CREATE INDEX idx_knowledge_embedding ON knowledge USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_knowledge_chunks_embedding ON knowledge_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 200);

-- Lexical indexes
CREATE INDEX idx_knowledge_search_tsv ON knowledge USING GIN (search_tsv);
CREATE INDEX idx_assets_search_tsv ON assets USING GIN (search_tsv);
CREATE INDEX idx_knowledge_chunks_search_tsv ON knowledge_chunks USING GIN (search_tsv);

-- Filter indexes
CREATE INDEX idx_knowledge_chunks_org ON knowledge_chunks (org_id);
CREATE INDEX idx_knowledge_chunks_project ON knowledge_chunks (project_id);
CREATE INDEX idx_knowledge_chunks_knowledge_id ON knowledge_chunks (knowledge_id);
CREATE INDEX idx_knowledge_chunks_updated_at ON knowledge_chunks (updated_at DESC);

-- Search logging (optional)
CREATE TABLE search_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    project_id UUID REFERENCES projects(id),
    query TEXT NOT NULL,
    filters JSONB,
    mode TEXT,
    exact BOOLEAN,
    results JSONB,
    result_count INT,
    duration_ms INT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    chosen_id UUID,
    chosen_source TEXT,
    chosen_at TIMESTAMP
);

CREATE INDEX idx_search_logs_org_created_at ON search_logs (org_id, created_at DESC);
