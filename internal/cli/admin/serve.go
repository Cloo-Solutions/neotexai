package admin

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloo-solutions/neotexai/internal/api/handlers"
	"github.com/cloo-solutions/neotexai/internal/config"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/jobs"
	"github.com/cloo-solutions/neotexai/internal/openai"
	"github.com/cloo-solutions/neotexai/internal/repository"
	"github.com/cloo-solutions/neotexai/internal/server"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/cloo-solutions/neotexai/internal/storage"
	"github.com/cloo-solutions/neotexai/internal/telemetry"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/cobra"
)

// ServeCmd returns the serve command
func ServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the API server",
		Long:  "Start the neotex API server on the specified port",
		RunE:  runServe,
	}

	cmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	cmd.Flags().Bool("no-migrate", false, "Skip automatic database migrations on startup")

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize Sentry with tracing if SENTRY_DSN is set
	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		environment := os.Getenv("ENVIRONMENT")
		if environment == "" {
			environment = "development"
		}

		// Default to 10% sampling in production, 100% in development
		sampleRate := 0.1
		if environment == "development" {
			sampleRate = 1.0
		}

		shutdownTelemetry, err := telemetry.Init(telemetry.Config{
			DSN:              dsn,
			Environment:      environment,
			TracesSampleRate: sampleRate,
		})
		if err != nil {
			log.Printf("telemetry init failed (continuing without tracing): %v", err)
		} else {
			defer shutdownTelemetry()
		}
	}

	portFlag, _ := cmd.Flags().GetString("port")
	if portFlag != "" && portFlag != "8080" {
		cfg.Port = portFlag
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	log.Println("connected to database")

	// Run migrations unless --no-migrate flag is set
	noMigrate, _ := cmd.Flags().GetBool("no-migrate")
	if !noMigrate {
		if err := runMigrations(cfg.DatabaseURL); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	knowledgeRepo := repository.NewKnowledgeRepository(pool)
	embeddingJobRepo := repository.NewEmbeddingJobRepository(pool)
	assetRepo := repository.NewAssetRepository(pool)
	orgRepo := repository.NewOrgRepository(pool)
	apiKeyRepo := repository.NewAPIKeyRepository(pool)
	contextRepo := repository.NewContextRepository(pool)
	projectRepo := repository.NewProjectRepository(pool)

	if cfg.InitOrgName != "" {
		if err := bootstrapInitialOrg(ctx, cfg, orgRepo, apiKeyRepo); err != nil {
			return fmt.Errorf("failed to bootstrap initial org: %w", err)
		}
	}

	var storageClient service.StorageClientInterface
	if cfg.HasS3() {
		s3Config := storage.S3ClientConfig{
			Endpoint:        cfg.S3Endpoint,
			Region:          cfg.S3Region,
			AccessKeyID:     cfg.S3AccessKey,
			SecretAccessKey: cfg.S3SecretKey,
			Bucket:          cfg.S3Bucket,
			UsePathStyle:    true,
		}
		s3Client, err := storage.NewS3Client(ctx, s3Config)
		if err != nil {
			return fmt.Errorf("failed to create S3 client: %w", err)
		}
		if err := s3Client.EnsureBucket(ctx); err != nil {
			return fmt.Errorf("failed to ensure S3 bucket: %w", err)
		}
		log.Printf("S3 bucket '%s' ready", cfg.S3Bucket)
		storageClient = &S3StorageAdapter{client: s3Client}
	}

	var embeddingClient service.EmbeddingClient
	var embeddingWorker *jobs.Worker
	if cfg.HasOpenAI() {
		embeddingClient = openai.NewClient(cfg.OpenAIAPIKey)
		embeddingSvc := service.NewEmbeddingServiceWithAssets(embeddingClient, knowledgeRepo, assetRepo)
		embeddingProcessor := jobs.NewEmbeddingWorker(embeddingJobRepo, embeddingSvc)
		embeddingWorker = jobs.NewWorker(embeddingProcessor, 10*time.Second)
		go embeddingWorker.Start(ctx)
		log.Println("embedding worker started")
	}

	uuidGen := &service.DefaultUUIDGenerator{}

	knowledgeSvc := service.NewKnowledgeService(knowledgeRepo, embeddingJobRepo)
	var assetSvc *service.AssetService
	if storageClient != nil {
		assetSvc = service.NewAssetServiceWithEmbeddings(assetRepo, storageClient, embeddingJobRepo)
	}
	authSvc := service.NewAuthService(orgRepo, apiKeyRepo, uuidGen)

	knowledgeHandler := handlers.NewKnowledgeHandler(knowledgeSvc)
	var assetHandler *handlers.AssetHandler
	if assetSvc != nil {
		assetHandler = handlers.NewAssetHandler(assetSvc)
	} else {
		assetHandler = handlers.NewAssetHandler(&NoOpAssetService{})
	}
	authHandler := handlers.NewAuthHandler(authSvc)
	projectHandler := handlers.NewProjectHandler(projectRepo)

	var contextHandler *handlers.ContextHandler
	if embeddingClient != nil {
		contextHandler = handlers.NewContextHandler(service.NewContextService(contextRepo, embeddingClient))
	} else {
		contextHandler = handlers.NewContextHandler(&NoOpContextService{})
	}

	routerCfg := server.RouterConfig{
		AuthValidator:    authSvc,
		KnowledgeHandler: knowledgeHandler,
		AssetHandler:     assetHandler,
		ContextHandler:   contextHandler,
		AuthHandler:      authHandler,
		ProjectHandler:   projectHandler,
	}

	router := server.NewRouter(routerCfg)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("starting server on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	if embeddingWorker != nil {
		embeddingWorker.Stop()
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("server exited")
	return nil
}

type S3StorageAdapter struct {
	client *storage.S3Client
}

func (a *S3StorageAdapter) GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error) {
	return a.client.GenerateUploadURL(ctx, key, contentType)
}

func (a *S3StorageAdapter) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
	return a.client.GenerateDownloadURL(ctx, key)
}

