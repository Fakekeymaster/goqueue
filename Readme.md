# goqueue

A distributed task queue written in Go, backed by Redis.

I built this to understand how background job processing works under the hood — no frameworks, just Go's standard library, goroutines, and Redis data structures.

---

## What it does

You submit a job (via CLI or HTTP), it gets queued in Redis, and a pool of goroutine workers picks it up and processes it. If a job fails, it retries automatically with exponential backoff. Jobs have three priority levels — high priority jobs always get processed before medium or low.

```
you submit a job
      │
      ▼
Redis queue (high / medium / low)
      │
      ▼
goroutine worker picks it up
      │
      ├── success → mark done
      └── failure → retry with backoff → mark failed after max retries
```

---

## Tech stack

- **Go** — core language, no web framework used
- **Redis** — message broker and job store
- **go-redis/v9** — Redis client
- **cobra** — CLI framework (same one kubectl uses)
- **Docker + docker-compose** — containerized deployment

---

## Project structure

```
goqueue/
├── main.go              entry point
├── config/              env-based configuration
├── queue/               Job struct, Priority and Status types
├── store/               all Redis operations
├── worker/              goroutine pool and job processing logic
├── api/                 REST API server (net/http)
├── cmd/                 CLI commands (server, submit, status, stats)
├── Dockerfile           multi-stage build
└── docker-compose.yml   Redis + goqueue wired together
```

---

## How Redis is used

This was one of the more interesting design decisions. I use four Redis data structures:

| Key | Type | Purpose |
|-----|------|---------|
| `goqueue:queue:high` | LIST | high priority job IDs waiting to be processed |
| `goqueue:queue:medium` | LIST | medium priority job IDs |
| `goqueue:queue:low` | LIST | low priority job IDs |
| `goqueue:job:<id>` | HASH | full job metadata stored as JSON |
| `goqueue:jobs` | SET | registry of all job IDs ever created |
| `goqueue:stats` | HASH | counters: enqueued, completed, failed, etc. |

Workers use `BRPOP` on all three queues in order — `[high, medium, low]`. Redis checks the high queue first and only moves to medium if high is empty. Priority is handled entirely by Redis, no extra logic needed.

`LPUSH` + `BRPOP` gives a FIFO queue within the same priority level. New jobs are pushed to the left, workers pop from the right.

---

## Getting started

### Option 1 — Docker (easiest)

```bash
git clone https://github.com/Fakekeymaster/goqueue.git
cd goqueue
make docker-up
```

This starts Redis and goqueue together. The API is available at `localhost:8080`.

### Option 2 — Run locally

You need Go 1.22+ and Redis running on `localhost:6379`.

```bash
git clone https://github.com/Fakekeymaster/goqueue.git
cd goqueue
make build
make run
```

---

## CLI usage

### Start the server
```bash
./goqueue server --workers 5 --port 8080
```

### Submit a job
```bash
./goqueue submit --name "send-welcome" --type email_send --priority high
./goqueue submit --name "resize-img"   --type image_resize --priority medium
./goqueue submit --name "cleanup"      --type log_cleanup --priority low
```

Output:
```
Job submitted successfully
  ID:       3f2a1b4c-...
  Name:     send-welcome
  Type:     email_send
  Priority: 10
  Status:   pending
```

### Check job status
```bash
./goqueue status --id <job-id>
```

Output:
```
Job details
────────────────────────────────
  ID:         3f2a1b4c-...
  Name:       send-welcome
  Type:       email_send
  Priority:   10
  Status:     done
  Retries:    0 / 3
  Created:    2026-04-26T19:47:22Z
  Updated:    2026-04-26T19:47:23Z
```

### View queue metrics
```bash
./goqueue stats
```

Output:
```
Queue metrics
────────────────────────────────
  completed:             5
  enqueued:              5
  failed:                0
  pending:               0
  queue_high:            0
  queue_low:             0
  queue_medium:          0
  running:               0
  total_jobs:            5
```

---

## REST API

The CLI is just a wrapper around the HTTP API. You can use curl directly too.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | health check |
| POST | `/jobs` | submit a job |
| GET | `/jobs` | list all jobs |
| GET | `/jobs/{id}` | get job by ID |
| GET | `/stats` | queue metrics |

```bash
# submit a job via curl
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"name":"my-job","type":"email_send","priority":"high"}'

# check stats
curl http://localhost:8080/stats
```

---

## Job lifecycle

```
pending → running → done
              │
              └── retrying → pending  (backoff: 2s, 4s, 8s...)
                      │
                      └── failed  (after max retries)
```

Retry backoff is exponential — `2^attempt` seconds, capped at 60s. So if a job fails three times, it waits 2s, then 4s, then 8s before each retry. After `MAX_RETRIES` attempts it's marked permanently failed.

---

## Configuration

Everything is configurable via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_PASS` | `""` | Redis password |
| `REDIS_DB` | `0` | Redis database number |
| `API_PORT` | `8080` | REST API port |
| `WORKER_COUNT` | `5` | number of goroutine workers |
| `MAX_RETRIES` | `3` | max retry attempts per job |

CLI flags override environment variables:
```bash
./goqueue server --workers 10 --port 9000
```

---

## Adding your own job handlers

Right now the default handler just logs and sleeps. To add real handlers, use `HandlerMap` in `worker/worker.go`:

```go
handlers := worker.HandlerMap{
    "email_send":   handleEmail,
    "image_resize": handleImageResize,
    "log_cleanup":  handleLogCleanup,
}

pool := worker.NewPool(cfg.WorkerCount, store, handlers.Dispatch)
```

Each handler is just a function:
```go
func handleEmail(ctx context.Context, job *queue.Job) error {
    // your logic here
    // return error to trigger retry
    // return nil for success
}
```

---

## Makefile targets

```bash
make build       # compile binary
make run         # build + start server
make test        # run tests with race detector
make lint        # go vet
make clean       # remove binary
make docker-up   # start Redis + goqueue via docker-compose
make docker-down # stop containers and remove volumes
make submit      # submit a sample job (server must be running)
make stats       # show live stats (server must be running)
make help        # list all targets
```

---

## What I learned building this

- How `BRPOP` with ordered keys gives you priority queuing for free — no sorting needed
- Why goroutines are so much cheaper than threads — the pool runs 5–100 workers with negligible overhead
- How multi-stage Docker builds cut image size from ~800MB to ~20MB
- The repository pattern — keeping all Redis logic in one place makes it easy to swap backends
- Graceful shutdown — cancelling a context propagates to every goroutine that uses it

---

## License

MIT