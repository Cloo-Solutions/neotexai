package repository

import (
	"context"

	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxRunner provides transactional repositories using a pgx pool.
type TxRunner struct {
	pool *pgxpool.Pool
}

func NewTxRunner(pool *pgxpool.Pool) *TxRunner {
	return &TxRunner{pool: pool}
}

func (r *TxRunner) WithTx(ctx context.Context, fn func(repos service.TxRepositories) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	repos := &txRepos{tx: tx}
	if err := fn(repos); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

type txRepos struct {
	tx pgx.Tx
}

func (r *txRepos) Knowledge() service.KnowledgeRepositoryInterface {
	return NewKnowledgeRepositoryWithTx(r.tx)
}

func (r *txRepos) EmbeddingJobs() service.EmbeddingJobRepositoryInterface {
	return NewEmbeddingJobRepositoryWithTx(r.tx)
}

func (r *txRepos) Assets() service.AssetRepositoryInterface {
	return NewAssetRepositoryWithTx(r.tx)
}
