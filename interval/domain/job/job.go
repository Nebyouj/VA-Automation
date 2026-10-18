package job

import "time"

type Status string

const (
	StatusPending   Status = "Pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Job struct {
	ID                 int            `json:"id"`
	Type               string         `json:"type"`
	Status             Status         `json:"status"`
	Priority           int            `json:"priority"`
	Payload            map[string]any `json:"payload,omitempty"`
	Attempts           int            `json:"attempts"`
	MaxAttempts        int            `json:"max_attempts"`
	ScheduledAt        *time.Time     `json:"scheduled_at"`
	RepeatEverySeconds int            `json:"repeat_every_seconds,omitempty"`
	CronExpression     string         `json:"cron_expression,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	StartedAt          *time.Time     `json:"started_at"`
	EndedAt            *time.Time     `json:"ended_at"`
	Result             map[string]any `json:"result,omitempty"`
}
