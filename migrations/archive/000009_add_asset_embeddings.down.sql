ALTER TABLE embedding_jobs DROP CONSTRAINT IF EXISTS chk_embedding_job_target;
ALTER TABLE embedding_jobs DROP COLUMN IF EXISTS asset_id;
ALTER TABLE embedding_jobs ALTER COLUMN knowledge_id SET NOT NULL;
DROP INDEX IF EXISTS idx_assets_embedding;
ALTER TABLE assets DROP COLUMN IF EXISTS embedding;
