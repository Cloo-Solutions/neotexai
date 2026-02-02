package service

import "context"

// TxRepositories provides transaction-bound repositories.
type TxRepositories interface {
	Knowledge() KnowledgeRepositoryInterface
	EmbeddingJobs() EmbeddingJobRepositoryInterface
	Assets() AssetRepositoryInterface
}

// TxRunner executes a function within a transaction.
type TxRunner interface {
	WithTx(ctx context.Context, fn func(repos TxRepositories) error) error
}
