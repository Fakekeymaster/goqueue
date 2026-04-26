package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var (
	flagJobName     string
	flagJobType     string
	flagJobPriority string
	flagAPIAddr     string
)

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a new job to the queue",
	Example: `  goqueue submit --name "resize-img-42" --type image_resize --priority high
  goqueue submit --name "send-welcome" --type email_send --priority medium`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flags before making any network call
		if flagJobName == "" || flagJobType == "" {
			return fmt.Errorf("--name and --type are required")
		}

		// Build the request body — same JSON the curl command sent
		payload := map[string]string{
			"name":     flagJobName,
			"type":     flagJobType,
			"priority": flagJobPriority,
		}

		body, _ := json.Marshal(payload)

		resp, err := http.Post(
			flagAPIAddr+"/jobs",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			return fmt.Errorf("could not reach server at %s — is it running? %w",
				flagAPIAddr, err)
		}
		defer resp.Body.Close()

		// Decode the response into a generic map
		// We use map[string]any instead of queue.Job to avoid
		// importing the queue package — the CLI only talks HTTP,
		// it doesn't need to know internal types
		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("server error: %v", result["error"])
		}

		// Pretty print the response
		fmt.Println("Job submitted successfully")
		fmt.Printf("  ID:       %v\n", result["id"])
		fmt.Printf("  Name:     %v\n", result["name"])
		fmt.Printf("  Type:     %v\n", result["type"])
		fmt.Printf("  Priority: %v\n", result["priority"])
		fmt.Printf("  Status:   %v\n", result["status"])
		return nil
	},
}

func init() {
	submitCmd.Flags().StringVarP(&flagJobName, "name", "n", "", "job name (required)")
	submitCmd.Flags().StringVarP(&flagJobType, "type", "t", "", "job type (required)")
	submitCmd.Flags().StringVarP(&flagJobPriority, "priority", "p", "medium",
		"priority: high | medium | low")
	submitCmd.Flags().StringVar(&flagAPIAddr, "api", "http://localhost:8080",
		"goqueue server address")
	rootCmd.AddCommand(submitCmd)
}