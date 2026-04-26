# The Worker

It will run in an infinit loop doing:
1. block on BRPOP waiting for a job
2. process the job (call the handler)
3. update job status (done / retry / failed)

## run function 
### Why 2 second timeout on Dequeue instead of blocking forever?
If we blocked forever and Ctrl+C was pressed, the worker would be stuck in BRPOP and never check ctx.Done(). The 2 second timeout means: "wait for a job, but wake up every 2 seconds to check if we should shut down." It's a heartbeat.
### What is select?
select is like a switch but for channels. It waits for whichever channel is ready first. Here we check ctx.Done() (a channel that closes on cancellation) and default (runs immediately if nothing else is ready). This gives us a non-blocking shutdown check.

### What is sync.WaitGroup?
A WaitGroup is a counter. You call Add(1) before launching a goroutine, Done() when it finishes, and Wait() to block until the counter hits zero. It's how the main goroutine knows all workers have cleanly exited before the program terminates.
In C++ terms, it's like joining all threads before exit — but much simpler.

###Why pass w as an argument to the goroutine instead of closing over it?
Classic Go gotcha. If you wrote:
gofor i := 1; i <= p.size; i++ {
    w := newWorker(i, ...)
    go func() {
        w.run(ctx) // WRONG — all goroutines share the same 'w'
    }()
}