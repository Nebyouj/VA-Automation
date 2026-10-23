package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
	"github.com/nebyoudamtew/go-job-queue/interval/repository"
)

type PostgresJobRepository struct {
	db *pgxpool.Pool
}

func NewJobRepository(db *pgxpool.Pool) *PostgresJobRepository {
	return &PostgresJobRepository{
		db: db,
	}
}

func (r *PostgresJobRepository) Create(
	ctx context.Context,
	j job.Job,
) (job.Job, error) {
	query := `
		INSERT INTO jobs (
			type,
			status,
			priority,
			payload,
			attempts,
			max_attempts,
			scheduled_at,
			repeat_every_seconds,
			cron_expression,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	payload, err := json.Marshal(j.Payload)
	if err != nil {
		return job.Job{}, fmt.Errorf("marshal job payload: %w", err)
	}
	err = r.db.QueryRow(
		ctx,
		query,
		j.Type,
		j.Status,
		j.Priority,
		payload,
		j.Attempts,
		j.MaxAttempts,
		j.ScheduledAt,
		j.RepeatEverySeconds,
		j.CronExpression,
		j.CreatedAt,
	).Scan(&j.ID)

	if err != nil {
		return job.Job{}, fmt.Errorf("create job: %w", err)
	}

	return j, nil
}

func (r *PostgresJobRepository) GetByID(
	ctx context.Context,
	id int,
) (job.Job, error) {
	query := `
		SELECT
			id,
			type,
			status,
			priority,
			payload,
			attempts,
			max_attempts,
			scheduled_at,
			repeat_every_seconds,
			cron_expression,
			created_at,
			started_at,
			ended_at,
			result
			FROM jobs
			WHERE id = $1
			`

	var j job.Job
	var payload []byte
	var result []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&j.ID,
		&j.Type,
		&j.Status,
		&j.Priority,
		&payload,
		&j.Attempts,
		&j.MaxAttempts,
		&j.ScheduledAt,
		&j.RepeatEverySeconds,
		&j.CronExpression,
		&j.CreatedAt,
		&j.StartedAt,
		&j.EndedAt,
		&result,
	)

	if err := json.Unmarshal(payload, &j.Payload); err != nil {
		return job.Job{}, fmt.Errorf("unmarshal job payload: %w", err)
	}

	if len(result) > 0 {
		if err := json.Unmarshal(result, &j.Result); err != nil {
			return job.Job{}, fmt.Errorf("unmarshal job result: %w", err)
		}
	}

	if err != nil {
		return job.Job{}, fmt.Errorf("get job: %w", err)
	}

	return j, nil
}

func (r *PostgresJobRepository) GetAll(
	ctx context.Context,
	status string,
	jobType string,
	page int,
	limit int,
) ([]job.Job, error) {

	offset := (page - 1) * limit

	query := `
		SELECT
			id,
			type,
			status,
			priority,
			payload,
			attempts,
			max_attempts,
			scheduled_at,
			repeat_every_seconds,
			cron_expression,
			created_at,
			started_at,
			ended_at,
			result
		FROM jobs
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR type = $2)
		ORDER BY id DESC
		LIMIT $3
		OFFSET $4
	`

	rows, err := r.db.Query(ctx, query, status, jobType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get jobs: %w", err)
	}
	defer rows.Close()

	var jobs []job.Job

	for rows.Next() {
		var j job.Job
		var payload []byte
		var result []byte

		err := rows.Scan(
			&j.ID,
			&j.Type,
			&j.Status,
			&j.Priority,
			&payload,
			&j.Attempts,
			&j.MaxAttempts,
			&j.ScheduledAt,
			&j.RepeatEverySeconds,
			&j.CronExpression,
			&j.CreatedAt,
			&j.StartedAt,
			&j.EndedAt,
			&result,
		)

		if err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}

		if err := json.Unmarshal(payload, &j.Payload); err != nil {
			return nil, fmt.Errorf("unmarshal job payload: %w", err)
		}

		if len(result) > 0 {
			if err := json.Unmarshal(result, &j.Result); err != nil {
				return nil, fmt.Errorf("unmarshal job result: %w", err)
			}
		}

		jobs = append(jobs, j)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs: %w", err)
	}

	return jobs, nil
}

func (r *PostgresJobRepository) Delete(
	ctx context.Context,
	id int,
) error {
	result, err := r.db.Exec(
		ctx,
		`DELETE FROM jobs WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete job %d: %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job %d not found", id)
	}

	return nil
}

func (r *PostgresJobRepository) IncrementAttempt(
	ctx context.Context,
	id int,
) (job.Job, error) {
	query := `
		UPDATE jobs
		SET attempts = attempts + 1
		WHERE id = $1
		RETURNING
			id,
			type,
			status,
			priority,
			payload,
			attempts,
			max_attempts,
			scheduled_at,
			repeat_every_seconds,
			cron_expression,
			created_at,
			started_at,
			ended_at,
			result
	`

	var j job.Job
	var payload []byte
	var result []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&j.ID,
		&j.Type,
		&j.Status,
		&j.Priority,
		&payload,
		&j.Attempts,
		&j.MaxAttempts,
		&j.ScheduledAt,
		&j.RepeatEverySeconds,
		&j.CronExpression,
		&j.CreatedAt,
		&j.StartedAt,
		&j.EndedAt,
		&result,
	)

	if err != nil {
		return job.Job{}, fmt.Errorf("increment attempt: %w", err)
	}

	if err := json.Unmarshal(payload, &j.Payload); err != nil {
		return job.Job{}, fmt.Errorf("unmarshal job payload: %w", err)
	}

	if len(result) > 0 {
		if err := json.Unmarshal(result, &j.Result); err != nil {
			return job.Job{}, fmt.Errorf("unmarshal job result: %w", err)
		}
	}

	return j, nil
}

