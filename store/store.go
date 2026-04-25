package store

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/Fakekeymaster/goqueue/config"
    "github.com/Fakekeymaster/goqueue/queue"
    "github.com/redis/go-redis/v9"
)

//All Redis key patterns in one place
const (
    keyJobMeta     = "goqueue:job:%s"       // %s = job ID
    keyQueueHigh   = "goqueue:queue:high"
    keyQueueMedium = "goqueue:queue:medium"
    keyQueueLow    = "goqueue:queue:low"
    keyAllJobs     = "goqueue:jobs"
    keyStats       = "goqueue:stats"
)

type Store struct {
	rdb *redis.Client
}

// New creates a Store and verifies the Redis connection and fails fast
func New(cfg *config.Config) (*Store, error) {
	    rdb := redis.NewClient(&redis.Options{
        Addr:     cfg.RedisAddr,
        Password: cfg.RedisPass,
        DB:       cfg.RedisDB,
    })

    // context.WithTimeout gives the Ping 5 seconds to succeed.
    // If Redis doesn't respond in time, we return an error.
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer means: “run this line later, just before the function ends”
    defer cancel() // always clean up the context when done

    if err := rdb.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("redis connection failed: %w", err)
    }

    return &Store{rdb: rdb}, nil
}

func (s *Store) Close() error {
	return s.rdb.Close()
}

// Enqueue adds a job to Redis atomically using a pipeline.
// A pipeline sends multiple commands to Redis in one network round-trip
// and executes them together — much faster than sending one by one.
func (s *Store) Enqueue(ctx context.Context, job *queue.Job) error {
    // Serialize the job struct to JSON bytes for storage.
    data, err := json.Marshal(job)
    if err != nil {
        return fmt.Errorf("marshal job: %w", err)
    }

    // Pipeline = batch multiple Redis commands into one round-trip.
    pipe := s.rdb.Pipeline()

    // Store full job metadata as a JSON blob in a HASH.
    pipe.HSet(ctx, fmt.Sprintf(keyJobMeta, job.ID), "data", data)

    // Add job ID to the master SET so we can list all jobs later.
    pipe.SAdd(ctx, keyAllJobs, job.ID)

    // Push the job ID to the front of the correct priority LIST.
    // Workers will pop from the back (BRPOP = blocking right pop).
    // LPUSH + BRPOP = FIFO queue within same priority level.
    pipe.LPush(ctx, queueKey(job.Priority), job.ID)

    // Increment counters for stats.
    pipe.HIncrBy(ctx, keyStats, "enqueued", 1)
    pipe.HIncrBy(ctx, keyStats, "pending", 1)

    // Execute all 5 commands in one shot.
    _, err = pipe.Exec(ctx)
    return err
}

// Dequeue blocks waiting for a job.
// BRPOP checks the high queue first, then medium, then low.
// If all queues are empty, it sleeps until a job arrives or timeout.
// This is the most elegant part of the whole system —
// no polling loop, no wasted CPU. Redis does the waiting.
func (s *Store) Dequeue(ctx context.Context, timeout time.Duration) (*queue.Job, error) {
    queues := []string{keyQueueHigh, keyQueueMedium, keyQueueLow}

    // BRPOP takes a list of keys and a timeout.
    // It checks each key in order — returns the first non-empty one.
    // If all are empty, it blocks until timeout.
    result, err := s.rdb.BRPop(ctx, timeout, queues...).Result()
    if err == redis.Nil {
        // redis.Nil means timeout reached, no jobs available. Not an error.
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("brpop: %w", err)
    }

    // BRPOP returns [queueName, jobID] — we want index 1.
    jobID := result[1]
    return s.GetJob(ctx, jobID)
}

// GetJob fetches a job's full metadata by ID.
func (s *Store) GetJob(ctx context.Context, id string) (*queue.Job, error) {
    data, err := s.rdb.HGet(ctx, fmt.Sprintf(keyJobMeta, id), "data").Bytes()
    if err == redis.Nil {
        return nil, fmt.Errorf("job %s not found", id)
    }
    if err != nil {
        return nil, err
    }

    var job queue.Job
    if err := json.Unmarshal(data, &job); err != nil {
        return nil, fmt.Errorf("unmarshal job: %w", err)
    }
    return &job, nil
}

// UpdateJob saves the current state of a job back to Redis.
// Called every time a job's status changes.
func (s *Store) UpdateJob(ctx context.Context, job *queue.Job) error {
    job.UpdatedAt = time.Now()
    data, err := json.Marshal(job)
    if err != nil {
        return err
    }
    return s.rdb.HSet(ctx, fmt.Sprintf(keyJobMeta, job.ID), "data", data).Err()
}

// RequeueJob pushes a job back onto its priority queue for retry.
func (s *Store) RequeueJob(ctx context.Context, job *queue.Job) error {
    return s.rdb.LPush(ctx, queueKey(job.Priority), job.ID).Err()
}

// GetStats returns all counters plus live queue depths.
func (s *Store) GetStats(ctx context.Context) (map[string]string, error) {
    stats, err := s.rdb.HGetAll(ctx, keyStats).Result()
    if err != nil {
        return nil, err
    }

    // Add live queue lengths — how many jobs are currently waiting.
    for _, p := range []struct {
        key  string
        name string
    }{
        {keyQueueHigh, "queue_high"},
        {keyQueueMedium, "queue_medium"},
        {keyQueueLow, "queue_low"},
    } {
        l, _ := s.rdb.LLen(ctx, p.key).Result()
        stats[p.name] = fmt.Sprintf("%d", l)
    }

    total, _ := s.rdb.SCard(ctx, keyAllJobs).Result()
    stats["total_jobs"] = fmt.Sprintf("%d", total)

    return stats, nil
}

// ListJobs returns metadata for every job ever created.
func (s *Store) ListJobs(ctx context.Context) ([]*queue.Job, error) {
    ids, err := s.rdb.SMembers(ctx, keyAllJobs).Result()
    if err != nil {
        return nil, err
    }

    jobs := make([]*queue.Job, 0, len(ids))
    for _, id := range ids {
        job, err := s.GetJob(ctx, id)
        if err != nil {
            continue // skip corrupted entries
        }
        jobs = append(jobs, job)
    }
    return jobs, nil
}

// IncrStat and DecrStat update named counters.
func (s *Store) IncrStat(ctx context.Context, key string) {
    s.rdb.HIncrBy(ctx, keyStats, key, 1)
}

func (s *Store) DecrStat(ctx context.Context, key string) {
    s.rdb.HIncrBy(ctx, keyStats, key, -1)
}

// queueKey maps a Priority to its Redis LIST key.
func queueKey(p queue.Priority) string {
    switch p {
    case queue.PriorityHigh:
        return keyQueueHigh
    case queue.PriorityMedium:
        return keyQueueMedium
    default:
        return keyQueueLow
    }
}