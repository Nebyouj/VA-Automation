package jobs

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
	"github.com/nebyoudamtew/go-job-queue/interval/infrastructure/queue"
	"github.com/nebyoudamtew/go-job-queue/interval/repository"
	"github.com/robfig/cron/v3"
)

type JobEventListener interface {
	OnJobCompleted(ctx context.Context, jobID int) error
	OnJobFailed(ctx context.Context, jobID int) error
	OnJobCancelled(ctx context.Context, jobID int) error
}

type Service struct {
	queue       *queue.Queue
	repository  repository.JobRepository
	mu          sync.RWMutex
	cancelFuncs map[int]context.CancelFunc
	listener    JobEventListener
}

func NewService(q *queue.Queue, repo repository.JobRepository) *Service {
	return &Service{
		queue:       q,
		repository:  repo,
		cancelFuncs: make(map[int]context.CancelFunc),
	}
}

func (m *Service) SetEventListener(l JobEventListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listener = l
}

// createJobRecord applies the shared scheduling, defaulting, validation,
// and persistence logic. It does not enqueue the job.
func (m *Service) createJobRecord(
	ctx context.Context,
	jobType string,
	priority int,
	maxAttempts int,
	scheduledAt *time.Time,
	repeatEverySeconds int,
	cronExpression string,
	payload map[string]any,
) (job.Job, error) {
	if strings.TrimSpace(cronExpression) != "" && scheduledAt == nil {
		schedule, err := cron.ParseStandard(cronExpression)
		if err != nil {
			return job.Job{}, fmt.Errorf("invalid cron expression: %w", err)
		}

		nextRun := schedule.Next(time.Now())
		scheduledAt = &nextRun
	}

	if maxAttempts == 0 {
		maxAttempts = 3
	}

	j := job.Job{
		Type:               jobType,
		Status:             job.StatusPending,
		Priority:           priority,
		MaxAttempts:        maxAttempts,
		ScheduledAt:        scheduledAt,
		RepeatEverySeconds: repeatEverySeconds,
		CronExpression:     cronExpression,
		Payload:            payload,
		Attempts:           0,
		CreatedAt:          time.Now(),
	}

	if err := job.Validate(j); err != nil {
		return job.Job{}, err
	}

	return m.repository.Create(ctx, j)
}

// CreatePersistedJob creates and stores a job without enqueueing it.
// Workflow orchestration can enqueue it later, when its dependencies are ready.
func (m *Service) CreatePersistedJob(
	ctx context.Context,
	jobType string,
	priority int,
	maxAttempts int,
	scheduledAt *time.Time,
	repeatEverySeconds int,
	cronExpression string,
	payload map[string]any,
) (job.Job, error) {
	return m.createJobRecord(
		ctx,
		jobType,
		priority,
		maxAttempts,
		scheduledAt,
		repeatEverySeconds,
		cronExpression,
		payload,
	)
}

// DeletePersistedJob removes a job that has not been enqueued.
func (m *Service) DeletePersistedJob(
	ctx context.Context,
	id int,
) error {
	return m.repository.Delete(ctx, id)
}

// EnqueuePersistedJob enqueues an existing persisted job into the execution queue.
// It returns true if the job was enqueued, or false if it was already enqueued or in a terminal state.
func (m *Service) EnqueuePersistedJob(ctx context.Context, id int) (bool, error) {
	m.mu.Lock()
	if _, exists := m.cancelFuncs[id]; exists {
		m.mu.Unlock()
		return false, nil
	}

	j, err := m.repository.GetByID(ctx, id)
	if err != nil {
		m.mu.Unlock()
		return false, fmt.Errorf("get job %d: %w", id, err)
	}

	if strings.EqualFold(string(j.Status), "completed") ||
		strings.EqualFold(string(j.Status), "failed") ||
		strings.EqualFold(string(j.Status), "cancelled") {
		m.mu.Unlock()
		return false, nil
	}

	jobCtx, cancel := context.WithCancel(context.Background())
	m.cancelFuncs[j.ID] = cancel
	m.mu.Unlock()

	err = m.queue.Enqueue(
		jobCtx,
		queue.QueueJob{
			Job:     j,
			Context: jobCtx,
		},
	)
	if err != nil {
		cancel()
		m.mu.Lock()
		delete(m.cancelFuncs, j.ID)
		m.mu.Unlock()
		return false, fmt.Errorf("enqueue job %d: %w", j.ID, err)
	}

	return true, nil
}

