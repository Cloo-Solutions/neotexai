package repository

import (
	"context"
	"errors"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrgPageResult struct {
	Items      []*domain.Organization
	NextCursor string
	HasMore    bool
}

type OrgRepository struct {
	pool *pgxpool.Pool
}

func NewOrgRepository(pool *pgxpool.Pool) *OrgRepository {
	return &OrgRepository{pool: pool}
}

func (r *OrgRepository) Create(ctx context.Context, org *domain.Organization) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO organizations (id, name, created_at) VALUES ($1, $2, $3)`,
		org.ID, org.Name, org.CreatedAt,
	)
	return err
}

func (r *OrgRepository) GetByID(ctx context.Context, id string) (*domain.Organization, error) {
	var org domain.Organization
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, created_at FROM organizations WHERE id = $1`,
		id,
	).Scan(&org.ID, &org.Name, &org.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrganizationNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (r *OrgRepository) List(ctx context.Context) ([]*domain.Organization, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, created_at FROM organizations ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*domain.Organization
	for rows.Next() {
		var org domain.Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.CreatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, &org)
	}
	return orgs, rows.Err()
}

func (r *OrgRepository) ListWithCursor(ctx context.Context, cursor *pagination.Cursor, limit int) (*OrgPageResult, error) {
	if limit <= 0 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	if cursor != nil {
		rows, err = r.pool.Query(ctx,
			`SELECT id, name, created_at FROM organizations 
			 WHERE (created_at, id) < ($1, $2)
			 ORDER BY created_at DESC, id DESC
			 LIMIT $3`,
			cursor.Timestamp, cursor.LastID, limit+1,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, name, created_at FROM organizations 
			 ORDER BY created_at DESC, id DESC
			 LIMIT $1`,
			limit+1,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*domain.Organization
	for rows.Next() {
		var org domain.Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.CreatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, &org)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(orgs) > limit
	if hasMore {
		orgs = orgs[:limit]
	}

	var nextCursor string
	if hasMore && len(orgs) > 0 {
		lastOrg := orgs[len(orgs)-1]
		nextCursor = pagination.EncodeCursor(lastOrg.ID, lastOrg.CreatedAt)
	}

	return &OrgPageResult{
		Items:      orgs,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (r *OrgRepository) Update(ctx context.Context, org *domain.Organization) error {
	cmdTag, err := r.pool.Exec(ctx,
		`UPDATE organizations SET name = $1 WHERE id = $2`,
		org.Name, org.ID,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrOrganizationNotFound
	}
	return nil
}

func (r *OrgRepository) Delete(ctx context.Context, id string) error {
	cmdTag, err := r.pool.Exec(ctx,
		`DELETE FROM organizations WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrOrganizationNotFound
	}
	return nil
}

func (r *OrgRepository) GetByName(ctx context.Context, name string) (*domain.Organization, error) {
	var org domain.Organization
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, created_at FROM organizations WHERE name = $1`,
		name,
	).Scan(&org.ID, &org.Name, &org.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrganizationNotFound
		}
		return nil, err
	}
	return &org, nil
}
