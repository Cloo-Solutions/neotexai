package repository

import (
	"context"
	"errors"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/pagination"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type KnowledgeRepository struct {
	db dbtx
}

func NewKnowledgeRepository(pool *pgxpool.Pool) *KnowledgeRepository {
	return &KnowledgeRepository{db: pool}
}

func NewKnowledgeRepositoryWithTx(tx pgx.Tx) *KnowledgeRepository {
	return &KnowledgeRepository{db: tx}
}

func (r *KnowledgeRepository) Create(ctx context.Context, k *domain.Knowledge) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO knowledge (id, org_id, project_id, type, status, title, summary, body_md, scope_path, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		k.ID, k.OrgID, nullableString(k.ProjectID), k.Type, k.Status, k.Title, k.Summary, k.BodyMD, nullableString(k.Scope), k.CreatedAt, k.UpdatedAt,
	)
	return err
}

func (r *KnowledgeRepository) GetByID(ctx context.Context, id string) (*domain.Knowledge, error) {
	var k domain.Knowledge
	var projectID, scope *string
	err := r.db.QueryRow(ctx,
		`SELECT id, org_id, project_id, type, status, title, summary, body_md, scope_path, created_at, updated_at
		 FROM knowledge WHERE id = $1`,
		id,
	).Scan(&k.ID, &k.OrgID, &projectID, &k.Type, &k.Status, &k.Title, &k.Summary, &k.BodyMD, &scope, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrKnowledgeNotFound
		}
		return nil, err
	}
	if projectID != nil {
		k.ProjectID = *projectID
	}
	if scope != nil {
		k.Scope = *scope
	}
	return &k, nil
}

func (r *KnowledgeRepository) ListByOrg(ctx context.Context, orgID string) ([]*domain.Knowledge, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, project_id, type, status, title, summary, body_md, scope_path, created_at, updated_at
		 FROM knowledge WHERE org_id = $1 ORDER BY updated_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeRows(rows)
}

func (r *KnowledgeRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Knowledge, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, project_id, type, status, title, summary, body_md, scope_path, created_at, updated_at
		 FROM knowledge WHERE project_id = $1 ORDER BY updated_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeRows(rows)
}

