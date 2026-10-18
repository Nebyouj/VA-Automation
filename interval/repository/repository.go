package repository

import (
	"context"
	"time"

	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
)

type JobStats struct {
	Pending   int `json:"pending"`
	Running   int `json:"running"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
}

type JobRepository interface {
	Create(ctx context.Context, j job.Job) (job.Job, error)
	GetByID(ctx context.Context, id int) (job.Job, error)
	GetAll(ctx context.Context, status string, jobType string, page int, limit int) ([]job.Job, error)
	IncrementAttempt(ctx context.Context, id int) (job.Job, error)
	MarkRunning(ctx context.Context, id int) error
	MarkCompleted(ctx context.Context, id int) error
	MarkFailed(ctx context.Context, id int) error
	MarkCancelled(ctx context.Context, id int) error
	GetRecoverableJobs(ctx context.Context) ([]job.Job, error)
	MarkPending(ctx context.Context, id int) error
	GetStats(ctx context.Context) (JobStats, error)
	Reschedule(ctx context.Context, id int, scheduledAt time.Time) error
	SaveResult(ctx context.Context, id int, result map[string]any) error
}
