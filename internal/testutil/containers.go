package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer represents a PostgreSQL container for testing
type PostgresContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	User      string
	Password  string
	Database  string
}

// NewPostgresContainer creates and starts a PostgreSQL container with pgvector
func NewPostgresContainer(ctx context.Context, t *testing.T) *PostgresContainer {
	req := testcontainers.ContainerRequest{
		Image:        "pgvector/pgvector:0.8.1-pg18",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "neotex",
			"POSTGRES_PASSWORD": "neotex",
			"POSTGRES_DB":       "neotex",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			wait.ForListeningPort("5432/tcp"),
		).WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to create postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	return &PostgresContainer{
		Container: container,
		Host:      host,
		Port:      port.Port(),
		User:      "neotex",
		Password:  "neotex",
		Database:  "neotex",
	}
}

// ConnectionString returns the PostgreSQL connection string
func (pc *PostgresContainer) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		pc.User, pc.Password, pc.Host, pc.Port, pc.Database)
}

// Terminate stops and removes the container
func (pc *PostgresContainer) Terminate(ctx context.Context) error {
	return testcontainers.TerminateContainer(pc.Container)
}

// RustFSContainer represents a RustFS container for testing
type RustFSContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
}

// NewRustFSContainer creates and starts a RustFS container
func NewRustFSContainer(ctx context.Context, t *testing.T) *RustFSContainer {
	req := testcontainers.ContainerRequest{
		Image:        "rustfs/rustfs:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"RUSTFS_ACCESS_KEY": "rustfsadmin",
			"RUSTFS_SECRET_KEY": "rustfsadmin",
		},
		WaitingFor: wait.ForListeningPort("9000/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to create rustfs container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "9000")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	return &RustFSContainer{
		Container: container,
		Host:      host,
		Port:      port.Port(),
	}
}

// Endpoint returns the RustFS endpoint URL
func (rc *RustFSContainer) Endpoint() string {
	return fmt.Sprintf("http://%s:%s", rc.Host, rc.Port)
}

// Terminate stops and removes the container
func (rc *RustFSContainer) Terminate(ctx context.Context) error {
	return testcontainers.TerminateContainer(rc.Container)
}

// NewTestPool creates a pgxpool connected to the test container and runs migrations
func NewTestPool(ctx context.Context, t *testing.T, pc *PostgresContainer, migrationsDir string) *pgxpool.Pool {
	var pool *pgxpool.Pool
	var err error
	for i := 0; i < 5; i++ {
		pool, err = pgxpool.New(ctx, pc.ConnectionString())
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				break
			}
			pool.Close()
		}
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("failed to create pool after retries: %v", err)
	}

	if err := RunMigrations(ctx, pool, migrationsDir); err != nil {
		pool.Close()
		t.Fatalf("failed to run migrations: %v", err)
	}

	return pool
}

// RunMigrations runs all up migrations from the specified directory
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var upMigrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			upMigrations = append(upMigrations, entry.Name())
		}
	}
	sort.Strings(upMigrations)

	for _, migration := range upMigrations {
		content, err := os.ReadFile(filepath.Join(migrationsDir, migration))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migration, err)
		}

		_, err = pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration, err)
		}
	}

	return nil
}

// TruncateAll truncates all tables in the database for test isolation
func TruncateAll(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{
		"embedding_jobs",
		"knowledge_assets",
		"knowledge_versions",
		"knowledge",
		"assets",
		"api_keys",
		"projects",
		"organizations",
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	return nil
}
