-- Add embedding column to assets for semantic search
ALTER TABLE assets ADD COLUMN embedding vector(1536);

-- Create index for vector similarity search
CREATE INDEX idx_assets_embedding ON assets USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Modify embedding_jobs to support both knowledge and asset embeddings
ALTER TABLE embedding_jobs ADD COLUMN asset_id UUID REFERENCES assets(id);
ALTER TABLE embedding_jobs ALTER COLUMN knowledge_id DROP NOT NULL;

-- Add constraint: must have either knowledge_id or asset_id
ALTER TABLE embedding_jobs ADD CONSTRAINT chk_embedding_job_target 
    CHECK ((knowledge_id IS NOT NULL AND asset_id IS NULL) OR (knowledge_id IS NULL AND asset_id IS NOT NULL));
