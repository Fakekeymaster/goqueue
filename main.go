package main

import (
	"fmt"
	"time"

	"github.com/Fakekeymaster/goqueue/queue"
)

func main() {
	fmt.Printf("goqueue starting...\n")

	job := queue.Job{
		ID:         "test-123",
		Name:       "send-welcome-email",
		Type:       "email_send",
		Priority:   queue.PriorityHigh,
		Status:     queue.StatusPending,
		Retries:    0,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	fmt.Printf("Job Name:     %s\n", job.Name)
	fmt.Printf("Job Priority: %s\n", job.Priority)
	fmt.Printf("Job Status:   %s\n", job.Status)

}