func (r *KnowledgeRepository) ListByOrgWithCursor(ctx context.Context, orgID string, cursor *pagination.Cursor, limit int) (*service.KnowledgePageResult, error) {
	if limit <= 0 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	if cursor != nil {
			rows, err = r.db.Query(ctx,
			`SELECT id, org_id, project_id, type, status, title, summary, body_md, scope_path, created_at, updated_at
			 FROM knowledge 
			 WHERE org_id = $1 AND (updated_at, id) < ($2, $3)
			 ORDER BY updated_at DESC, id DESC
			 LIMIT $4`,
			orgID, cursor.Timestamp, cursor.LastID, limit+1,
		)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT id, org_id, project_id, type, status, title, summary, body_md, scope_path, created_at, updated_at
			 FROM knowledge 
			 WHERE org_id = $1
			 ORDER BY updated_at DESC, id DESC
			 LIMIT $2`,
			orgID, limit+1,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items, err := scanKnowledgeRows(rows)
	if err != nil {
		return nil, err
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	var nextCursor string
	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		nextCursor = pagination.EncodeCursor(lastItem.ID, lastItem.UpdatedAt)
	}

	return &service.KnowledgePageResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (r *KnowledgeRepository) ListByProjectWithCursor(ctx context.Context, projectID string, cursor *pagination.Cursor, limit int) (*service.KnowledgePageResult, error) {
	if limit <= 0 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	if cursor != nil {
			rows, err = r.db.Query(ctx,
			`SELECT id, org_id, project_id, type, status, title, summary, body_md, scope_path, created_at, updated_at
			 FROM knowledge 
			 WHERE project_id = $1 AND (updated_at, id) < ($2, $3)
			 ORDER BY updated_at DESC, id DESC
			 LIMIT $4`,
			projectID, cursor.Timestamp, cursor.LastID, limit+1,
		)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT id, org_id, project_id, type, status, title, summary, body_md, scope_path, created_at, updated_at
			 FROM knowledge 
			 WHERE project_id = $1
			 ORDER BY updated_at DESC, id DESC
			 LIMIT $2`,
			projectID, limit+1,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items, err := scanKnowledgeRows(rows)
	if err != nil {
		return nil, err
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	var nextCursor string
	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		nextCursor = pagination.EncodeCursor(lastItem.ID, lastItem.UpdatedAt)
	}

	return &service.KnowledgePageResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (r *KnowledgeRepository) Update(ctx context.Context, k *domain.Knowledge) error {
	k.UpdatedAt = time.Now().UTC()
	cmdTag, err := r.db.Exec(ctx,
		`UPDATE knowledge SET type = $1, status = $2, title = $3, summary = $4, body_md = $5, scope_path = $6, updated_at = $7
		 WHERE id = $8`,
		k.Type, k.Status, k.Title, k.Summary, k.BodyMD, nullableString(k.Scope), k.UpdatedAt, k.ID,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrKnowledgeNotFound
	}
	return nil
}

func (r *KnowledgeRepository) Delete(ctx context.Context, id string) error {
	cmdTag, err := r.db.Exec(ctx,
		`DELETE FROM knowledge WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrKnowledgeNotFound
	}
	return nil
}

func (r *KnowledgeRepository) UpdateEmbedding(ctx context.Context, id string, embedding []float32) error {
	cmdTag, err := r.db.Exec(ctx,
		`UPDATE knowledge SET embedding = $1, updated_at = $2 WHERE id = $3`,
		pgvector.NewVector(embedding), time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrKnowledgeNotFound
	}
	return nil
}

func (r *KnowledgeRepository) CreateVersion(ctx context.Context, v *domain.KnowledgeVersion) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO knowledge_versions (id, knowledge_id, version_number, title, summary, body_md, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		v.ID, v.KnowledgeID, v.VersionNumber, v.Title, v.Summary, v.BodyMD, v.CreatedAt,
	)
	return err
}

func (r *KnowledgeRepository) GetVersions(ctx context.Context, knowledgeID string) ([]*domain.KnowledgeVersion, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, knowledge_id, version_number, title, summary, body_md, created_at
		 FROM knowledge_versions WHERE knowledge_id = $1 ORDER BY version_number DESC`,
		knowledgeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*domain.KnowledgeVersion
	for rows.Next() {
		var v domain.KnowledgeVersion
		if err := rows.Scan(&v.ID, &v.KnowledgeID, &v.VersionNumber, &v.Title, &v.Summary, &v.BodyMD, &v.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, &v)
	}
	return versions, rows.Err()
}

func (r *KnowledgeRepository) GetLatestVersion(ctx context.Context, knowledgeID string) (*domain.KnowledgeVersion, error) {
	var v domain.KnowledgeVersion
	err := r.db.QueryRow(ctx,
		`SELECT id, knowledge_id, version_number, title, summary, body_md, created_at
		 FROM knowledge_versions WHERE knowledge_id = $1 ORDER BY version_number DESC LIMIT 1`,
		knowledgeID,
	).Scan(&v.ID, &v.KnowledgeID, &v.VersionNumber, &v.Title, &v.Summary, &v.BodyMD, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrKnowledgeNotFound
		}
		return nil, err
	}
	return &v, nil
}

func scanKnowledgeRows(rows pgx.Rows) ([]*domain.Knowledge, error) {
	var results []*domain.Knowledge
	for rows.Next() {
		var k domain.Knowledge
		var projectID, scope *string
		if err := rows.Scan(&k.ID, &k.OrgID, &projectID, &k.Type, &k.Status, &k.Title, &k.Summary, &k.BodyMD, &scope, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		if projectID != nil {
			k.ProjectID = *projectID
		}
		if scope != nil {
			k.Scope = *scope
		}
		results = append(results, &k)
	}
	return results, rows.Err()
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
