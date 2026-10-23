package repository

import (
	"context"
	"time"

	"github.com/nebyoudamtew/go-job-queue/interval/domain/workflow"
)

type WorkflowRepository interface {
	Create(ctx context.Context, w workflow.Workflow) (workflow.Workflow, error)

	GetByID(ctx context.Context, id int) (workflow.Workflow, error)

	GetAll(
		ctx context.Context,
		status string,
		page int,
		limit int,
	) ([]workflow.Workflow, error)

	UpdateStatus(
		ctx context.Context,
		id int,
		status workflow.Status,
	) error

	UpdateNextRunAt(
		ctx context.Context,
		id int,
		nextRunAt *time.Time,
	) error

	GetByJobID(ctx context.Context, jobID int) (workflow.Workflow, error)

	GetActiveWorkflows(ctx context.Context) ([]workflow.Workflow, error)

	InitSchema(ctx context.Context) error
}
