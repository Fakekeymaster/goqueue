package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/Fakekeymaster/goqueue/queue"
	"github.com/Fakekeymaster/goqueue/store"
)

// Handler is a function type that processes a job.
// Returning an error triggers the retry logic.
// Returning nil means success.
//
// This is the only thing callers need to implement —
// the worker handles all the lifecycle around it.
type Handler func(ctx context.Context, job *queue.Job) error

//DefaultHandler simulates job processing
func defaultHandler(ctx context.Context, job *queue.Job) error {
	log.Printf("[worker] processing type=%-15s name=%s", job.Type, job.Name)
	time.Sleep(500 * time.Millisecond)	//work simulation
	return nil
}

type Worker struct {
    id      int          // for log identification
    store   *store.Store // to fetch and update jobs
    handler Handler      // what to do with each job
}

// newWorker creates a Worker. If no handler provided, use the default.
func newWorker(id int, s *store.Store, h Handler) *Worker {
    if h == nil {
        h = defaultHandler
    }
    return &Worker{id: id, store: s, handler: h}
}



// run is the worker's main loop.
// It blocks on Dequeue, processes jobs, and repeats.
// It exits only when ctx is cancelled (graceful shutdown).
func (w *Worker) run(ctx context.Context) {
    log.Printf("[worker-%d] started", w.id)

    for {
        // Check if shutdown was requested before trying to dequeue.
        // select with a default case is non-blocking —
        // it checks ctx.Done() and moves on immediately if not cancelled.
        select {
        case <-ctx.Done():
            log.Printf("[worker-%d] shutting down", w.id)
            return
        default:
            // not cancelled, continue
        }

        // Block waiting for a job — but only for 2 seconds.
        // After 2 seconds with no job, loop back and check ctx again.
        // This is how we stay responsive to shutdown signals
        // even when the queue is empty.
        job, err := w.store.Dequeue(ctx, 2*time.Second)
        if err != nil {
            if ctx.Err() != nil {
                // context was cancelled during dequeue — clean exit
                return
            }
            log.Printf("[worker-%d] dequeue error: %v", w.id, err)
            continue
        }

        if job == nil {
            // timeout — no jobs available, loop again
            continue
        }

        // We have a job — process it
        w.process(ctx, job)
    }
}

// process handles the full lifecycle of one job —
// running it, handling success, failure, and retry.
func (w *Worker) process(ctx context.Context, job *queue.Job) {
    // Mark job as running and record start time
    now := time.Now()
    job.Status = queue.StatusRunning
    job.StartedAt = &now
    w.store.UpdateJob(ctx, job)
    w.store.DecrStat(ctx, "pending")
    w.store.IncrStat(ctx, "running")

    log.Printf("[worker-%d] started job id=%.8s type=%s", w.id, job.ID, job.Type)

    // Call the handler — this is where actual work happens
    err := w.handler(ctx, job)

    // Handler finished — decrement running count
    w.store.DecrStat(ctx, "running")

    if err == nil {
        // Success path
        completed := time.Now()
        job.Status = queue.StatusDone
        job.CompletedAt = &completed
        w.store.UpdateJob(ctx, job)
        w.store.IncrStat(ctx, "completed")
        log.Printf("[worker-%d] job done id=%.8s", w.id, job.ID)
        return
    }

    // Failure path — decide whether to retry or give up
    job.Retries++
    job.Error = err.Error()

    if job.Retries >= job.MaxRetries {
        // Exceeded retry limit — permanently failed
        job.Status = queue.StatusFailed
        w.store.UpdateJob(ctx, job)
        w.store.IncrStat(ctx, "failed")
        log.Printf("[worker-%d] job failed id=%.8s retries=%d error=%s",
            w.id, job.ID, job.Retries, job.Error)
        return
    }

    // Still have retries left — calculate backoff and requeue
    backoff := backoffDuration(job.Retries)
    job.Status = queue.StatusRetrying
    w.store.UpdateJob(ctx, job)
    w.store.IncrStat(ctx, "retried")

    log.Printf("[worker-%d] retrying id=%.8s attempt=%d backoff=%s",
        w.id, job.ID, job.Retries, backoff)

    // Wait for backoff duration, but exit early if shutdown requested
    select {
    case <-time.After(backoff):
        w.store.RequeueJob(ctx, job)
        w.store.IncrStat(ctx, "pending")
    case <-ctx.Done():
        return
    }
}


// backoffDuration returns 2^attempt seconds, capped at 60s.
//
// attempt 1 →  2s
// attempt 2 →  4s
// attempt 3 →  8s
// attempt 4 → 16s
// attempt 5 → 32s
// attempt 6 → 60s (capped)
//
// This is called "exponential backoff" — standard in distributed systems.
// The idea: if a job fails, the downstream service is probably struggling.
// Waiting longer between retries gives it time to recover.
func backoffDuration(attempt int) time.Duration {
    seconds := math.Pow(2, float64(attempt))
    if seconds > 60 {
        seconds = 60
    }
    return time.Duration(seconds) * time.Second
}


// HandlerMap lets you register different handlers per job type.
// Instead of one handler for everything, you can have:
//   "email_send"   → handleEmail
//   "image_resize" → handleResize
//
// Usage:
//   handlers := worker.HandlerMap{
//       "email_send": myEmailHandler,
//   }
//   pool := worker.NewPool(5, store, handlers.Dispatch)
type HandlerMap map[string]Handler

func (hm HandlerMap) Dispatch(ctx context.Context, job *queue.Job) error {
    h, ok := hm[job.Type]
    if !ok {
        return fmt.Errorf("no handler registered for job type %q", job.Type)
    }
    return h(ctx, job)
}