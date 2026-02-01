package admin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/pagination"
	"github.com/cloo-solutions/neotexai/internal/repository"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func resolveOrgID(ctx context.Context, orgRepo *repository.OrgRepository, orgRef string) (string, error) {
	if _, err := uuid.Parse(orgRef); err == nil {
		org, err := orgRepo.GetByID(ctx, orgRef)
		if err != nil {
			return "", fmt.Errorf("organization not found: %s", orgRef)
		}
		return org.ID, nil
	}

	org, err := orgRepo.GetByName(ctx, orgRef)
	if err != nil {
		if err == domain.ErrOrganizationNotFound {
			return "", fmt.Errorf("organization not found: %s", orgRef)
		}
		return "", err
	}
	return org.ID, nil
}

func APIKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apikey",
		Short: "Manage API keys",
		Long:  "Create, list, and revoke API keys",
	}

	cmd.AddCommand(APIKeyCreateCmd())
	cmd.AddCommand(APIKeyListCmd())
	cmd.AddCommand(APIKeyRevokeCmd())

	return cmd
}

func APIKeyCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new API key",
		Long:  "Create a new API key for an organization",
		RunE:  runAPIKeyCreate,
	}

	cmd.Flags().StringP("org", "o", "", "Organization ID or name (required)")
	cmd.Flags().StringP("name", "n", "", "API key name (required)")
	cmd.Flags().StringP("output", "", "text", "Output format (text or json)")
	cmd.MarkFlagRequired("org")
	cmd.MarkFlagRequired("name")

	return cmd
}

func runAPIKeyCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	orgRef, _ := cmd.Flags().GetString("org")
	name, _ := cmd.Flags().GetString("name")
	outputFormat, _ := cmd.Flags().GetString("output")

	pool, err := getDBPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	apiKeyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &service.DefaultUUIDGenerator{}
	authSvc := service.NewAuthService(orgRepo, apiKeyRepo, uuidGen)

	orgID, err := resolveOrgID(ctx, orgRepo, orgRef)
	if err != nil {
		return err
	}

	plaintext, err := authSvc.CreateAPIKey(ctx, orgID, name)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	keys, err := authSvc.ListAPIKeys(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to retrieve created key: %w", err)
	}

	var keyID string
	if len(keys) > 0 {
		keyID = keys[len(keys)-1].ID
	}

	if outputFormat == "json" {
		data := map[string]interface{}{
			"id":    keyID,
			"name":  name,
			"org":   orgID,
			"token": plaintext,
		}
		jsonBytes, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("API key created for organization %s\n", orgID)
		fmt.Printf("Key ID: %s\n", keyID)
		fmt.Printf("Key Name: %s\n", name)
		fmt.Printf("Token: %s\n", plaintext)
		fmt.Println("\n⚠️  Save this token now. You won't be able to see it again!")
	}

	return nil
}

func APIKeyListCmd() *cobra.Command {
	var (
		limit  int
		cursor string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API keys for an organization",
		Long:  "List all API keys for a specific organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			orgRef, _ := cmd.Flags().GetString("org")
			outputFormat, _ := cmd.Flags().GetString("output")
			return runAPIKeyList(orgRef, outputFormat, limit, cursor)
		},
	}

	cmd.Flags().StringP("org", "o", "", "Organization ID or name (required)")
	cmd.Flags().StringP("output", "", "text", "Output format (text or json)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Maximum number of results")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor from previous response")
	cmd.MarkFlagRequired("org")

	return cmd
}

func runAPIKeyList(orgRef, outputFormat string, limit int, cursorStr string) error {
	ctx := context.Background()

	pool, err := getDBPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	apiKeyRepo := repository.NewAPIKeyRepository(pool)

	orgID, err := resolveOrgID(ctx, orgRepo, orgRef)
	if err != nil {
		return err
	}

	cursor, _ := pagination.DecodeCursor(cursorStr)
	result, err := apiKeyRepo.ListByOrgWithCursor(ctx, orgID, cursor, limit)
	if err != nil {
		return fmt.Errorf("failed to list API keys: %w", err)
	}

	if outputFormat == "json" {
		data := make([]map[string]interface{}, len(result.Items))
		for i, key := range result.Items {
			data[i] = map[string]interface{}{
				"id":         key.ID,
				"name":       key.Name,
				"org_id":     key.OrgID,
				"created_at": key.CreatedAt,
				"revoked_at": key.RevokedAt,
				"revoked":    key.IsRevoked(),
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
			fmt.Printf("No API keys found for organization %s\n", orgID)
			return nil
		}
		fmt.Printf("API keys for organization %s:\n", orgID)
		for _, key := range result.Items {
			status := "active"
			if key.IsRevoked() {
				status = "revoked"
			}
			fmt.Printf("  %s: %s (%s, created: %s)\n", key.ID, key.Name, status, key.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		if result.HasMore && result.NextCursor != "" {
			fmt.Printf("\nMore results available. Use --cursor %s\n", result.NextCursor)
		}
	}

	return nil
}

func APIKeyRevokeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke an API key",
		Long:  "Revoke an API key by its ID",
		Args:  cobra.ExactArgs(1),
		RunE:  runAPIKeyRevoke,
	}

	cmd.Flags().StringP("output", "", "text", "Output format (text or json)")

	return cmd
}

func runAPIKeyRevoke(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	keyID := args[0]
	outputFormat, _ := cmd.Flags().GetString("output")

	pool, err := getDBPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	apiKeyRepo := repository.NewAPIKeyRepository(pool)
	err = apiKeyRepo.Revoke(ctx, keyID)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	if outputFormat == "json" {
		data := map[string]interface{}{
			"id":      keyID,
			"revoked": true,
			"message": "API key revoked successfully",
		}
		jsonBytes, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("API key %s revoked successfully\n", keyID)
	}

	return nil
}
