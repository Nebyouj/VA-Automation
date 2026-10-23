package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nebyoudamtew/go-job-queue/interval/domain/workflow"
)

type PostgresWorkflowRepository struct {
	db *pgxpool.Pool
}

func NewWorkflowRepository(db *pgxpool.Pool) *PostgresWorkflowRepository {
	return &PostgresWorkflowRepository{db: db}
}

func (r *PostgresWorkflowRepository) InitSchema(ctx context.Context) error {
	queries := []string{
		`ALTER TABLE workflow_jobs ADD COLUMN IF NOT EXISTS job_key VARCHAR(100);`,
		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS cron_expression VARCHAR(100);`,
		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS next_run_at TIMESTAMPTZ;`,
	}
	for _, q := range queries {
		if _, err := r.db.Exec(ctx, q); err != nil {
			return fmt.Errorf("init schema (%s): %w", q, err)
		}
	}
	return nil
}

func (r *PostgresWorkflowRepository) Create(
	ctx context.Context,
	w workflow.Workflow,
) (workflow.Workflow, error) {
	if w.Name == "" {
		return workflow.Workflow{}, fmt.Errorf("workflow name is required")
	}

	if w.Status == "" {
		w.Status = workflow.StatusPending
	}

	if w.Status != workflow.StatusPending {
		return workflow.Workflow{}, fmt.Errorf(
			"new workflow must have pending status",
		)
	}

	jobIDs := make(map[int]bool, len(w.Jobs))
	for _, item := range w.Jobs {
		if item.JobID <= 0 {
			return workflow.Workflow{}, fmt.Errorf(
				"workflow job ID must be positive",
			)
		}
		if item.Position < 0 {
			return workflow.Workflow{}, fmt.Errorf(
				"workflow job position cannot be negative",
			)
		}
		if jobIDs[item.JobID] {
			return workflow.Workflow{}, fmt.Errorf(
				"duplicate job ID %d in workflow",
				item.JobID,
			)
		}
		jobIDs[item.JobID] = true
	}

	for _, item := range w.Jobs {
		for _, dependencyID := range item.DependsOnJobIDs {
			if dependencyID == item.JobID {
				return workflow.Workflow{}, fmt.Errorf(
					"job %d cannot depend on itself",
					item.JobID,
				)
			}
			if !jobIDs[dependencyID] {
				return workflow.Workflow{}, fmt.Errorf(
					"dependency job %d is not part of the workflow",
					dependencyID,
				)
			}
		}
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return workflow.Workflow{}, fmt.Errorf(
			"begin workflow transaction: %w", err,
		)
	}
	defer tx.Rollback(ctx)

	var cronExpr *string
	if strings.TrimSpace(w.CronExpression) != "" {
		trimmed := strings.TrimSpace(w.CronExpression)
		cronExpr = &trimmed
	}

	err = tx.QueryRow(
		ctx,
		`INSERT INTO workflows (name, status, cron_expression, next_run_at)
         VALUES ($1, $2, $3, $4)
         RETURNING id, status, COALESCE(cron_expression, ''), next_run_at, created_at, started_at, ended_at`,
		w.Name,
		w.Status,
		cronExpr,
		w.NextRunAt,
	).Scan(
		&w.ID,
		&w.Status,
		&w.CronExpression,
		&w.NextRunAt,
		&w.CreatedAt,
		&w.StartedAt,
		&w.EndedAt,
	)
	if err != nil {
		return workflow.Workflow{}, fmt.Errorf(
			"create workflow: %w", err,
		)
	}

	for _, item := range w.Jobs {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO workflow_jobs (workflow_id, job_id, position, job_key)
             VALUES ($1, $2, $3, $4)`,
			w.ID,
			item.JobID,
			item.Position,
			item.Key,
		)
		if err != nil {
			return workflow.Workflow{}, fmt.Errorf(
				"add job %d to workflow: %w", item.JobID, err,
			)
		}
	}

	for _, item := range w.Jobs {
		for _, dependencyID := range item.DependsOnJobIDs {
			_, err = tx.Exec(
				ctx,
				`INSERT INTO workflow_dependencies
                    (workflow_id, job_id, depends_on_job_id)
                 VALUES ($1, $2, $3)`,
				w.ID,
				item.JobID,
				dependencyID,
			)
			if err != nil {
				return workflow.Workflow{}, fmt.Errorf(
					"add dependency from job %d to job %d: %w",
					item.JobID,
					dependencyID,
					err,
				)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return workflow.Workflow{}, fmt.Errorf(
			"commit workflow transaction: %w", err,
		)
	}

	return w, nil
}

func (r *PostgresWorkflowRepository) GetByID(
	ctx context.Context,
	id int,
) (workflow.Workflow, error) {
	var w workflow.Workflow

	err := r.db.QueryRow(
		ctx,
		`SELECT id, name, status, COALESCE(cron_expression, ''), next_run_at, created_at, started_at, ended_at
         FROM workflows
         WHERE id = $1`,
		id,
	).Scan(
		&w.ID,
		&w.Name,
		&w.Status,
		&w.CronExpression,
		&w.NextRunAt,
		&w.CreatedAt,
		&w.StartedAt,
		&w.EndedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return workflow.Workflow{}, fmt.Errorf(
				"workflow %d not found", id,
			)
		}
		return workflow.Workflow{}, fmt.Errorf(
			"get workflow %d: %w", id, err,
		)
	}

	rows, err := r.db.Query(
		ctx,
		`SELECT job_id, position, COALESCE(job_key, '')
         FROM workflow_jobs
         WHERE workflow_id = $1
         ORDER BY position, job_id`,
		id,
	)
	if err != nil {
		return workflow.Workflow{}, fmt.Errorf(
			"get workflow jobs: %w", err,
		)
	}

	for rows.Next() {
		var item workflow.WorkflowJob
		if err := rows.Scan(&item.JobID, &item.Position, &item.Key); err != nil {
			rows.Close()
			return workflow.Workflow{}, fmt.Errorf(
				"scan workflow job: %w", err,
			)
		}
		w.Jobs = append(w.Jobs, item)
	}

	err = rows.Err()
	rows.Close()
	if err != nil {
		return workflow.Workflow{}, fmt.Errorf(
			"iterate workflow jobs: %w", err,
		)
	}

	dependencyRows, err := r.db.Query(
		ctx,
		`SELECT job_id, depends_on_job_id
         FROM workflow_dependencies
         WHERE workflow_id = $1`,
		id,
	)
	if err != nil {
		return workflow.Workflow{}, fmt.Errorf(
			"get workflow dependencies: %w", err,
		)
	}

	jobIndex := make(map[int]int, len(w.Jobs))
	for i := range w.Jobs {
		jobIndex[w.Jobs[i].JobID] = i
	}

	for dependencyRows.Next() {
		var jobID, dependencyID int
		if err := dependencyRows.Scan(&jobID, &dependencyID); err != nil {
			dependencyRows.Close()
			return workflow.Workflow{}, fmt.Errorf(
				"scan workflow dependency: %w", err,
			)
		}

		if index, exists := jobIndex[jobID]; exists {
			w.Jobs[index].DependsOnJobIDs = append(
				w.Jobs[index].DependsOnJobIDs,
				dependencyID,
			)
		}
	}

	err = dependencyRows.Err()
	dependencyRows.Close()
	if err != nil {
		return workflow.Workflow{}, fmt.Errorf(
			"iterate workflow dependencies: %w", err,
		)
	}

	return w, nil
}

func (r *PostgresWorkflowRepository) GetAll(
	ctx context.Context,
	status string,
	page int,
	limit int,
) ([]workflow.Workflow, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	rows, err := r.db.Query(
		ctx,
		`SELECT id
         FROM workflows
         WHERE ($1 = '' OR status = $1)
         ORDER BY created_at DESC, id DESC
         LIMIT $2 OFFSET $3`,
		status,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan workflow ID: %w", err)
		}
		ids = append(ids, id)
	}

	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, fmt.Errorf("iterate workflow IDs: %w", err)
	}

	workflows := make([]workflow.Workflow, 0, len(ids))
	for _, id := range ids {
		item, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, item)
	}

	return workflows, nil
}

func (r *PostgresWorkflowRepository) UpdateStatus(
	ctx context.Context,
	id int,
	status workflow.Status,
) error {
	switch status {
	case workflow.StatusPending,
		workflow.StatusRunning,
		workflow.StatusCompleted,
		workflow.StatusFailed,
		workflow.StatusCancelled:
	default:
		return fmt.Errorf("invalid workflow status %q", status)
	}

	result, err := r.db.Exec(
		ctx,
		`UPDATE workflows
         SET status = $2::text,
             started_at = CASE
                 WHEN $2::text = 'running' AND started_at IS NULL
                 THEN NOW()
                 ELSE started_at
             END,
             ended_at = CASE
                 WHEN $2::text IN ('completed', 'failed', 'cancelled')
                 THEN NOW()
                 WHEN $2::text IN ('pending', 'running')
                 THEN NULL
                 ELSE ended_at
             END
         WHERE id = $1`,
		id,
		string(status),
	)
	if err != nil {
		return fmt.Errorf("update workflow status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("workflow %d not found", id)
	}

	return nil
}

func (r *PostgresWorkflowRepository) UpdateNextRunAt(
	ctx context.Context,
	id int,
	nextRunAt *time.Time,
) error {
	result, err := r.db.Exec(
		ctx,
		`UPDATE workflows
         SET next_run_at = $2
         WHERE id = $1`,
		id,
		nextRunAt,
	)
	if err != nil {
		return fmt.Errorf("update workflow next_run_at: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("workflow %d not found", id)
	}

	return nil
}

func (r *PostgresWorkflowRepository) GetByJobID(
	ctx context.Context,
	jobID int,
) (workflow.Workflow, error) {
	var workflowID int
	err := r.db.QueryRow(
		ctx,
		`SELECT workflow_id
         FROM workflow_jobs
         WHERE job_id = $1`,
		jobID,
	).Scan(&workflowID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return workflow.Workflow{}, fmt.Errorf("workflow for job %d not found", jobID)
		}
		return workflow.Workflow{}, fmt.Errorf("get workflow for job %d: %w", jobID, err)
	}

	return r.GetByID(ctx, workflowID)
}

func (r *PostgresWorkflowRepository) GetActiveWorkflows(
	ctx context.Context,
) ([]workflow.Workflow, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT id
         FROM workflows
         WHERE status IN ('pending', 'running')
            OR (cron_expression IS NOT NULL AND TRIM(cron_expression) != '')
         ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list active workflows: %w", err)
	}

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan workflow ID: %w", err)
		}
		ids = append(ids, id)
	}

	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, fmt.Errorf("iterate active workflow IDs: %w", err)
	}

	workflows := make([]workflow.Workflow, 0, len(ids))
	for _, id := range ids {
		item, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, item)
	}

	return workflows, nil
}
