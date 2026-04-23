# Stage 2 - Job Struct

A job is just a unit of work. It needs to carry enough information so that:

- The producer knows what to create
- Redis knows where to put it (which priority queue)
- The worker knows what to execute
- Anyone checking status knows what happened

Think of it like a ticket at a restaurant — it has an ID, what was ordered, who ordered it, what state it's in (waiting / cooking / done).

## What are the backtick tags like json:"id"?
These are struct tags — metadata attached to fields. The encoding/json package reads them to know what name to use when converting this struct to/from JSON. Without tags, Go would use the field name as-is (ID, Name) which looks ugly in JSON. With tags you get clean lowercase snake_case output.

submit
  │
  ▼
pending  ──► running ──► done
                │
                ▼
            retrying ──► pending (after backoff delay)
                │
                ▼ (retries >= maxRetries)
              failed