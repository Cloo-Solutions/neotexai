package service

import "context"

type testTxRepos struct {
	knowledge     KnowledgeRepositoryInterface
	embeddingJobs EmbeddingJobRepositoryInterface
	assets        AssetRepositoryInterface
}

func (t *testTxRepos) Knowledge() KnowledgeRepositoryInterface {
	return t.knowledge
}

func (t *testTxRepos) EmbeddingJobs() EmbeddingJobRepositoryInterface {
	return t.embeddingJobs
}

func (t *testTxRepos) Assets() AssetRepositoryInterface {
	return t.assets
}

type testTxRunner struct {
	repos  TxRepositories
	called bool
}

func (t *testTxRunner) WithTx(ctx context.Context, fn func(repos TxRepositories) error) error {
	t.called = true
	return fn(t.repos)
}
