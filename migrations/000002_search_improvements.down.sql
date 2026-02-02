-- Roll back hybrid search + chunking support

DROP INDEX IF EXISTS idx_knowledge_chunks_updated_at;
DROP INDEX IF EXISTS idx_knowledge_chunks_knowledge_id;
DROP INDEX IF EXISTS idx_knowledge_chunks_project;
DROP INDEX IF EXISTS idx_knowledge_chunks_org;

DROP INDEX IF EXISTS idx_search_logs_org_created_at;
DROP TABLE IF EXISTS search_logs;

DROP INDEX IF EXISTS idx_knowledge_chunks_search_tsv;
DROP INDEX IF EXISTS idx_assets_search_tsv;
DROP INDEX IF EXISTS idx_knowledge_search_tsv;

DROP INDEX IF EXISTS idx_knowledge_chunks_embedding;
DROP INDEX IF EXISTS idx_knowledge_embedding;

DROP TABLE IF EXISTS knowledge_chunks;

ALTER TABLE assets DROP COLUMN IF EXISTS search_tsv;
ALTER TABLE knowledge DROP COLUMN IF EXISTS search_tsv;
