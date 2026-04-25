# Redis connection
Redis is our broker — the middleman between producers and workers. 


## Why environment variables and not a config file?
Two reasons. First, secrets (Redis password) should never be in code or committed to git. Second, Docker and Kubernetes set config via env vars — your app works the same way locally and in production without changing a single line of code.

### Note 
Environment variables are always strings, that is why we use `getEnvInt` to convert them to integers.

### What is context?
Context is Go's way of carrying deadlines, cancellation signals, and request-scoped values across function calls. Every Redis operation takes a ctx context.Context as the first argument. If you cancel the context (e.g. server shutting down), all in-flight Redis calls stop cleanly. You'll see this pattern everywhere in Go.

### JSON <-> GO
- **json.Marshal()** 👉 converts Go struct → JSON string
- **json.Unmarshal()** 👉 converts JSON → Go struct

### Store wraps the Redis client and exposes
only the operations our app needs.
This is called the "repository pattern" —
he rest of the app never touches Redis directly,
only through this Store. Makes testing and swapping
the backend easy.

### Why wrap the Redis client instead of using it directly everywhere?
If you used redis.Client directly in your worker, API, and CLI — and you later wanted to switch from Redis to RabbitMQ — you'd have to change code in 10 places. With a Store, you change one file. This is the dependency inversion principle — depend on an abstraction, not a concrete implementation.

#### Redis keys pattern
- goqueue:queue:high    → LIST   (job IDs waiting to be processed)
- goqueue:queue:medium  → LIST   (job IDs waiting to be processed)
- goqueue:queue:low     → LIST   (job IDs waiting to be processed)
- goqueue:job:<id>      → HASH   (full job metadata as JSON)
- goqueue:jobs          → SET    (all job IDs ever created)
- goqueue:stats         → HASH   (counters: enqueued, done, failed...)

#### What just happened end to end:
- Connected to Redis on localhost:6379
- Serialized a Job struct to JSON and pushed its ID to goqueue:queue:high
- Stored the JSON in goqueue:job:<id>
- Added the ID to goqueue:jobs
- BRPOP pulled the ID back from the high queue
- Fetched and deserialized the full job from Redis

### The Enqueue Flow (Go → Redis)
When you call s.Enqueue(ctx, job), several things happen inside Redis via a Pipeline (one big batch):

- Metadata Storage (The Vault): The job is converted to JSON and stored in a Redis Hash at goqueue:job:<ID>. This is the "source of truth" containing the status, retry count, etc.
- The Index: The Job ID is added to a Redis Set (goqueue:jobs). This is just a list of all jobs so we can find them later.
- The Waiting Line (The Queue): The Job ID is pushed into a Redis List (e.g., goqueue:queue:high).
Note: We don't push the whole JSON into the list; we only push the ID to keep the queue lightweight.

### The Dequeue Flow (Redis → Go)
When a worker (or your main.go test) calls s.Dequeue(...):

- The Wait: Redis looks at the goqueue:queue:* lists. If they are empty, it makes your Go program "sleep" (BRPOP).
- The Grab: The moment a Job ID appears in a list, Redis "pops" it out and hands the ID back to Go.
- The Hydration: Your code then uses that ID to look up the full JSON metadata from the Vault (the Hash we made in step 2) and converts it back into a Go struct.