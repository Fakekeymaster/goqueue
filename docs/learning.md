# GO TaskQ
What is a task queue — in plain words
Imagine you're running a food delivery app. User places an order → you need to:
- Send a confirmation email
- Notify the restaurant
- Update analytics

You can't make the user wait for all that. So instead of doing it immediately, you drop a job into a queue and say "someone handle this later." Workers pick it up and process it in the background.
That's a task queue. Decouple the work from the request.

Producer  ──pushes──►  Queue (Redis)  ──pulls──►  Worker
(your API)              (broker)                   (goroutine)

- Producer — whoever creates the job (your REST API, a CLI command)
- Queue — holds jobs waiting to be processed (we use Redis)
- Worker — picks up jobs and executes them

### Why Redis specifically for the queue?
Redis has a data structure called a List with a special command BRPOP — "Blocking Right Pop." A worker calls BRPOP and just sleeps until a job appears. The moment a producer pushes something, Redis wakes the worker up instantly. Zero polling, zero wasted CPU.
This is the foundation of the whole system.

* More to be added here why redis is used *

We'll build it in 7 stages. Each stage is a working, runnable piece:
1. Project skeleton + go mod init
2. Job struct (what does a job look like?)
3. Redis connection + store layer (push/pop jobs)
4. Single worker (dequeue + process one job)
5. Worker pool (N goroutines, graceful shutdown)
6. REST API (submit jobs via HTTP)
7. CLI with cobra (submit/status/stats from terminal)
8. Makefile + Dockerfile at the end