func (a *S3StorageAdapter) DeleteObject(ctx context.Context, key string) error {
	return a.client.DeleteObject(ctx, key)
}

func (a *S3StorageAdapter) HeadObject(ctx context.Context, key string) (*service.ObjectMetadata, error) {
	meta, err := a.client.HeadObject(ctx, key)
	if err != nil {
		return nil, err
	}
	return &service.ObjectMetadata{
		ContentLength: meta.ContentLength,
		ContentType:   meta.ContentType,
		ETag:          meta.ETag,
	}, nil
}

type NoOpAssetService struct{}

func (s *NoOpAssetService) InitUpload(ctx context.Context, input service.InitUploadInput) (*service.InitUploadResult, error) {
	return nil, fmt.Errorf("asset service not configured: S3_ENDPOINT required")
}

func (s *NoOpAssetService) CompleteUpload(ctx context.Context, input service.CompleteUploadInput) (*domain.Asset, error) {
	return nil, fmt.Errorf("asset service not configured: S3_ENDPOINT required")
}

func (s *NoOpAssetService) GetDownloadURL(ctx context.Context, assetID string) (string, error) {
	return "", fmt.Errorf("asset service not configured: S3_ENDPOINT required")
}

func (s *NoOpAssetService) GetByID(ctx context.Context, assetID string) (*domain.Asset, error) {
	return nil, fmt.Errorf("asset service not configured: S3_ENDPOINT required")
}

type NoOpContextService struct{}

func (s *NoOpContextService) GetManifest(ctx context.Context, orgID, projectID string) ([]*service.KnowledgeManifestItem, error) {
	return nil, fmt.Errorf("context service not configured: embedding provider required")
}

func (s *NoOpContextService) Search(ctx context.Context, input service.SearchInput) (*service.SearchOutput, error) {
	return nil, fmt.Errorf("context service not configured: embedding provider required")
}

func bootstrapInitialOrg(ctx context.Context, cfg *config.Config, orgRepo *repository.OrgRepository, apiKeyRepo *repository.APIKeyRepository) error {
	org, err := orgRepo.GetByName(ctx, cfg.InitOrgName)
	if err != nil && err != domain.ErrOrganizationNotFound {
		return fmt.Errorf("failed to check existing org: %w", err)
	}

	uuidGen := &service.DefaultUUIDGenerator{}
	authSvc := service.NewAuthService(orgRepo, apiKeyRepo, uuidGen)

	if org == nil {
		org, err = authSvc.CreateOrg(ctx, cfg.InitOrgName)
		if err != nil {
			return fmt.Errorf("failed to create org: %w", err)
		}
		log.Printf("bootstrap: created organization '%s' (id: %s)", org.Name, org.ID)
	} else {
		log.Printf("bootstrap: organization '%s' already exists (id: %s)", org.Name, org.ID)
	}

	if cfg.InitAPIKey != "" {
		if !service.IsValidAPIToken(cfg.InitAPIKey) {
			return fmt.Errorf("invalid NEOTEX_INIT_API_KEY format (expected 'ntx_<64 hex chars>')")
		}

		existingKey, err := authSvc.GetAPIKeyByHash(ctx, cfg.InitAPIKey)
		if err == nil && existingKey != nil {
			log.Printf("bootstrap: API key already exists (id: %s)", existingKey.ID)
			return nil
		}

		if err := authSvc.CreateAPIKeyWithToken(ctx, org.ID, "bootstrap", cfg.InitAPIKey); err != nil {
			return fmt.Errorf("failed to create API key: %w", err)
		}
		log.Printf("bootstrap: created API key")
	}

	return nil
}

func runMigrations(databaseURL string) error {
	// Create a sql.DB connection for golang-migrate
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer db.Close()

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create migrate instance with file source
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Get migration version and status
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if err == migrate.ErrNilVersion {
		log.Println("migrations: database is up to date (no migrations applied)")
	} else if dirty {
		return fmt.Errorf("migration version %d is dirty - manual intervention required", version)
	} else if err == migrate.ErrNoChange {
		log.Printf("migrations: database is up to date (version %d)", version)
	} else {
		log.Printf("migrations: applied successfully (version %d)", version)
	}

	return nil
}
