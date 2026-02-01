package admin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloo-solutions/neotexai/internal/config"
	"github.com/cloo-solutions/neotexai/internal/pagination"
	"github.com/cloo-solutions/neotexai/internal/repository"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

func OrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage organizations",
		Long:  "Create and list organizations",
	}

	cmd.AddCommand(OrgCreateCmd())
	cmd.AddCommand(OrgListCmd())

	return cmd
}

func OrgCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new organization",
		Long:  "Create a new organization with the specified name",
		Args:  cobra.ExactArgs(1),
		RunE:  runOrgCreate,
	}

	cmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	return cmd
}

func runOrgCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	name := args[0]
	outputFormat, _ := cmd.Flags().GetString("output")

	pool, err := getDBPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	uuidGen := &service.DefaultUUIDGenerator{}
	authSvc := service.NewAuthService(orgRepo, nil, uuidGen)

	org, err := authSvc.CreateOrg(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	if outputFormat == "json" {
		data := map[string]interface{}{
			"id":         org.ID,
			"name":       org.Name,
			"created_at": org.CreatedAt,
		}
		jsonBytes, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("Organization created: %s (%s)\n", org.Name, org.ID)
	}

	return nil
}

func OrgListCmd() *cobra.Command {
	var (
		limit  int
		cursor string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all organizations",
		Long:  "List all organizations in the system",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, _ := cmd.Flags().GetString("output")
			return runOrgList(outputFormat, limit, cursor)
		},
	}

	cmd.Flags().StringP("output", "o", "text", "Output format (text or json)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Maximum number of results")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor from previous response")

	return cmd
}

func runOrgList(outputFormat string, limit int, cursorStr string) error {
	ctx := context.Background()

	pool, err := getDBPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)

	cursor, _ := pagination.DecodeCursor(cursorStr)
	result, err := orgRepo.ListWithCursor(ctx, cursor, limit)
	if err != nil {
		return fmt.Errorf("failed to list organizations: %w", err)
	}

	if outputFormat == "json" {
		data := make([]map[string]interface{}, len(result.Items))
		for i, org := range result.Items {
			data[i] = map[string]interface{}{
				"id":         org.ID,
				"name":       org.Name,
				"created_at": org.CreatedAt,
			}
		}
		output := map[string]interface{}{
			"items":    data,
			"cursor":   result.NextCursor,
			"has_more": result.HasMore,
		}
		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		if len(result.Items) == 0 {
			fmt.Println("No organizations found")
			return nil
		}
		fmt.Println("Organizations:")
		for _, org := range result.Items {
			fmt.Printf("  %s: %s (created: %s)\n", org.ID, org.Name, org.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		if result.HasMore && result.NextCursor != "" {
			fmt.Printf("\nMore results available. Use --cursor %s\n", result.NextCursor)
		}
	}

	return nil
}

func getDBPool(ctx context.Context) (*pgxpool.Pool, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
