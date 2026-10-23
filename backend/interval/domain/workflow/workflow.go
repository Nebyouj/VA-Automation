package workflow

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Workflow struct {
	ID             int           `json:"id"`
	Name           string        `json:"name"`
	Status         Status        `json:"status"`
	CronExpression string        `json:"cron_expression,omitempty"`
	NextRunAt      *time.Time    `json:"next_run_at,omitempty"`
	Jobs           []WorkflowJob `json:"jobs,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	StartedAt      *time.Time    `json:"started_at,omitempty"`
	EndedAt        *time.Time    `json:"ended_at,omitempty"`
}

type WorkflowJob struct {
	JobID           int    `json:"job_id"`
	Position        int    `json:"position"`
	Key             string `json:"key,omitempty"`
	DependsOnJobIDs []int  `json:"depends_on_job_ids,omitempty"`
}
