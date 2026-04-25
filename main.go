package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/Fakekeymaster/goqueue/config"
    "github.com/Fakekeymaster/goqueue/queue"
    "github.com/Fakekeymaster/goqueue/store"
    "github.com/google/uuid"
)

func main() {
    cfg := config.Load()

    s, err := store.New(cfg)
    if err != nil {
        log.Fatalf("store init failed: %v", err)
    }
    defer s.Close()

    ctx := context.Background()

    // Create and enqueue a job
    job := &queue.Job{
        ID:         uuid.New().String(),
        Name:       "send-welcome-email",
        Type:       "email_send",
        Priority:   queue.PriorityHigh,
        Status:     queue.StatusPending,
        MaxRetries: 3,
        CreatedAt:  time.Now(),
        UpdatedAt:  time.Now(),
    }

    if err := s.Enqueue(ctx, job); err != nil {
        log.Fatalf("enqueue failed: %v", err)
    }
    fmt.Printf("Enqueued job: %s\n", job.ID)

    // Dequeue it back
    fetched, err := s.Dequeue(ctx, 2*time.Second)
    if err != nil {
        log.Fatalf("dequeue failed: %v", err)
    }

    fmt.Printf("Dequeued job: %s\n", fetched.Name)
    fmt.Printf("Priority:     %s\n", fetched.Priority)
    fmt.Printf("Status:       %s\n", fetched.Status)

    // Get stats
    stats, _ := s.GetStats(ctx)
    fmt.Printf("Stats:        enqueued=%s pending=%s\n",
        stats["enqueued"], stats["pending"])
}