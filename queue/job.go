package queue

import "time"

type Priority int

//this is basically an enum
const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 5
	PriorityHigh   Priority = 10
)

//ParsePriority converts a string from CLI/API into our priority type

func ParsePriority(s string) Priority{
	switch(s) {
	case "high":
		return PriorityHigh
	case "medium":
		return PriorityMedium
	default:
		return PriorityLow
	}
}

// String lets Priority print itself as a human-readable word.
// Go calls this automatically when you use fmt.Println(priority).
func (p Priority) String() string {
    switch p {
    case PriorityHigh:
        return "high"
    case PriorityMedium:
        return "medium"
    default:
        return "low"
    }
}

//Using a named string type (not plain string) prevents typos and for compile-time safety
type Status string

const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusDone     Status = "done"
	StatusFailed   Status = "failed"
	StatusRetrying Status = "retrying"
)

// Job is the core data structure of the entire systim.
type Job struct {
    ID          string     `json:"id"`
    Name        string     `json:"name"`
    Type        string     `json:"type"`
    Priority    Priority   `json:"priority"`
    Status      Status     `json:"status"`
    Retries     int        `json:"retries"`
    MaxRetries  int        `json:"max_retries"`
    Error       string     `json:"error,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    StartedAt   *time.Time `json:"started_at,omitempty"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
}