package main

import (
    "context"
    "log"
    "time"

    "github.com/Fakekeymaster/goqueue/config"
    "github.com/Fakekeymaster/goqueue/queue"
    "github.com/Fakekeymaster/goqueue/store"
    "github.com/Fakekeymaster/goqueue/worker"
    "github.com/google/uuid"
)

func main() {
    cfg := config.Load()

    s, err := store.New(cfg)
    if err != nil {
        log.Fatalf("store: %v", err)
    }
    defer s.Close()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Enqueue 5 jobs at different priorities
    jobs := []struct {
        name     string
        jobType  string
        priority queue.Priority
    }{
        {"urgent-report",  "report_gen",    queue.PriorityHigh},
        {"welcome-email",  "email_send",    queue.PriorityHigh},
        {"resize-avatar",  "image_resize",  queue.PriorityMedium},
        {"weekly-digest",  "email_send",    queue.PriorityMedium},
        {"cleanup-logs",   "log_cleanup",   queue.PriorityLow},
    }

    for _, j := range jobs {
        job := &queue.Job{
            ID:         uuid.New().String(),
            Name:       j.name,
            Type:       j.jobType,
            Priority:   j.priority,
            Status:     queue.StatusPending,
            MaxRetries: cfg.MaxRetries,
            CreatedAt:  time.Now(),
            UpdatedAt:  time.Now(),
        }
        if err := s.Enqueue(ctx, job); err != nil {
            log.Fatalf("enqueue: %v", err)
        }
        log.Printf("enqueued: %-20s priority=%s", job.Name, job.Priority)
    }

    // Start pool with 3 workers — let it run for 5 seconds then stop
    log.Println("--- starting worker pool ---")
    pool := worker.NewPool(3, s, nil)

    // Run pool in a goroutine so we can stop it after a timeout
    go pool.Start(ctx)

    // Let workers process for 5 seconds
    time.Sleep(5 * time.Second)

    log.Println("--- stopping pool ---")
    cancel() // signal all workers to stop
    pool.Stop()

    log.Println("--- done ---")
}