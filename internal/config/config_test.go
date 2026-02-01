package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_WithEnvVars(t *testing.T) {
	os.Setenv("NEOTEX_DATABASE_URL", "postgres://test:test@localhost:5432/test")
	os.Setenv("NEOTEX_PORT", "9090")
	os.Setenv("NEOTEX_DEBUG", "true")
	os.Setenv("NEOTEX_S3_ENDPOINT", "http://localhost:9000")
	os.Setenv("NEOTEX_S3_ACCESS_KEY_ID", "key")
	os.Setenv("NEOTEX_S3_SECRET_ACCESS_KEY", "secret")
	os.Setenv("NEOTEX_OPENAI_API_KEY", "sk-test")
	defer func() {
		os.Unsetenv("NEOTEX_DATABASE_URL")
		os.Unsetenv("NEOTEX_PORT")
		os.Unsetenv("NEOTEX_DEBUG")
		os.Unsetenv("NEOTEX_S3_ENDPOINT")
		os.Unsetenv("NEOTEX_S3_ACCESS_KEY_ID")
		os.Unsetenv("NEOTEX_S3_SECRET_ACCESS_KEY")
		os.Unsetenv("NEOTEX_OPENAI_API_KEY")
	}()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "postgres://test:test@localhost:5432/test", cfg.DatabaseURL)
	assert.Equal(t, "9090", cfg.Port)
	assert.True(t, cfg.Debug)
	assert.Equal(t, "http://localhost:9000", cfg.S3Endpoint)
	assert.Equal(t, "key", cfg.S3AccessKey)
	assert.Equal(t, "secret", cfg.S3SecretKey)
	assert.Equal(t, "sk-test", cfg.OpenAIAPIKey)
}

func TestLoad_Defaults(t *testing.T) {
	os.Setenv("NEOTEX_DATABASE_URL", "postgres://test:test@localhost:5432/test")
	defer os.Unsetenv("NEOTEX_DATABASE_URL")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port)
	assert.False(t, cfg.Debug)
	assert.Equal(t, "neotex-assets", cfg.S3Bucket)
	assert.Equal(t, "us-east-1", cfg.S3Region)
}

func TestLoad_RequiredDatabaseURL(t *testing.T) {
	os.Unsetenv("NEOTEX_DATABASE_URL")

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestHasS3(t *testing.T) {
	cfg := &Config{
		S3Endpoint:  "http://localhost:9000",
		S3AccessKey: "key",
		S3SecretKey: "secret",
	}
	assert.True(t, cfg.HasS3())

	cfg.S3Endpoint = ""
	assert.False(t, cfg.HasS3())
}

func TestHasOpenAI(t *testing.T) {
	cfg := &Config{OpenAIAPIKey: "sk-test"}
	assert.True(t, cfg.HasOpenAI())

	cfg.OpenAIAPIKey = ""
	assert.False(t, cfg.HasOpenAI())
}
