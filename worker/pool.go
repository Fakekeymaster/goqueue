package worker

import (
	"context"
	"log"
	"sync"

	"github.com/Fakekeymaster/goqueue/store"
)

// Pool manages a fixed number of concurrent workers.
// Think of it as a goroutine supervisor.
type Pool struct {
    size    int
    store   *store.Store
    handler Handler
    wg      sync.WaitGroup  // tracks how many workers are still running
    cancel  context.CancelFunc
}

func NewPool(size int, s *store.Store, h Handler) *Pool {
    return &Pool{
        size:    size,
        store:   s,
        handler: h,
    }
}

// Start launches all worker goroutines and blocks until they all stop.
// Pass in a parent context — when it's cancelled, all workers stop.
func (p *Pool) Start(ctx context.Context) {
    // Create a child context we control.
    // When Stop() is called, this context gets cancelled,
    // which propagates to all workers.
    ctx, p.cancel = context.WithCancel(ctx)

    log.Printf("[pool] starting %d workers", p.size)

    for i := 1; i <= p.size; i++ {
        p.wg.Add(1) // tell WaitGroup: one more goroutine starting

        w := newWorker(i, p.store, p.handler)

        // Launch each worker as a goroutine.
        // The 'go' keyword is all it takes — no thread creation,
        // no stack allocation ceremony like in C++.
        go func(w *Worker) {
            defer p.wg.Done() // tell WaitGroup: this goroutine is done
            w.run(ctx)
        }(w)
    }

    // Block here until all workers have exited.
    p.wg.Wait()
    log.Printf("[pool] all workers stopped")
}

// Stop signals all workers to exit and waits for them to finish.
func (p *Pool) Stop() {
    if p.cancel != nil {
        log.Println("[pool] stopping...")
        p.cancel() // cancel the context → all workers see ctx.Done()
    }
    p.wg.Wait() // wait for all goroutines to exit
}