package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var flagJobID string

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Check the status of a job by ID",
	Example: `  goqueue status --id <job-id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagJobID == "" {
			return fmt.Errorf("--id is required")
		}

		addr, _ := cmd.Flags().GetString("api")

		resp, err := http.Get(addr + "/jobs/" + flagJobID)
		if err != nil {
			return fmt.Errorf("could not reach server: %w", err)
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("error: %v", result["error"])
		}

		fmt.Println("Job details")
		fmt.Println("────────────────────────────────")
		fmt.Printf("  ID:         %v\n", result["id"])
		fmt.Printf("  Name:       %v\n", result["name"])
		fmt.Printf("  Type:       %v\n", result["type"])
		fmt.Printf("  Priority:   %v\n", result["priority"])
		fmt.Printf("  Status:     %v\n", result["status"])
		fmt.Printf("  Retries:    %v / %v\n", result["retries"], result["max_retries"])
		fmt.Printf("  Created:    %v\n", result["created_at"])
		fmt.Printf("  Updated:    %v\n", result["updated_at"])

		// Only print error field if it exists and is non-empty
		if e, ok := result["error"].(string); ok && e != "" {
			fmt.Printf("  Error:      %v\n", e)
		}
		return nil
	},
}

func init() {
	statusCmd.Flags().StringVar(&flagJobID, "id", "", "job ID (required)")
	statusCmd.Flags().String("api", "http://localhost:8080", "goqueue server address")
	rootCmd.AddCommand(statusCmd)
}