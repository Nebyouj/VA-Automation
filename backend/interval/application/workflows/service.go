package workflows

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nebyoudamtew/go-job-queue/interval/application/jobs"
	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
	"github.com/nebyoudamtew/go-job-queue/interval/domain/workflow"
	"github.com/nebyoudamtew/go-job-queue/interval/repository"
	"github.com/robfig/cron/v3"
)

// JobDefinition describes a job before it receives a database ID.
// Key is a workflow-local identifier used by DependsOn.
type JobDefinition struct {
	Key                string         `json:"key"`
	Type               string         `json:"type"`
	Priority           int            `json:"priority"`
	MaxAttempts        int            `json:"max_attempts"`
	ScheduledAt        *time.Time     `json:"scheduled_at,omitempty"`
	RepeatEverySeconds int            `json:"repeat_every_seconds,omitempty"`
	CronExpression     string         `json:"cron_expression,omitempty"`
	Payload            map[string]any `json:"payload"`
	DependsOn          []string       `json:"depends_on,omitempty"`
}

// CreateRequest is the input for creating a workflow.
type CreateRequest struct {
	Name           string          `json:"name"`
	CronExpression string          `json:"cron_expression,omitempty"`
	Jobs           []JobDefinition `json:"jobs"`
}

type Service struct {
	jobService *jobs.Service
	repository repository.WorkflowRepository
	mu         sync.Mutex
}

func NewService(
	jobService *jobs.Service,
	repo repository.WorkflowRepository,
) *Service {
	return &Service{
		jobService: jobService,
		repository: repo,
	}
}

// validate checks the complete workflow before any jobs are persisted.
func validate(req CreateRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("workflow name is required")
	}

	if strings.TrimSpace(req.CronExpression) != "" {
		if _, err := cron.ParseStandard(req.CronExpression); err != nil {
			return fmt.Errorf("invalid workflow cron_expression: %w", err)
		}
	}

	if len(req.Jobs) == 0 {
		return fmt.Errorf("workflow must contain at least one job")
	}

	keys := make(map[string]bool, len(req.Jobs))

	for i, definition := range req.Jobs {
		key := strings.TrimSpace(definition.Key)
		if key == "" {
			return fmt.Errorf("job at position %d requires a key", i)
		}
		if keys[key] {
			return fmt.Errorf("duplicate workflow job key %q", key)
		}
		keys[key] = true

		// Recurring jobs need separate workflow semantics. Reject them
		// until recurring workflow execution is explicitly supported.
		if definition.RepeatEverySeconds != 0 ||
			strings.TrimSpace(definition.CronExpression) != "" {
			return fmt.Errorf(
				"recurring job %q is not supported in workflows",
				key,
			)
		}

		if definition.MaxAttempts < 0 {
			return fmt.Errorf(
				"max_attempts cannot be negative for job %q",
				key,
			)
		}

		candidate := job.Job{
			Type:               definition.Type,
			Status:             job.StatusPending,
			Priority:           definition.Priority,
			MaxAttempts:        definition.MaxAttempts,
			ScheduledAt:        definition.ScheduledAt,
			RepeatEverySeconds: definition.RepeatEverySeconds,
			CronExpression:     definition.CronExpression,
			Payload:            definition.Payload,
		}

		if err := job.Validate(candidate); err != nil {
			return fmt.Errorf("invalid job %q: %w", key, err)
		}
	}

	for _, definition := range req.Jobs {
		seen := make(map[string]bool)

		for _, dependency := range definition.DependsOn {
			dependency = strings.TrimSpace(dependency)

			if !keys[dependency] {
				return fmt.Errorf(
					"job %q depends on unknown job key %q",
					definition.Key,
					dependency,
				)
			}

			if dependency == strings.TrimSpace(definition.Key) {
				return fmt.Errorf(
					"job %q cannot depend on itself",
					definition.Key,
				)
			}

			if seen[dependency] {
				return fmt.Errorf(
					"job %q has duplicate dependency %q",
					definition.Key,
					dependency,
				)
			}
			seen[dependency] = true
		}
	}

	// Detect cycles using depth-first search.
	// A cycle would prevent the affected jobs from becoming runnable.
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	dependencies := make(map[string][]string, len(req.Jobs))

	for _, definition := range req.Jobs {
		dependencies[strings.TrimSpace(definition.Key)] = definition.DependsOn
	}

	var visit func(string) error
	visit = func(key string) error {
		if visiting[key] {
			return fmt.Errorf("workflow contains a dependency cycle at job %q", key)
		}
		if visited[key] {
			return nil
		}

		visiting[key] = true

		for _, dependency := range dependencies[key] {
			if err := visit(strings.TrimSpace(dependency)); err != nil {
				return err
			}
		}

		delete(visiting, key)
		visited[key] = true
		return nil
	}

	for key := range keys {
		if err := visit(key); err != nil {
			return err
		}
	}

	return nil
}

