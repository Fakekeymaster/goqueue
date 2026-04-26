package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show queue-wide metrics and counters",
	Example: `  goqueue stats
  goqueue stats --api http://localhost:9000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("api")

		resp, err := http.Get(addr + "/stats")
		if err != nil {
			return fmt.Errorf("could not reach server: %w", err)
		}
		defer resp.Body.Close()

		var stats map[string]any
		json.NewDecoder(resp.Body).Decode(&stats)

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("error: %v", stats["error"])
		}

		fmt.Println("Queue metrics")
		fmt.Println("────────────────────────────────")

		// Sort keys for consistent output — maps in Go have
		// random iteration order, so without sorting the output
		// would change every run
		keys := make([]string, 0, len(stats))
		for k := range stats {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Printf("  %-22s %v\n", k+":", stats[k])
		}
		return nil
	},
}

func init() {
	statsCmd.Flags().String("api", "http://localhost:8080", "goqueue server address")
	rootCmd.AddCommand(statsCmd)
}