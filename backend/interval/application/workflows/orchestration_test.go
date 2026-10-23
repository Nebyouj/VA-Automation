package workflows

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nebyoudamtew/go-job-queue/interval/application/jobs"
	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
	"github.com/nebyoudamtew/go-job-queue/interval/domain/workflow"
	"github.com/nebyoudamtew/go-job-queue/interval/infrastructure/queue"
	"github.com/nebyoudamtew/go-job-queue/interval/repository"
)

// inMemoryJobRepository is a thread-safe fake repository for unit tests.
type inMemoryJobRepository struct {
	mu           sync.Mutex
	jobs         map[int]job.Job
	nextID       int
	workflowJobs map[int]bool // jobID -> isWorkflowJob
}

func newInMemoryJobRepository() *inMemoryJobRepository {
	return &inMemoryJobRepository{
		jobs:         make(map[int]job.Job),
		nextID:       1,
		workflowJobs: make(map[int]bool),
	}
}

func (r *inMemoryJobRepository) markAsWorkflowJob(jobID int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflowJobs[jobID] = true
}

func (r *inMemoryJobRepository) Create(ctx context.Context, j job.Job) (job.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j.ID = r.nextID
	r.nextID++
	if j.Status == "" {
		j.Status = job.StatusPending
	}
	r.jobs[j.ID] = j
	return j, nil
}

func (r *inMemoryJobRepository) Delete(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[id]; !exists {
		return fmt.Errorf("job %d not found", id)
	}
	delete(r.jobs, id)
	return nil
}

func (r *inMemoryJobRepository) GetByID(ctx context.Context, id int) (job.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.jobs[id]
	if !exists {
		return job.Job{}, fmt.Errorf("job %d not found", id)
	}
	return j, nil
}

func (r *inMemoryJobRepository) GetAll(ctx context.Context, status string, jobType string, page int, limit int) ([]job.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []job.Job
	for _, j := range r.jobs {
		if status != "" && !strings.EqualFold(string(j.Status), status) {
			continue
		}
		if jobType != "" && j.Type != jobType {
			continue
		}
		result = append(result, j)
	}
	return result, nil
}

func (r *inMemoryJobRepository) IncrementAttempt(ctx context.Context, id int) (job.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.jobs[id]
	if !exists {
		return job.Job{}, fmt.Errorf("job %d not found", id)
	}
	j.Attempts++
	r.jobs[id] = j
	return j, nil
}

func (r *inMemoryJobRepository) MarkRunning(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}
	now := time.Now()
	j.Status = job.StatusRunning
	j.StartedAt = &now
	r.jobs[id] = j
	return nil
}

func (r *inMemoryJobRepository) MarkCompleted(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}
	now := time.Now()
	j.Status = job.StatusCompleted
	j.EndedAt = &now
	r.jobs[id] = j
	return nil
}

func (r *inMemoryJobRepository) MarkFailed(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}
	now := time.Now()
	j.Status = job.StatusFailed
	j.EndedAt = &now
	r.jobs[id] = j
	return nil
}

func (r *inMemoryJobRepository) MarkCancelled(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}
	now := time.Now()
	j.Status = job.StatusCancelled
	j.EndedAt = &now
	r.jobs[id] = j
	return nil
}

func (r *inMemoryJobRepository) GetRecoverableJobs(ctx context.Context) ([]job.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []job.Job
	for id, j := range r.jobs {
		// Scoped to standalone jobs, exactly matching Postgres query:
		// WHERE status IN ('pending', 'running') AND id NOT IN (SELECT job_id FROM workflow_jobs)
		if r.workflowJobs[id] {
			continue
		}
		if strings.EqualFold(string(j.Status), "pending") || strings.EqualFold(string(j.Status), "running") {
			result = append(result, j)
		}
	}
	return result, nil
}

func (r *inMemoryJobRepository) MarkPending(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}
	j.Status = job.StatusPending
	j.StartedAt = nil
	r.jobs[id] = j
	return nil
}

