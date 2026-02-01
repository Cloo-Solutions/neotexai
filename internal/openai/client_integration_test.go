//go:build integration

package openai

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GenerateEmbedding_RealAPI(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	client := NewClient(apiKey)
	ctx := context.Background()
	text := "This is a test document for generating embeddings."

	embedding, err := client.GenerateEmbedding(ctx, text)

	require.NoError(t, err)
	assert.Len(t, embedding, DefaultEmbeddingDimensions)
}
