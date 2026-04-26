package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Fakekeymaster/goqueue/api"
	"github.com/Fakekeymaster/goqueue/config"
	"github.com/Fakekeymaster/goqueue/store"
	"github.com/Fakekeymaster/goqueue/worker"
	"github.com/spf13/cobra"
)

// flagWorkers and flagPort are package-level variables
// that cobra populates from CLI flags before RunE executes.
var (
	flagWorkers int
	flagPort    string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server and worker pool",
	// RunE is like Run but can return an error.
	// Cobra prints the error and exits with code 1 automatically.
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// CLI flags override environment variables
		if flagWorkers > 0 {
			cfg.WorkerCount = flagWorkers
		}
		if flagPort != "" {
			cfg.APIPort = flagPort
		}

		s, err := store.New(cfg)
		if err != nil {
			return err
		}
		defer s.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		pool := worker.NewPool(cfg.WorkerCount, s, nil)
		go pool.Start(ctx)

		apiServer := api.NewServer(cfg, s)
		go func() {
			if err := apiServer.Start(); err != nil {
				log.Printf("[server] api stopped: %v", err)
			}
		}()

		log.Printf("[server] goqueue running — workers=%d port=%s redis=%s",
			cfg.WorkerCount, cfg.APIPort, cfg.RedisAddr)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("[server] shutting down...")
		cancel()

		shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutCancel()
		apiServer.Shutdown(shutCtx)
		pool.Stop()

		log.Println("[server] bye")
		return nil
	},
}

// init registers serverCmd onto rootCmd and defines its flags.
// This runs automatically before main().
func init() {
	serverCmd.Flags().IntVarP(&flagWorkers, "workers", "w", 0,
		"number of worker goroutines (default: from env or 5)")
	serverCmd.Flags().StringVarP(&flagPort, "port", "p", "",
		"API port (default: from env or 8080)")
	rootCmd.AddCommand(serverCmd)
}