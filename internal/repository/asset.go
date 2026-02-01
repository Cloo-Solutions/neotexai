package repository

import (
	"context"
	"errors"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type AssetRepository struct {
	pool *pgxpool.Pool
}

func NewAssetRepository(pool *pgxpool.Pool) *AssetRepository {
	return &AssetRepository{pool: pool}
}

func (r *AssetRepository) Create(ctx context.Context, a *domain.Asset) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO assets (id, org_id, project_id, filename, mime_type, sha256, storage_key, keywords, description, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		a.ID, a.OrgID, nullableString(a.ProjectID), a.Filename, a.MimeType, a.SHA256, a.StorageKey, a.Keywords, a.Description, a.CreatedAt,
	)
	return err
}

func (r *AssetRepository) GetByID(ctx context.Context, id string) (*domain.Asset, error) {
	var a domain.Asset
	var projectID *string
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, project_id, filename, mime_type, sha256, storage_key, keywords, description, created_at
		 FROM assets WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.OrgID, &projectID, &a.Filename, &a.MimeType, &a.SHA256, &a.StorageKey, &a.Keywords, &a.Description, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAssetNotFound
		}
		return nil, err
	}
	if projectID != nil {
		a.ProjectID = *projectID
	}
	return &a, nil
}

func (r *AssetRepository) ListByKnowledge(ctx context.Context, knowledgeID string) ([]*domain.Asset, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT a.id, a.org_id, a.project_id, a.filename, a.mime_type, a.sha256, a.storage_key, a.keywords, a.description, a.created_at
		 FROM assets a
		 INNER JOIN knowledge_assets ka ON a.id = ka.asset_id
		 WHERE ka.knowledge_id = $1
		 ORDER BY a.created_at DESC`,
		knowledgeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*domain.Asset
	for rows.Next() {
		var a domain.Asset
		var projectID *string
		if err := rows.Scan(&a.ID, &a.OrgID, &projectID, &a.Filename, &a.MimeType, &a.SHA256, &a.StorageKey, &a.Keywords, &a.Description, &a.CreatedAt); err != nil {
			return nil, err
		}
		if projectID != nil {
			a.ProjectID = *projectID
		}
		assets = append(assets, &a)
	}
	return assets, rows.Err()
}

func (r *AssetRepository) Delete(ctx context.Context, id string) error {
	cmdTag, err := r.pool.Exec(ctx,
		`DELETE FROM assets WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrAssetNotFound
	}
	return nil
}

func (r *AssetRepository) LinkToKnowledge(ctx context.Context, knowledgeID, assetID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO knowledge_assets (knowledge_id, asset_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		knowledgeID, assetID,
	)
	return err
}

func (r *AssetRepository) UnlinkFromKnowledge(ctx context.Context, knowledgeID, assetID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM knowledge_assets WHERE knowledge_id = $1 AND asset_id = $2`,
		knowledgeID, assetID,
	)
	return err
}

func (r *AssetRepository) UpdateEmbedding(ctx context.Context, id string, embedding []float32) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE assets SET embedding = $1 WHERE id = $2`,
		pgvector.NewVector(embedding), id,
	)
	return err
}