func (m *Service) MarkPending(ctx context.Context, id int) error {
	m.mu.Lock()
	delete(m.cancelFuncs, id)
	m.mu.Unlock()
	return m.repository.MarkPending(ctx, id)
}

// CreateJob creates, stores, and immediately enqueues a standalone job.
func (m *Service) CreateJob(
	ctx context.Context,
	jobType string,
	priority int,
	maxAttempts int,
	scheduledAt *time.Time,
	repeatEverySeconds int,
	cronExpression string,
	payload map[string]any,
) (job.Job, error) {
	createdJob, err := m.createJobRecord(
		ctx,
		jobType,
		priority,
		maxAttempts,
		scheduledAt,
		repeatEverySeconds,
		cronExpression,
		payload,
	)
	if err != nil {
		return job.Job{}, err
	}

	jobCtx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.cancelFuncs[createdJob.ID] = cancel
	m.mu.Unlock()

	err = m.queue.Enqueue(
		jobCtx,
		queue.QueueJob{
			Job:     createdJob,
			Context: jobCtx,
		},
	)
	if err != nil {
		cancel()
		_ = m.repository.MarkFailed(context.Background(), createdJob.ID)

		m.mu.Lock()
		delete(m.cancelFuncs, createdJob.ID)
		m.mu.Unlock()

		return job.Job{}, err
	}

	return createdJob, nil
}

// func (m *Service) CreateJob(
// 	ctx context.Context,
// 	jobType string,
// 	priority int,
// 	maxAttempts int,
// 	scheduledAt *time.Time,
// 	repeatEverySeconds int,
// 	cronExpression string,
// 	payload map[string]any,
// ) (job.Job, error) {

// 	if strings.TrimSpace(cronExpression) != "" && scheduledAt == nil {
// 		schedule, err := cron.ParseStandard(cronExpression)
// 		if err != nil {
// 			return job.Job{}, fmt.Errorf(
// 				"inalid cron expression: %w",
// 				err,
// 			)
// 		}

// 		nextRun := schedule.Next(time.Now())
// 		scheduledAt = &nextRun
// 	}

// 	if maxAttempts == 0 {
// 		maxAttempts = 3
// 	}

// 	j := job.Job{
// 		Type:               jobType,
// 		Status:             job.StatusPending,
// 		Priority:           priority,
// 		MaxAttempts:        maxAttempts,
// 		ScheduledAt:        scheduledAt,
// 		RepeatEverySeconds: repeatEverySeconds,
// 		CronExpression:     cronExpression,
// 		Payload:            payload,
// 		Attempts:           0,
// 		CreatedAt:          time.Now(),
// 	}

// 	if err := job.Validate(j); err != nil {
// 		return job.Job{}, err
// 	}

// 	jobCtx, cancel := context.WithCancel(context.Background())

// 	createdJob, err := m.repository.Create(ctx, j)
// 	if err != nil {
// 		cancel()
// 		return job.Job{}, err
// 	}
// 	m.mu.Lock()
// 	m.cancelFuncs[createdJob.ID] = cancel
// 	m.mu.Unlock()

// 	err = m.queue.Enqueue(jobCtx,
// 		queue.QueueJob{
// 			Job:     createdJob,
// 			Context: jobCtx,
// 		},
// 	)

// 	if err != nil {
// 		cancel()
// 		_ = m.repository.MarkFailed(context.Background(), createdJob.ID)

// 		m.mu.Lock()
// 		delete(m.cancelFuncs, createdJob.ID)
// 		m.mu.Unlock()

// 		return job.Job{}, err
// 	}

// 	// m.mu.Lock()
// 	// m.jobs[j.ID] = j
// 	// m.mu.Unlock()

// 	return createdJob, nil
// }

func (m *Service) GetJob(ctx context.Context, id int) (job.Job, error) {
	return m.repository.GetByID(ctx, id)
}

func (m *Service) GetJobs(ctx context.Context, status string, jobType string, page int, limit int) ([]job.Job, error) {
	return m.repository.GetAll(ctx, status, jobType, page, limit)
}

func (m *Service) GetStats(ctx context.Context) (repository.JobStats, error) {
	return m.repository.GetStats(ctx)
}

func (m *Service) MarkRunning(id int) error {
	return m.repository.MarkRunning(
		context.Background(),
		id,
	)
}

func (m *Service) MarkCompleted(id int) error {
	err := m.repository.MarkCompleted(
		context.Background(),
		id,
	)
	if err != nil {
		return err
	}

	m.mu.Lock()
	delete(m.cancelFuncs, id)
	listener := m.listener
	m.mu.Unlock()

	if listener != nil {
		if err := listener.OnJobCompleted(context.Background(), id); err != nil {
			fmt.Printf("warning: onJobCompleted failed for job %d: %v\n", id, err)
		}
	}

	return nil
}

