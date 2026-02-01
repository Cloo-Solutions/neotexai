package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/cloo-solutions/neotexai/internal/domain"
)

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

func (r *ProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO projects (id, org_id, name, created_at) VALUES ($1, $2, $3, $4)`,
		project.ID, project.OrgID, project.Name, project.CreatedAt,
	)
	return err
}

func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	var p domain.Project
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, name, created_at FROM projects WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.OrgID, &p.Name, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) ListByOrg(ctx context.Context, orgID string) ([]*domain.Project, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, name, created_at FROM projects WHERE org_id = $1 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*domain.Project
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	cmdTag, err := r.pool.Exec(ctx,
		`UPDATE projects SET name = $1 WHERE id = $2`,
		project.Name, project.ID,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrProjectNotFound
	}
	return nil
}

func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	cmdTag, err := r.pool.Exec(ctx,
		`DELETE FROM projects WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrProjectNotFound
	}
	return nil
}