func (r *PostgresJobRepository) MarkRunning(
	ctx context.Context,
	id int,
) error {
	query := `
		UPDATE jobs
		SET
			status = $1,
			started_at = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(
		ctx,
		query,
		job.StatusRunning,
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("mark job running: %w", err)
	}

	return nil
}

func (r *PostgresJobRepository) MarkCompleted(
	ctx context.Context,
	id int,
) error {
	query := `
		UPDATE jobs
		SET
			status = $1,
			ended_at = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(
		ctx,
		query,
		job.StatusCompleted,
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("mark job completed: %w", err)
	}

	return nil
}

func (r *PostgresJobRepository) MarkFailed(
	ctx context.Context,
	id int,
) error {
	query := `
		UPDATE jobs
		SET
			status = $1,
			ended_at = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(
		ctx,
		query,
		job.StatusFailed,
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("mark job failed: %w", err)
	}

	return nil
}

func (r *PostgresJobRepository) MarkCancelled(
	ctx context.Context,
	id int,
) error {
	query := `
		UPDATE jobs
		SET
			status = $1,
			ended_at = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(
		ctx,
		query,
		job.StatusCancelled,
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("mark job cancelled: %w", err)
	}

	return nil
}

func (r *PostgresJobRepository) GetRecoverableJobs(
	ctx context.Context,
) ([]job.Job, error) {
	query := `
		SELECT
			id,
			type,
			status,
			priority,
			payload,
			attempts,
			max_attempts,
			scheduled_at,
			repeat_every_seconds,
			cron_expression,
			created_at,
			started_at,
			ended_at
		FROM jobs
		WHERE status IN ($1, $2, $3)
		  AND id NOT IN (SELECT job_id FROM workflow_jobs)
		ORDER BY id
	`

	rows, err := r.db.Query(
		ctx,
		query,
		job.StatusPending,
		"pending",
		job.StatusRunning,
	)
	if err != nil {
		return nil, fmt.Errorf("get recoverable jobs: %w", err)
	}
	defer rows.Close()

	var jobs []job.Job

	for rows.Next() {
		var j job.Job
		var payload []byte

		err := rows.Scan(
			&j.ID,
			&j.Type,
			&j.Status,
			&j.Priority,
			&payload,
			&j.Attempts,
			&j.MaxAttempts,
			&j.ScheduledAt,
			&j.RepeatEverySeconds,
			&j.CronExpression,
			&j.CreatedAt,
			&j.StartedAt,
			&j.EndedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan recoverable job: %w", err)
		}

		if err := json.Unmarshal(payload, &j.Payload); err != nil {
			return nil, fmt.Errorf("unmarshal job payload: %w", err)
		}

		jobs = append(jobs, j)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recoverable jobs: %w", err)
	}

	return jobs, nil
}

func (r *PostgresJobRepository) MarkPending(
	ctx context.Context,
	id int,
) error {
	query := `
		UPDATE jobs
		SET
			status = $1,
			started_at = NULL,
			ended_at = NULL,
			attempts = 0
		WHERE id = $2
	`

	_, err := r.db.Exec(
		ctx,
		query,
		job.StatusPending,
		id,
	)

	if err != nil {
		return fmt.Errorf("mark job pending: %w", err)
	}

	return nil
}

func (r *PostgresJobRepository) GetStats(ctx context.Context) (repository.JobStats, error) {
	var stats repository.JobStats

	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending'),
			COUNT(*) FILTER (WHERE status = 'running'),
			COUNT(*) FILTER (WHERE status = 'completed'),
			COUNT(*) FILTER (WHERE status = 'failed'),
			COUNT(*) FILTER (WHERE status = 'cancelled')
		FROM jobs
	`

	err := r.db.QueryRow(ctx, query).Scan(
		&stats.Pending,
		&stats.Running,
		&stats.Completed,
		&stats.Failed,
		&stats.Cancelled,
	)

	if err != nil {
		return repository.JobStats{}, fmt.Errorf("get job stats: %w", err)
	}

	return stats, nil
}

func (r *PostgresJobRepository) Reschedule(
	ctx context.Context,
	id int,
	scheduledAt time.Time,
) error {
	query := `
		UPDATE jobs
		SET status = $1,
			scheduled_at = $2,
			attempts = 0,
			started_at = NULL,
			ended_at = NULL
		where id = $3
	`

	result, err := r.db.Exec(
		ctx,
		query,
		job.StatusPending,
		scheduledAt,
		id,
	)

	if err != nil {
		return fmt.Errorf("reschedule job %d: %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("reschedule job %d: job not found", id)
	}

	return nil
}

func (r *PostgresJobRepository) SaveResult(
	ctx context.Context,
	id int,
	result map[string]any,
) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal job result: %w", err)
	}

	query := `
        UPDATE jobs
        SET result = $1
        WHERE id = $2
    `

	commandTag, err := r.db.Exec(ctx, query, resultJSON, id)
	if err != nil {
		return fmt.Errorf("save job result: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("save job result: job %d not found", id)
	}

	return nil
}

func (r *PostgresJobRepository) UpdatePayload(
	ctx context.Context,
	id int,
	payload map[string]any,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal job payload: %w", err)
	}

	query := `UPDATE jobs SET payload = $1 WHERE id = $2`
	commandTag, err := r.db.Exec(ctx, query, payloadJSON, id)
	if err != nil {
		return fmt.Errorf("update job payload: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("update job payload: job %d not found", id)
	}

	return nil
}