func (r *inMemoryJobRepository) GetStats(ctx context.Context) (repository.JobStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var stats repository.JobStats
	for _, j := range r.jobs {
		switch {
		case strings.EqualFold(string(j.Status), "pending"):
			stats.Pending++
		case strings.EqualFold(string(j.Status), "running"):
			stats.Running++
		case strings.EqualFold(string(j.Status), "completed"):
			stats.Completed++
		case strings.EqualFold(string(j.Status), "failed"):
			stats.Failed++
		case strings.EqualFold(string(j.Status), "cancelled"):
			stats.Cancelled++
		}
	}
	return stats, nil
}

func (r *inMemoryJobRepository) Reschedule(ctx context.Context, id int, scheduledAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}
	j.ScheduledAt = &scheduledAt
	j.Status = job.StatusPending
	j.Attempts = 0
	r.jobs[id] = j
	return nil
}

func (r *inMemoryJobRepository) SaveResult(ctx context.Context, id int, result map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}
	j.Result = result
	r.jobs[id] = j
	return nil
}

func (r *inMemoryJobRepository) UpdatePayload(ctx context.Context, id int, payload map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}
	j.Payload = payload
	r.jobs[id] = j
	return nil
}

// inMemoryWorkflowRepository is a thread-safe fake workflow repository for unit tests.
type inMemoryWorkflowRepository struct {
	mu           sync.Mutex
	workflows    map[int]workflow.Workflow
	jobWorkflows map[int]int // jobID -> workflowID
	nextID       int
	jobRepo      *inMemoryJobRepository
}

func newInMemoryWorkflowRepository(jobRepo *inMemoryJobRepository) *inMemoryWorkflowRepository {
	return &inMemoryWorkflowRepository{
		workflows:    make(map[int]workflow.Workflow),
		jobWorkflows: make(map[int]int),
		nextID:       1,
		jobRepo:      jobRepo,
	}
}

func (r *inMemoryWorkflowRepository) Create(ctx context.Context, w workflow.Workflow) (workflow.Workflow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	w.ID = r.nextID
	r.nextID++
	w.CreatedAt = time.Now()
	if w.Status == "" {
		w.Status = workflow.StatusPending
	}

	for _, j := range w.Jobs {
		r.jobWorkflows[j.JobID] = w.ID
		if r.jobRepo != nil {
			r.jobRepo.markAsWorkflowJob(j.JobID)
		}
	}

	r.workflows[w.ID] = w
	return w, nil
}

func (r *inMemoryWorkflowRepository) GetByID(ctx context.Context, id int) (workflow.Workflow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, exists := r.workflows[id]
	if !exists {
		return workflow.Workflow{}, fmt.Errorf("workflow %d not found", id)
	}
	return w, nil
}

func (r *inMemoryWorkflowRepository) GetAll(ctx context.Context, status string, page int, limit int) ([]workflow.Workflow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []workflow.Workflow
	for _, w := range r.workflows {
		if status != "" && !strings.EqualFold(string(w.Status), status) {
			continue
		}
		result = append(result, w)
	}
	return result, nil
}

func (r *inMemoryWorkflowRepository) UpdateStatus(ctx context.Context, id int, status workflow.Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, exists := r.workflows[id]
	if !exists {
		return fmt.Errorf("workflow %d not found", id)
	}

	w.Status = status
	now := time.Now()
	if status == workflow.StatusRunning && w.StartedAt == nil {
		w.StartedAt = &now
	}
	if status == workflow.StatusCompleted || status == workflow.StatusFailed || status == workflow.StatusCancelled {
		w.EndedAt = &now
	}

	r.workflows[id] = w
	return nil
}

func (r *inMemoryWorkflowRepository) GetByJobID(ctx context.Context, jobID int) (workflow.Workflow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	workflowID, exists := r.jobWorkflows[jobID]
	if !exists {
		return workflow.Workflow{}, fmt.Errorf("workflow for job %d not found", jobID)
	}

	w, exists := r.workflows[workflowID]
	if !exists {
		return workflow.Workflow{}, fmt.Errorf("workflow %d not found", workflowID)
	}
	return w, nil
}