// CreateWorkflow validates the workflow, persists its jobs without
// enqueueing them, and then stores their dependency relationships.
func (s *Service) CreateWorkflow(
	ctx context.Context,
	req CreateRequest,
) (workflow.Workflow, error) {
	if err := validate(req); err != nil {
		return workflow.Workflow{}, err
	}

	// Convert workflow-local keys to database job IDs.
	createdJobIDs := make([]int, 0, len(req.Jobs))
	jobIDs := make(map[string]int, len(req.Jobs))
	workflowJobs := make([]workflow.WorkflowJob, 0, len(req.Jobs))

	cleanupJobs := func() {
		for _, id := range createdJobIDs {
			if err := s.jobService.DeletePersistedJob(ctx, id); err != nil {
				fmt.Printf(
					"warning: failed to clean up job %d: %v\n",
					id,
					err,
				)
			}
		}
	}

	for position, definition := range req.Jobs {
		created, err := s.jobService.CreatePersistedJob(
			ctx,
			definition.Type,
			definition.Priority,
			definition.MaxAttempts,
			definition.ScheduledAt,
			definition.RepeatEverySeconds,
			definition.CronExpression,
			definition.Payload,
		)
		if err != nil {
			cleanupJobs()
			return workflow.Workflow{}, fmt.Errorf(
				"create workflow job %q: %w",
				definition.Key,
				err,
			)
		}

		key := strings.TrimSpace(definition.Key)
		jobIDs[key] = created.ID
		createdJobIDs = append(createdJobIDs, created.ID)

		workflowJobs = append(workflowJobs, workflow.WorkflowJob{
			JobID:    created.ID,
			Position: position,
			Key:      key,
		})
	}

	// Resolve dependencies now that every job has a database ID.
	for i, definition := range req.Jobs {
		for _, dependency := range definition.DependsOn {
			dependencyID, exists := jobIDs[strings.TrimSpace(dependency)]
			if !exists {
				return workflow.Workflow{}, fmt.Errorf(
					"resolve dependency %q for job %q",
					dependency,
					definition.Key,
				)
			}

			workflowJobs[i].DependsOnJobIDs = append(
				workflowJobs[i].DependsOnJobIDs,
				dependencyID,
			)
		}
	}

	var nextRun *time.Time
	if strings.TrimSpace(req.CronExpression) != "" {
		if schedule, err := cron.ParseStandard(req.CronExpression); err == nil {
			t := schedule.Next(time.Now())
			nextRun = &t
		}
	}

	createdWorkflow, err := s.repository.Create(ctx, workflow.Workflow{
		Name:           strings.TrimSpace(req.Name),
		Status:         workflow.StatusPending,
		CronExpression: strings.TrimSpace(req.CronExpression),
		NextRunAt:      nextRun,
		Jobs:           workflowJobs,
	})
	if err != nil {
		cleanupJobs()
		return workflow.Workflow{}, fmt.Errorf("persist workflow: %w", err)
	}

	// Orchestrate root jobs (jobs with no dependencies)
	if err := s.startWorkflow(ctx, createdWorkflow); err != nil {
		return createdWorkflow, fmt.Errorf("start workflow: %w", err)
	}

	// Refresh workflow to return the latest status and started_at
	updated, err := s.repository.GetByID(ctx, createdWorkflow.ID)
	if err == nil {
		return updated, nil
	}

	return createdWorkflow, nil
}

func (s *Service) GetWorkflow(
	ctx context.Context,
	id int,
) (workflow.Workflow, error) {
	if id <= 0 {
		return workflow.Workflow{}, fmt.Errorf("workflow ID must be positive")
	}

	return s.repository.GetByID(ctx, id)
}

func (s *Service) GetWorkflows(
	ctx context.Context,
	status string,
	page int,
	limit int,
) ([]workflow.Workflow, error) {
	return s.repository.GetAll(ctx, status, page, limit)
}

func (s *Service) startWorkflow(ctx context.Context, w workflow.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.repository.UpdateStatus(ctx, w.ID, workflow.StatusRunning); err != nil {
		return fmt.Errorf("update workflow %d status to running: %w", w.ID, err)
	}

	for _, wj := range w.Jobs {
		if len(wj.DependsOnJobIDs) == 0 {
			if _, err := s.jobService.EnqueuePersistedJob(ctx, wj.JobID); err != nil {
				return fmt.Errorf("enqueue root job %d: %w", wj.JobID, err)
			}
		}
	}

	return nil
}

