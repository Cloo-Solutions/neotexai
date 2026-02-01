CREATE TABLE knowledge_assets (
    knowledge_id UUID NOT NULL REFERENCES knowledge(id),
    asset_id UUID NOT NULL REFERENCES assets(id),
    PRIMARY KEY (knowledge_id, asset_id)
);