func (r *inMemoryWorkflowRepository) UpdateNextRunAt(ctx context.Context, id int, nextRunAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, exists := r.workflows[id]
	if !exists {
		return fmt.Errorf("workflow %d not found", id)
	}
	w.NextRunAt = nextRunAt
	r.workflows[id] = w
	return nil
}

func (r *inMemoryWorkflowRepository) GetActiveWorkflows(ctx context.Context) ([]workflow.Workflow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []workflow.Workflow
	for _, w := range r.workflows {
		if w.Status == workflow.StatusPending || w.Status == workflow.StatusRunning || strings.TrimSpace(w.CronExpression) != "" {
			result = append(result, w)
		}
	}
	return result, nil
}

func (r *inMemoryWorkflowRepository) InitSchema(ctx context.Context) error {
	return nil
}

// setupWorkflowTest creates a wired test environment with in-memory repositories.
func setupWorkflowTest() (*jobs.Service, *Service, *queue.Queue, *inMemoryJobRepository, *inMemoryWorkflowRepository) {
	q := queue.NewQueue(100)
	jobRepo := newInMemoryJobRepository()
	jobService := jobs.NewService(q, jobRepo)
	wfRepo := newInMemoryWorkflowRepository(jobRepo)
	wfService := NewService(jobService, wfRepo)
	jobService.SetEventListener(wfService)

	return jobService, wfService, q, jobRepo, wfRepo
}

