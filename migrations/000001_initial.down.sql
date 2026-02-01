-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS embedding_jobs;
DROP TABLE IF EXISTS knowledge_assets;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS knowledge_versions;
DROP TABLE IF EXISTS knowledge;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS organizations;

-- Drop extension
DROP EXTENSION IF EXISTS vector;
