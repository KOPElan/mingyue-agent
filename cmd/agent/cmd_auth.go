package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication and token management",
		Long:  "Manage API tokens and authentication",
	}

	cmd.AddCommand(authTokenCreateCmd())
	cmd.AddCommand(authTokenListCmd())
	cmd.AddCommand(authTokenRevokeCmd())

	return cmd
}

func authTokenCreateCmd() *cobra.Command {
	var (
		userID    string
		expiresIn int
	)

	cmd := &cobra.Command{
		Use:   "token-create <name>",
		Short: "Create a new API token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			name := args[0]

			body := map[string]interface{}{
				"user_id":    userID,
				"name":       name,
				"expires_in": expiresIn,
			}

			resp, err := client.Post("/api/v1/auth/tokens/create", body)
			if err != nil {
				return err
			}

			var result struct {
				Token     string `json:"token"`
				TokenID   string `json:"token_id"`
				ExpiresAt string `json:"expires_at"`
			}

			if err := json.Unmarshal(resp.Data, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("Token created successfully:\n")
			fmt.Printf("  Token ID:   %s\n", result.TokenID)
			fmt.Printf("  Token:      %s\n", result.Token)
			fmt.Printf("  Expires at: %s\n", result.ExpiresAt)
			fmt.Println("\nIMPORTANT: Save this token now. You won't be able to see it again!")

			return nil
		},
	}

	cmd.Flags().StringVarP(&userID, "user", "u", "admin", "User ID")
	cmd.Flags().IntVarP(&expiresIn, "expires", "e", 31536000, "Token expiration in seconds (default: 1 year)")

	return cmd
}

func authTokenListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "token-list",
		Short: "List all API tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()

			resp, err := client.Get("/api/v1/auth/tokens")
			if err != nil {
				return err
			}

			var tokens []struct {
				ID        string `json:"id"`
				UserID    string `json:"user_id"`
				Name      string `json:"name"`
				CreatedAt string `json:"created_at"`
				ExpiresAt string `json:"expires_at"`
			}

			if err := json.Unmarshal(resp.Data, &tokens); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if len(tokens) == 0 {
				fmt.Println("No API tokens")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tUSER\tNAME\tCREATED\tEXPIRES")
			for _, t := range tokens {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					t.ID, t.UserID, t.Name, t.CreatedAt, t.ExpiresAt)
			}
			w.Flush()

			return nil
		},
	}
}

func authTokenRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "token-revoke <token-id>",
		Short: "Revoke an API token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			tokenID := args[0]

			_, err := client.Post("/api/v1/auth/tokens/revoke", map[string]string{
				"token_id": tokenID,
			})
			if err != nil {
				return err
			}

			fmt.Printf("Token %s revoked\n", tokenID)

			return nil
		},
	}
}