func TestWorkflowRootJobEnqueuedDependentBlocked(t *testing.T) {
	ctx := context.Background()
	jobService, wfService, q, jobRepo, wfRepo := setupWorkflowTest()
	_ = jobService
	_ = wfRepo

	// Workflow: step1 (root) -> step2 (depends on step1)
	wf, err := wfService.CreateWorkflow(ctx, CreateRequest{
		Name: "Two-step workflow",
		Jobs: []JobDefinition{
			{Key: "step1", Type: "failing_job"},
			{Key: "step2", Type: "failing_job", DependsOn: []string{"step1"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow error: %v", err)
	}

	if wf.Status != workflow.StatusRunning {
		t.Fatalf("expected workflow status running, got %s", wf.Status)
	}

	// Dequeue from queue: only step1 should be enqueued
	qJob, ok := q.Dequeue(ctx)
	if !ok {
		t.Fatal("expected step1 to be in queue")
	}

	step1ID := wf.Jobs[0].JobID
	step2ID := wf.Jobs[1].JobID

	if qJob.Job.ID != step1ID {
		t.Fatalf("expected job %d (step1) to be dequeued, got %d", step1ID, qJob.Job.ID)
	}

	// Verify step2 remains pending in repo and is not ready/queued
	step2, err := jobRepo.GetByID(ctx, step2ID)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if step2.Status != job.StatusPending {
		t.Fatalf("expected step2 to be pending, got %s", step2.Status)
	}

	// Verify queue is now empty (step2 was NOT enqueued)
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_, ok = q.Dequeue(timeoutCtx)
	if ok {
		t.Fatal("step2 should NOT be in the queue while step1 is not completed")
	}
}

func TestDependentJobBecomesEligibleAfterPrerequisitesSucceed(t *testing.T) {
	ctx := context.Background()
	jobService, wfService, q, _, wfRepo := setupWorkflowTest()

	// Workflow: step1 -> step2
	wf, err := wfService.CreateWorkflow(ctx, CreateRequest{
		Name: "Sequential workflow",
		Jobs: []JobDefinition{
			{Key: "step1", Type: "failing_job"},
			{Key: "step2", Type: "failing_job", DependsOn: []string{"step1"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow error: %v", err)
	}

	step1ID := wf.Jobs[0].JobID
	step2ID := wf.Jobs[1].JobID

	// Root job is dequeued
	qJob, ok := q.Dequeue(ctx)
	if !ok || qJob.Job.ID != step1ID {
		t.Fatalf("expected step1 to be dequeued")
	}

	// Simulate successful completion of step1
	if err := jobService.MarkCompleted(step1ID); err != nil {
		t.Fatalf("MarkCompleted error: %v", err)
	}

	// Now step2 must become eligible and be present in queue
	qJob2, ok := q.Dequeue(ctx)
	if !ok {
		t.Fatal("step2 was not enqueued after step1 completed")
	}
	if qJob2.Job.ID != step2ID {
		t.Fatalf("expected job %d (step2), got %d", step2ID, qJob2.Job.ID)
	}

	// Complete step2
	if err := jobService.MarkCompleted(step2ID); err != nil {
		t.Fatalf("MarkCompleted step2 error: %v", err)
	}

	// Workflow must now be completed
	updatedWf, err := wfRepo.GetByID(ctx, wf.ID)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if updatedWf.Status != workflow.StatusCompleted {
		t.Fatalf("expected workflow status completed, got %s", updatedWf.Status)
	}
}

func TestMultiplePrerequisitesWaitUntilAllSucceed(t *testing.T) {
	ctx := context.Background()
	jobService, wfService, q, _, wfRepo := setupWorkflowTest()

	// Workflow: job1 & job2 (roots) -> job3 (depends on both job1 and job2)
	wf, err := wfService.CreateWorkflow(ctx, CreateRequest{
		Name: "Fan-in workflow",
		Jobs: []JobDefinition{
			{Key: "job1", Type: "failing_job"},
			{Key: "job2", Type: "failing_job"},
			{Key: "job3", Type: "failing_job", DependsOn: []string{"job1", "job2"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow error: %v", err)
	}

	job1ID := wf.Jobs[0].JobID
	job2ID := wf.Jobs[1].JobID
	job3ID := wf.Jobs[2].JobID

	// Both root jobs (job1 and job2) should be in the queue
	dequeuedRoots := make(map[int]bool)
	q1, ok1 := q.Dequeue(ctx)
	q2, ok2 := q.Dequeue(ctx)
	if !ok1 || !ok2 {
		t.Fatal("expected both root jobs to be dequeued")
	}
	dequeuedRoots[q1.Job.ID] = true
	dequeuedRoots[q2.Job.ID] = true
	if !dequeuedRoots[job1ID] || !dequeuedRoots[job2ID] {
		t.Fatalf("expected roots %d and %d in queue, got %v", job1ID, job2ID, dequeuedRoots)
	}

	// Mark only job1 completed
	if err := jobService.MarkCompleted(job1ID); err != nil {
		t.Fatalf("MarkCompleted job1 error: %v", err)
	}

	// Job3 must NOT be enqueued yet because job2 is not completed
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if _, ok := q.Dequeue(timeoutCtx); ok {
		t.Fatal("job3 should not be enqueued while job2 is still incomplete")
	}

	// Now mark job2 completed
	if err := jobService.MarkCompleted(job2ID); err != nil {
		t.Fatalf("MarkCompleted job2 error: %v", err)
	}

	// Now job3 MUST be enqueued
	q3, ok3 := q.Dequeue(ctx)
	if !ok3 {
		t.Fatal("job3 was not enqueued after all prerequisites completed")
	}
	if q3.Job.ID != job3ID {
		t.Fatalf("expected job %d (job3), got %d", job3ID, q3.Job.ID)
	}

	// Complete job3
	if err := jobService.MarkCompleted(job3ID); err != nil {
		t.Fatalf("MarkCompleted job3 error: %v", err)
	}

	updatedWf, err := wfRepo.GetByID(ctx, wf.ID)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if updatedWf.Status != workflow.StatusCompleted {
		t.Fatalf("expected workflow status completed, got %s", updatedWf.Status)
	}
}

func TestFailedPrerequisitePreventsDependentExecution(t *testing.T) {
	ctx := context.Background()
	jobService, wfService, q, _, wfRepo := setupWorkflowTest()

	// Workflow: step1 -> step2
	wf, err := wfService.CreateWorkflow(ctx, CreateRequest{
		Name: "Failing workflow",
		Jobs: []JobDefinition{
			{Key: "step1", Type: "failing_job"},
			{Key: "step2", Type: "failing_job", DependsOn: []string{"step1"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow error: %v", err)
	}

	step1ID := wf.Jobs[0].JobID

	// Dequeue step1
	_, ok := q.Dequeue(ctx)
	if !ok {
		t.Fatal("expected step1 to be in queue")
	}

	// Mark step1 failed
	if err := jobService.MarkFailed(step1ID); err != nil {
		t.Fatalf("MarkFailed error: %v", err)
	}

	// Workflow should be failed
	updatedWf, err := wfRepo.GetByID(ctx, wf.ID)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if updatedWf.Status != workflow.StatusFailed {
		t.Fatalf("expected workflow status failed, got %s", updatedWf.Status)
	}

	// Step2 should NOT be in the queue
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if _, ok := q.Dequeue(timeoutCtx); ok {
		t.Fatal("step2 should never be enqueued when step1 failed")
	}
}

func TestCancelledPrerequisitePreventsDependentExecution(t *testing.T) {
	ctx := context.Background()
	jobService, wfService, q, _, wfRepo := setupWorkflowTest()

	// Workflow: step1 -> step2
	wf, err := wfService.CreateWorkflow(ctx, CreateRequest{
		Name: "Cancelled workflow",
		Jobs: []JobDefinition{
			{Key: "step1", Type: "failing_job"},
			{Key: "step2", Type: "failing_job", DependsOn: []string{"step1"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow error: %v", err)
	}

	step1ID := wf.Jobs[0].JobID

	// Cancel step1
	if err := jobService.CancelJob(ctx, step1ID); err != nil {
		t.Fatalf("CancelJob error: %v", err)
	}

	// Workflow should be cancelled
	updatedWf, err := wfRepo.GetByID(ctx, wf.ID)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if updatedWf.Status != workflow.StatusCancelled {
		t.Fatalf("expected workflow status cancelled, got %s", updatedWf.Status)
	}

	// Step2 should NOT be enqueued
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if _, ok := q.Dequeue(timeoutCtx); ok {
		t.Fatal("step2 should never be enqueued when step1 was cancelled")
	}
}

func TestStartupRecoveryWorkflowAware(t *testing.T) {
	ctx := context.Background()
	q := queue.NewQueue(100)
	jobRepo := newInMemoryJobRepository()
	jobService := jobs.NewService(q, jobRepo)
	wfRepo := newInMemoryWorkflowRepository(jobRepo)
	wfService := NewService(jobService, wfRepo)
	jobService.SetEventListener(wfService)

	// 1. Create a standalone job that was running when process stopped
	standaloneJob, err := jobRepo.Create(ctx, job.Job{
		Type:     "failing_job",
		Status:   job.StatusRunning,
		Priority: 10,
	})
	if err != nil {
		t.Fatalf("Create standalone job error: %v", err)
	}

	// 2. Create a workflow: job1 (completed) -> job2 (was running) -> job3 (blocked pending)
	j1, _ := jobRepo.Create(ctx, job.Job{Type: "failing_job", Status: job.StatusCompleted})
	j2, _ := jobRepo.Create(ctx, job.Job{Type: "failing_job", Status: job.StatusRunning})
	j3, _ := jobRepo.Create(ctx, job.Job{Type: "failing_job", Status: job.StatusPending})

	wf, err := wfRepo.Create(ctx, workflow.Workflow{
		Name:   "Interrupted workflow",
		Status: workflow.StatusRunning,
		Jobs: []workflow.WorkflowJob{
			{JobID: j1.ID, Position: 0},
			{JobID: j2.ID, Position: 1, DependsOnJobIDs: []int{j1.ID}},
			{JobID: j3.ID, Position: 2, DependsOnJobIDs: []int{j2.ID}},
		},
	})
	if err != nil {
		t.Fatalf("Create workflow error: %v", err)
	}
	_ = wf

	// Run recovery
	if err := jobService.RecoverJobs(ctx); err != nil {
		t.Fatalf("RecoverJobs error: %v", err)
	}

	// Both the standalone job and workflow job2 (which was running) should be enqueued.
	// Workflow job3 (blocked by job2) and job1 (already completed) must NOT be enqueued!
	dequeuedIDs := make(map[int]bool)
	for i := 0; i < 2; i++ {
		qJob, ok := q.Dequeue(ctx)
		if !ok {
			t.Fatalf("expected job %d to be dequeued", i+1)
		}
		dequeuedIDs[qJob.Job.ID] = true
	}

	if !dequeuedIDs[standaloneJob.ID] {
		t.Errorf("expected standalone job %d to be recovered", standaloneJob.ID)
	}
	if !dequeuedIDs[j2.ID] {
		t.Errorf("expected interrupted workflow job %d to be recovered", j2.ID)
	}

	// Verify no more jobs are in queue (j3 is NOT enqueued!)
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if extraJob, ok := q.Dequeue(timeoutCtx); ok {
		t.Fatalf("unexpected job %d in queue (dependency-blocked job was enqueued!)", extraJob.Job.ID)
	}
}

func TestRepeatedCompletionAndRecoveryNoDuplicateExecution(t *testing.T) {
	ctx := context.Background()
	jobService, wfService, q, _, _ := setupWorkflowTest()

	// Workflow: step1 -> step2
	wf, err := wfService.CreateWorkflow(ctx, CreateRequest{
		Name: "Duplicate test workflow",
		Jobs: []JobDefinition{
			{Key: "step1", Type: "failing_job"},
			{Key: "step2", Type: "failing_job", DependsOn: []string{"step1"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow error: %v", err)
	}

	step1ID := wf.Jobs[0].JobID
	step2ID := wf.Jobs[1].JobID

	// Dequeue step1
	q1, ok := q.Dequeue(ctx)
	if !ok || q1.Job.ID != step1ID {
		t.Fatalf("expected step1 to be dequeued")
	}

	// Mark completed twice (simulating duplicate worker completion event)
	if err := jobService.MarkCompleted(step1ID); err != nil {
		t.Fatalf("first MarkCompleted error: %v", err)
	}
	if err := jobService.MarkCompleted(step1ID); err != nil {
		t.Fatalf("second MarkCompleted error: %v", err)
	}

	// Step2 should only be enqueued ONCE
	q2, ok := q.Dequeue(ctx)
	if !ok || q2.Job.ID != step2ID {
		t.Fatalf("expected step2 to be dequeued")
	}

	// Verify queue is empty (no duplicate step2!)
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if dupJob, ok := q.Dequeue(timeoutCtx); ok {
		t.Fatalf("step2 was duplicated in queue: job %d", dupJob.Job.ID)
	}

	// Run RecoverJobs multiple times
	for i := 0; i < 3; i++ {
		if err := jobService.RecoverJobs(ctx); err != nil {
			t.Fatalf("RecoverJobs error on iteration %d: %v", i, err)
		}
	}

	// Queue should STILL be empty
	timeoutCtx2, cancel2 := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel2()
	if extraJob, ok := q.Dequeue(timeoutCtx2); ok {
		t.Fatalf("duplicate job enqueued during recovery: %d", extraJob.Job.ID)
	}
}

// mockExecutor for end-to-end testing
type recordingExecutor struct {
	mu            sync.Mutex
	executedOrder []int
}

func (e *recordingExecutor) Execute(ctx context.Context, j job.Job) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executedOrder = append(e.executedOrder, j.ID)
	return nil
}

func TestEndToEndWorkflowExecutionWithWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := queue.NewQueue(10)
	jobRepo := newInMemoryJobRepository()
	jobService := jobs.NewService(q, jobRepo)
	wfRepo := newInMemoryWorkflowRepository(jobRepo)
	wfService := NewService(jobService, wfRepo)
	jobService.SetEventListener(wfService)

	exec := &recordingExecutor{}

	// Start 3 workers
	q.StartWorker(ctx, 1, exec, jobService)
	q.StartWorker(ctx, 2, exec, jobService)
	q.StartWorker(ctx, 3, exec, jobService)

	// Create a 4-job dependency chain:
	// root -> process -> report -> email
	wf, err := wfService.CreateWorkflow(ctx, CreateRequest{
		Name: "End-to-end chain",
		Jobs: []JobDefinition{
			{Key: "root", Type: "failing_job"},
			{Key: "process", Type: "failing_job", DependsOn: []string{"root"}},
			{Key: "report", Type: "failing_job", DependsOn: []string{"process"}},
			{Key: "email", Type: "failing_job", DependsOn: []string{"report"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow error: %v", err)
	}

	rootID := wf.Jobs[0].JobID
	processID := wf.Jobs[1].JobID
	reportID := wf.Jobs[2].JobID
	emailID := wf.Jobs[3].JobID

	// Poll until the workflow reaches completed status or times out
	deadline := time.After(5 * time.Second)
	for {
		w, err := wfRepo.GetByID(ctx, wf.ID)
		if err == nil && w.Status == workflow.StatusCompleted {
			break
		}

		select {
		case <-deadline:
			t.Fatalf("workflow did not complete within deadline; last status was %s", w.Status)
		case <-time.After(20 * time.Millisecond):
		}
	}

	cancel()
	q.Close()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	q.Wait(shutdownCtx)

	exec.mu.Lock()
	order := exec.executedOrder
	exec.mu.Unlock()

	expectedOrder := []int{rootID, processID, reportID, emailID}
	if len(order) != len(expectedOrder) {
		t.Fatalf("expected %d executed jobs, got %d (%v)", len(expectedOrder), len(order), order)
	}

	for i := range expectedOrder {
		if order[i] != expectedOrder[i] {
			t.Errorf("position %d: expected job %d, got %d", i, expectedOrder[i], order[i])
		}
	}
}

func TestDataPassingBetweenWorkflowJobs(t *testing.T) {
	ctx := context.Background()
	jobService, wfService, q, jobRepo, _ := setupWorkflowTest()

	// Workflow:
	// step1 (produces output) -> step2 (interpolates step1 output)
	wf, err := wfService.CreateWorkflow(ctx, CreateRequest{
		Name: "Data passing workflow",
		Jobs: []JobDefinition{
			{
				Key:  "step1",
				Type: "failing_job",
				Payload: map[string]any{
					"query": "artificial intelligence",
				},
			},
			{
				Key:       "step2",
				Type:      "failing_job",
				DependsOn: []string{"step1"},
				Payload: map[string]any{
					"title":   "Summary of {{step1.topic}}",
					"message": "URL was {{step1.url}}",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow error: %v", err)
	}

	step1ID := wf.Jobs[0].JobID
	step2ID := wf.Jobs[1].JobID

	// Dequeue step1
	qJob, ok := q.Dequeue(ctx)
	if !ok || qJob.Job.ID != step1ID {
		t.Fatalf("expected step1 to be dequeued")
	}

	// Simulate step1 saving results
	err = jobService.SaveResult(step1ID, map[string]any{
		"topic": "AI Research",
		"url":   "https://ai.google.dev",
	})
	if err != nil {
		t.Fatalf("SaveResult error: %v", err)
	}

	// Complete step1
	if err := jobService.MarkCompleted(step1ID); err != nil {
		t.Fatalf("MarkCompleted step1 error: %v", err)
	}

	// Step2 should be enqueued now
	qJob2, ok := q.Dequeue(ctx)
	if !ok || qJob2.Job.ID != step2ID {
		t.Fatalf("expected step2 to be dequeued")
	}

	// Verify step2 payload has the resolved upstream variables
	updatedStep2, err := jobRepo.GetByID(ctx, step2ID)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}

	if gotTitle := updatedStep2.Payload["title"]; gotTitle != "Summary of AI Research" {
		t.Errorf("step2 payload[title] = %v, want 'Summary of AI Research'", gotTitle)
	}
	if gotMsg := updatedStep2.Payload["message"]; gotMsg != "URL was https://ai.google.dev" {
		t.Errorf("step2 payload[message] = %v, want 'URL was https://ai.google.dev'", gotMsg)
	}
}

func TestWorkflowCronScheduling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, wfService, _, _, _ := setupWorkflowTest()

	// Workflow with cron expression
	wf, err := wfService.CreateWorkflow(ctx, CreateRequest{
		Name:           "Hourly workflow",
		CronExpression: "0 * * * *",
		Jobs: []JobDefinition{
			{Key: "step1", Type: "failing_job"},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow error: %v", err)
	}

	if wf.CronExpression != "0 * * * *" {
		t.Errorf("expected cron_expression '0 * * * *', got %q", wf.CronExpression)
	}
	if wf.NextRunAt == nil {
		t.Fatal("expected next_run_at to be populated")
	}
	if wf.NextRunAt.Before(time.Now()) {
		t.Errorf("expected next_run_at to be in future, got %v", wf.NextRunAt)
	}
}

func TestWorkflowCronRecurringExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobService, wfService, q, jobRepo, wfRepo := setupWorkflowTest()

	// Workflow: step1 -> step2
	wf, err := wfService.CreateWorkflow(ctx, CreateRequest{
		Name:           "Recurring workflow",
		CronExpression: "* * * * *",
		Jobs: []JobDefinition{
			{Key: "step1", Type: "failing_job"},
			{Key: "step2", Type: "failing_job", DependsOn: []string{"step1"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow error: %v", err)
	}

	step1ID := wf.Jobs[0].JobID
	step2ID := wf.Jobs[1].JobID

	// Initial run: step1 dequeued and completed
	qJob, ok := q.Dequeue(ctx)
	if !ok || qJob.Job.ID != step1ID {
		t.Fatalf("expected step1 to be dequeued")
	}
	if err := jobService.MarkCompleted(step1ID); err != nil {
		t.Fatalf("MarkCompleted step1 error: %v", err)
	}

	// step2 dequeued and completed
	qJob2, ok := q.Dequeue(ctx)
	if !ok || qJob2.Job.ID != step2ID {
		t.Fatalf("expected step2 to be dequeued")
	}
	if err := jobService.MarkCompleted(step2ID); err != nil {
		t.Fatalf("MarkCompleted step2 error: %v", err)
	}

	// Workflow should now be completed
	completedWf, err := wfRepo.GetByID(ctx, wf.ID)
	if err != nil || completedWf.Status != workflow.StatusCompleted {
		t.Fatalf("expected workflow completed, got %v", completedWf.Status)
	}

	// Backdate NextRunAt to simulate cron schedule elapsed
	past := time.Now().Add(-10 * time.Second)
	_ = wfRepo.UpdateNextRunAt(ctx, wf.ID, &past)

	// Trigger scheduler tick
	wfService.triggerScheduledWorkflows(ctx)

	// Check workflow status is running again
	updatedWf, err := wfRepo.GetByID(ctx, wf.ID)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if updatedWf.Status != workflow.StatusRunning {
		t.Errorf("expected status %q, got %q", workflow.StatusRunning, updatedWf.Status)
	}

	// NextRunAt should be updated to the future
	if updatedWf.NextRunAt == nil || !updatedWf.NextRunAt.After(time.Now()) {
		t.Errorf("expected next_run_at to be in future, got %v", updatedWf.NextRunAt)
	}

	// All jobs should be reset to pending
	for _, wj := range wf.Jobs {
		j, err := jobRepo.GetByID(ctx, wj.JobID)
		if err != nil {
			t.Fatalf("GetByID job error: %v", err)
		}
		if j.Status != job.StatusPending {
			t.Errorf("expected job %d status pending, got %q", wj.JobID, j.Status)
		}
	}

	// Root job (step1) should be enqueued for the new cycle
	qJobReTrigger, ok := q.Dequeue(ctx)
	if !ok || qJobReTrigger.Job.ID != step1ID {
		t.Fatalf("expected root job step1 (%d) to be enqueued for recurring run, got %v", step1ID, qJobReTrigger)
	}
}
