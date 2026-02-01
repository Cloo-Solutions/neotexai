package repository

import (
	"context"
	"errors"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyPageResult struct {
	Items      []*domain.APIKey
	NextCursor string
	HasMore    bool
}

type APIKeyRepository struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *APIKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO api_keys (id, org_id, name, key_hash, created_at, revoked_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		key.ID, key.OrgID, key.Name, key.KeyHash, key.CreatedAt, key.RevokedAt,
	)
	return err
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id string) (*domain.APIKey, error) {
	var key domain.APIKey
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, name, key_hash, created_at, revoked_at 
		 FROM api_keys WHERE id = $1`,
		id,
	).Scan(&key.ID, &key.OrgID, &key.Name, &key.KeyHash, &key.CreatedAt, &key.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	var key domain.APIKey
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, name, key_hash, created_at, revoked_at 
		 FROM api_keys WHERE key_hash = $1`,
		hash,
	).Scan(&key.ID, &key.OrgID, &key.Name, &key.KeyHash, &key.CreatedAt, &key.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (r *APIKeyRepository) GetByOrgID(ctx context.Context, orgID string) ([]*domain.APIKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, name, key_hash, created_at, revoked_at 
		 FROM api_keys WHERE org_id = $1 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		var key domain.APIKey
		if err := rows.Scan(&key.ID, &key.OrgID, &key.Name, &key.KeyHash, &key.CreatedAt, &key.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}
	return keys, rows.Err()
}

func (r *APIKeyRepository) ListByOrgWithCursor(ctx context.Context, orgID string, cursor *pagination.Cursor, limit int) (*APIKeyPageResult, error) {
	if limit <= 0 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	if cursor != nil {
		rows, err = r.pool.Query(ctx,
			`SELECT id, org_id, name, key_hash, created_at, revoked_at 
			 FROM api_keys 
			 WHERE org_id = $1 AND (created_at, id) < ($2, $3)
			 ORDER BY created_at DESC, id DESC
			 LIMIT $4`,
			orgID, cursor.Timestamp, cursor.LastID, limit+1,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, org_id, name, key_hash, created_at, revoked_at 
			 FROM api_keys 
			 WHERE org_id = $1
			 ORDER BY created_at DESC, id DESC
			 LIMIT $2`,
			orgID, limit+1,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		var key domain.APIKey
		if err := rows.Scan(&key.ID, &key.OrgID, &key.Name, &key.KeyHash, &key.CreatedAt, &key.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(keys) > limit
	if hasMore {
		keys = keys[:limit]
	}

	var nextCursor string
	if hasMore && len(keys) > 0 {
		lastKey := keys[len(keys)-1]
		nextCursor = pagination.EncodeCursor(lastKey.ID, lastKey.CreatedAt)
	}

	return &APIKeyPageResult{
		Items:      keys,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id string) error {
	now := time.Now().UTC()
	cmdTag, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at = $1 WHERE id = $2 AND revoked_at IS NULL`,
		now, id,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrAPIKeyNotFound
	}
	return nil
}

func (r *APIKeyRepository) Delete(ctx context.Context, id string) error {
	cmdTag, err := r.pool.Exec(ctx,
		`DELETE FROM api_keys WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrAPIKeyNotFound
	}
	return nil
}