func (s *Service) OnJobCompleted(ctx context.Context, jobID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, err := s.repository.GetByJobID(ctx, jobID)
	if err != nil {
		// Not a workflow job, ignore
		return nil
	}

	if w.Status == workflow.StatusCompleted ||
		w.Status == workflow.StatusFailed ||
		w.Status == workflow.StatusCancelled {
		return nil
	}

	jobStatuses := make(map[int]job.Status, len(w.Jobs))
	for _, wj := range w.Jobs {
		j, err := s.jobService.GetJob(ctx, wj.JobID)
		if err != nil {
			return fmt.Errorf("get status for job %d in workflow %d: %w", wj.JobID, w.ID, err)
		}
		jobStatuses[wj.JobID] = j.Status
	}

	// Check for failures or cancellations
	for _, status := range jobStatuses {
		if strings.EqualFold(string(status), "failed") {
			return s.repository.UpdateStatus(ctx, w.ID, workflow.StatusFailed)
		}
		if strings.EqualFold(string(status), "cancelled") {
			return s.repository.UpdateStatus(ctx, w.ID, workflow.StatusCancelled)
		}
	}

	// Enqueue newly eligible pending jobs whose prerequisites are completed
	allCompleted := true
	for _, wj := range w.Jobs {
		status := jobStatuses[wj.JobID]
		if !strings.EqualFold(string(status), "completed") {
			allCompleted = false
		}

		if strings.EqualFold(string(status), "pending") {
			depsSatisfied := true
			for _, depID := range wj.DependsOnJobIDs {
				depStatus := jobStatuses[depID]
				if !strings.EqualFold(string(depStatus), "completed") {
					depsSatisfied = false
					break
				}
			}

			if depsSatisfied {
				stepResults := s.collectStepResults(ctx, w)
				currentJob, err := s.jobService.GetJob(ctx, wj.JobID)
				if err == nil && len(currentJob.Payload) > 0 {
					resolved := resolvePayload(currentJob.Payload, stepResults)
					_ = s.jobService.UpdatePayload(ctx, wj.JobID, resolved)
				}

				if _, err := s.jobService.EnqueuePersistedJob(ctx, wj.JobID); err != nil {
					return fmt.Errorf("enqueue eligible job %d: %w", wj.JobID, err)
				}
			}
		}
	}

	if allCompleted {
		return s.repository.UpdateStatus(ctx, w.ID, workflow.StatusCompleted)
	}

	return nil
}

func (s *Service) OnJobFailed(ctx context.Context, jobID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, err := s.repository.GetByJobID(ctx, jobID)
	if err != nil {
		return nil
	}

	if w.Status == workflow.StatusCompleted ||
		w.Status == workflow.StatusFailed ||
		w.Status == workflow.StatusCancelled {
		return nil
	}

	return s.repository.UpdateStatus(ctx, w.ID, workflow.StatusFailed)
}

func (s *Service) OnJobCancelled(ctx context.Context, jobID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, err := s.repository.GetByJobID(ctx, jobID)
	if err != nil {
		return nil
	}

	if w.Status == workflow.StatusCompleted ||
		w.Status == workflow.StatusFailed ||
		w.Status == workflow.StatusCancelled {
		return nil
	}

	return s.repository.UpdateStatus(ctx, w.ID, workflow.StatusCancelled)
}

func (s *Service) RecoverWorkflows(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	activeWorkflows, err := s.repository.GetActiveWorkflows(ctx)
	if err != nil {
		return fmt.Errorf("get active workflows: %w", err)
	}

	for _, w := range activeWorkflows {
		jobStatuses := make(map[int]job.Status, len(w.Jobs))

		for _, wj := range w.Jobs {
			j, err := s.jobService.GetJob(ctx, wj.JobID)
			if err != nil {
				return fmt.Errorf("get job %d in workflow %d: %w", wj.JobID, w.ID, err)
			}

			// If a job was running when server stopped, reset it to pending
			if strings.EqualFold(string(j.Status), "running") {
				if err := s.jobService.MarkPending(ctx, j.ID); err != nil {
					return fmt.Errorf("reset running job %d: %w", j.ID, err)
				}
				j.Status = job.StatusPending
			}

			jobStatuses[wj.JobID] = j.Status
		}

		hasFailed := false
		hasCancelled := false
		allCompleted := true

		for _, status := range jobStatuses {
			if strings.EqualFold(string(status), "failed") {
				hasFailed = true
			}
			if strings.EqualFold(string(status), "cancelled") {
				hasCancelled = true
			}
			if !strings.EqualFold(string(status), "completed") {
				allCompleted = false
			}
		}

		if hasFailed {
			_ = s.repository.UpdateStatus(ctx, w.ID, workflow.StatusFailed)
			continue
		}
		if hasCancelled {
			_ = s.repository.UpdateStatus(ctx, w.ID, workflow.StatusCancelled)
			continue
		}
		if allCompleted {
			_ = s.repository.UpdateStatus(ctx, w.ID, workflow.StatusCompleted)
			continue
		}

		if w.Status != workflow.StatusRunning {
			_ = s.repository.UpdateStatus(ctx, w.ID, workflow.StatusRunning)
		}

		// Enqueue newly eligible pending jobs whose prerequisites are completed
		for _, wj := range w.Jobs {
			if strings.EqualFold(string(jobStatuses[wj.JobID]), "pending") {
				depsSatisfied := true
				for _, depID := range wj.DependsOnJobIDs {
					if !strings.EqualFold(string(jobStatuses[depID]), "completed") {
						depsSatisfied = false
						break
					}
				}

				if depsSatisfied {
					stepResults := s.collectStepResults(ctx, w)
					currentJob, err := s.jobService.GetJob(ctx, wj.JobID)
					if err == nil && len(currentJob.Payload) > 0 {
						resolved := resolvePayload(currentJob.Payload, stepResults)
						_ = s.jobService.UpdatePayload(ctx, wj.JobID, resolved)
					}

					if _, err := s.jobService.EnqueuePersistedJob(ctx, wj.JobID); err != nil {
						return fmt.Errorf("recover eligible job %d: %w", wj.JobID, err)
					}
					fmt.Printf("Recovered eligible workflow job %d in workflow %d\n", wj.JobID, w.ID)
				}
			}
		}
	}

	return nil
}

