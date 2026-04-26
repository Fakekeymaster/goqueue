package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command — called when no subcommand is given.
// Running just "goqueue" with no args shows this help text.
var rootCmd = &cobra.Command{
	Use:   "goqueue",
	Short: "goqueue — distributed task queue backed by Redis",
	Long: `goqueue is a distributed task queue written in Go.

It uses Redis as a broker with three priority levels (high, medium, low),
a goroutine-based worker pool, exponential backoff retry, and a REST API.

Examples:
  goqueue server --workers 5 --port 8080
  goqueue submit --name "my-job" --type email_send --priority high
  goqueue status --id <job-id>
  goqueue stats`,
}

// Execute is called from main.go.
// It parses os.Args, finds the right subcommand, and runs it.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}