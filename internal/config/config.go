package config

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port  string `envconfig:"PORT" default:"8080"`
	Debug bool   `envconfig:"DEBUG" default:"false"`

	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

	S3Endpoint  string `envconfig:"S3_ENDPOINT"`
	S3AccessKey string `envconfig:"S3_ACCESS_KEY_ID"`
	S3SecretKey string `envconfig:"S3_SECRET_ACCESS_KEY"`
	S3Bucket    string `envconfig:"S3_BUCKET" default:"neotex-assets"`
	S3Region    string `envconfig:"S3_REGION" default:"us-east-1"`

	OpenAIAPIKey string `envconfig:"OPENAI_API_KEY"`

	// Bootstrap: create initial organization and API key on startup
	InitOrgName string `envconfig:"INIT_ORG_NAME"`
	InitAPIKey  string `envconfig:"INIT_API_KEY"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process("NEOTEX", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process config: %w", err)
	}

	return &cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return cfg
}

func (c *Config) HasS3() bool {
	return c.S3Endpoint != "" && c.S3AccessKey != "" && c.S3SecretKey != ""
}

func (c *Config) HasOpenAI() bool {
	return c.OpenAIAPIKey != ""
}