func (s *Service) collectStepResults(ctx context.Context, w workflow.Workflow) map[string]map[string]any {
	results := make(map[string]map[string]any)
	for _, wj := range w.Jobs {
		j, err := s.jobService.GetJob(ctx, wj.JobID)
		if err == nil && len(j.Result) > 0 {
			key := wj.Key
			if key == "" {
				key = j.Type
			}
			results[key] = j.Result
		}
	}
	return results
}

func resolvePayload(payload map[string]any, stepResults map[string]map[string]any) map[string]any {
	resolved := make(map[string]any, len(payload))
	for k, v := range payload {
		resolved[k] = resolveValue(v, stepResults)
	}
	return resolved
}

func resolveValue(v any, stepResults map[string]map[string]any) any {
	switch val := v.(type) {
	case string:
		trimmed := strings.TrimSpace(val)
		if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") && !strings.Contains(trimmed[2:len(trimmed)-2], "{{") {
			expr := strings.TrimSpace(trimmed[2 : len(trimmed)-2])
			parts := strings.Split(expr, ".")
			if len(parts) >= 2 {
				stepKey := parts[0]
				field := parts[1]
				if res, ok := stepResults[stepKey]; ok {
					if targetVal, found := res[field]; found {
						return targetVal
					}
				}
			}
		}

		result := val
		for stepKey, res := range stepResults {
			for field, targetVal := range res {
				placeholder := fmt.Sprintf("{{%s.%s}}", stepKey, field)
				if strings.Contains(result, placeholder) {
					result = strings.ReplaceAll(result, placeholder, fmt.Sprint(targetVal))
				}
			}
		}
		return result

	case map[string]any:
		resMap := make(map[string]any, len(val))
		for mk, mv := range val {
			resMap[mk] = resolveValue(mv, stepResults)
		}
		return resMap

	case []any:
		resSlice := make([]any, len(val))
		for i, sv := range val {
			resSlice[i] = resolveValue(sv, stepResults)
		}
		return resSlice

	default:
		return v
	}
}

// StartScheduler runs a background scheduler checking for scheduled workflows.
func (s *Service) StartScheduler(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.triggerScheduledWorkflows(ctx)
			}
		}
	}()
}

func (s *Service) triggerScheduledWorkflows(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	activeWorkflows, err := s.repository.GetActiveWorkflows(ctx)
	if err != nil {
		return
	}

	now := time.Now()
	for _, w := range activeWorkflows {
		if w.CronExpression == "" || w.NextRunAt == nil {
			continue
		}

		if !now.Before(*w.NextRunAt) {
			schedule, err := cron.ParseStandard(w.CronExpression)
			if err != nil {
				continue
			}

			// Update next run time
			next := schedule.Next(now)
			w.NextRunAt = &next
			_ = s.repository.UpdateNextRunAt(ctx, w.ID, &next)

			// Trigger workflow execution (if completed or pending, restart all jobs)
			if w.Status == workflow.StatusCompleted || w.Status == workflow.StatusPending {
				_ = s.repository.UpdateStatus(ctx, w.ID, workflow.StatusRunning)
				for _, wj := range w.Jobs {
					_ = s.jobService.MarkPending(ctx, wj.JobID)
				}
				for _, wj := range w.Jobs {
					if len(wj.DependsOnJobIDs) == 0 {
						_, _ = s.jobService.EnqueuePersistedJob(ctx, wj.JobID)
					}
				}
			}
		}
	}
}
