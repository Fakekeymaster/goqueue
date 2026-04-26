package main

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
)

func main() {
	cfg := config.Load()

	// Connect to Redis
	s, err := store.New(cfg)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer s.Close()

	// Root context — cancelling this shuts everything down
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker pool in background
	pool := worker.NewPool(cfg.WorkerCount, s, nil)
	go pool.Start(ctx)

	// Start API server in background
	apiServer := api.NewServer(cfg, s)
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Printf("[main] api server stopped: %v", err)
		}
	}()

	log.Printf("[main] goqueue running — workers=%d port=%s redis=%s",
		cfg.WorkerCount, cfg.APIPort, cfg.RedisAddr)

	// Block until SIGINT (Ctrl+C) or SIGTERM (docker stop)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // wait here

	log.Println("[main] shutting down...")
	cancel() // stop workers

	// Give HTTP server 10 seconds to finish in-flight requests
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	apiServer.Shutdown(shutCtx)

	pool.Stop()
	log.Println("[main] bye")
}