func (m *Service) MarkFailed(id int) error {
	err := m.repository.MarkFailed(
		context.Background(),
		id,
	)
	if err != nil {
		return err
	}

	m.mu.Lock()
	delete(m.cancelFuncs, id)
	listener := m.listener
	m.mu.Unlock()

	if listener != nil {
		if err := listener.OnJobFailed(context.Background(), id); err != nil {
			fmt.Printf("warning: onJobFailed failed for job %d: %v\n", id, err)
		}
	}

	return nil
}

func (m *Service) IncrementAttempt(
	id int,
) (job.Job, error) {
	return m.repository.IncrementAttempt(
		context.Background(),
		id,
	)
}

func (m *Service) CancelJob(
	ctx context.Context,
	id int,
) error {
	j, err := m.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if strings.EqualFold(string(j.Status), "completed") {
		return fmt.Errorf("job already completed")
	}

	if strings.EqualFold(string(j.Status), "failed") {
		return fmt.Errorf("job already failed")
	}

	if strings.EqualFold(string(j.Status), "cancelled") {
		return fmt.Errorf("job already cancelled")
	}

	m.mu.Lock()
	cancel, exists := m.cancelFuncs[id]
	if exists {
		cancel()
	} else {
		m.cancelFuncs[id] = func() {}
	}
	m.mu.Unlock()

	// Persist the cancellation.
	err = m.repository.MarkCancelled(
		context.Background(),
		id,
	)

	if err != nil {
		return fmt.Errorf("cancel job: %w", err)
	}

	m.mu.RLock()
	listener := m.listener
	m.mu.RUnlock()

	if listener != nil {
		if err := listener.OnJobCancelled(ctx, id); err != nil {
			fmt.Printf("warning: onJobCancelled failed for job %d: %v\n", id, err)
		}
	}

	return nil
}

func (m *Service) RecoverJobs(
	ctx context.Context,
) error {
	jobs, err := m.repository.GetRecoverableJobs(ctx)
	if err != nil {
		return err
	}

	for _, j := range jobs {
		m.mu.Lock()
		if _, exists := m.cancelFuncs[j.ID]; exists {
			m.mu.Unlock()
			continue
		}

		jobCtx, cancel := context.WithCancel(
			context.Background(),
		)
		m.cancelFuncs[j.ID] = cancel
		m.mu.Unlock()

		// A job that was running when the server stopped
		// needs to be treated as pending again.
		if strings.EqualFold(string(j.Status), "running") {
			err := m.repository.MarkPending(
				context.Background(),
				j.ID,
			)

			if err != nil {
				cancel()
				m.mu.Lock()
				delete(m.cancelFuncs, j.ID)
				m.mu.Unlock()
				return fmt.Errorf(
					"reset recovered job %d: %w",
					j.ID,
					err,
				)
			}

			j.Status = job.StatusPending
		}

		err := m.queue.Enqueue(
			ctx,
			queue.QueueJob{
				Job:     j,
				Context: jobCtx,
			},
		)

		if err != nil {
			cancel()

			m.mu.Lock()
			delete(m.cancelFuncs, j.ID)
			m.mu.Unlock()

			return fmt.Errorf(
				"recover job %d: %w",
				j.ID,
				err,
			)
		}

		fmt.Printf(
			"Recovered job %d (%s)\n",
			j.ID,
			j.Type,
		)
	}

	m.mu.RLock()
	listener := m.listener
	m.mu.RUnlock()

	if listener != nil {
		if recoveryListener, ok := listener.(interface {
			RecoverWorkflows(context.Context) error
		}); ok {
			if err := recoveryListener.RecoverWorkflows(ctx); err != nil {
				return fmt.Errorf("recover workflows: %w", err)
			}
		}
	}

	return nil
}

func (m *Service) RescheduleJob(
	id int,
	scheduleAt time.Time,
) error {
	return m.repository.Reschedule(
		context.Background(),
		id,
		scheduleAt,
	)
}

func (m *Service) SaveResult(id int, result map[string]any) error {
	return m.repository.SaveResult(
		context.Background(),
		id,
		result,
	)
}

func (m *Service) UpdatePayload(ctx context.Context, id int, payload map[string]any) error {
	return m.repository.UpdatePayload(ctx, id, payload)
}
