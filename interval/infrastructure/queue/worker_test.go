package queue

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
)

type mockJobExecutor struct {
	mu     sync.Mutex
	errors []error
	calls  int
}

func (m *mockJobExecutor) Execute(
	ctx context.Context,
	j job.Job,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls++

	if err := ctx.Err(); err != nil {
		return err
	}

	index := m.calls - 1
	if index < len(m.errors) {
		return m.errors[index]
	}

	return nil
}

func (m *mockJobExecutor) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

type mockStatusUpdater struct {
	mu sync.Mutex

	currentJob job.Job
	attempts   int
	running    int
	completed  int
	failed     int

	rescheduledAt time.Time
	terminal      chan string
}

func newMockStatusUpdater(j job.Job) *mockStatusUpdater {
	return &mockStatusUpdater{
		currentJob: j,
		terminal:   make(chan string, 1),
	}
}

func (m *mockStatusUpdater) MarkRunning(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.running++
	return nil
}

func (m *mockStatusUpdater) MarkCompleted(id int) error {
	m.mu.Lock()
	m.completed++
	m.mu.Unlock()

	m.terminal <- "completed"
	return nil
}

func (m *mockStatusUpdater) MarkFailed(id int) error {
	m.mu.Lock()
	m.failed++
	m.mu.Unlock()

	m.terminal <- "failed"
	return nil
}

func (m *mockStatusUpdater) IncrementAttempt(id int) (job.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.attempts++
	updated := m.currentJob
	updated.Attempts = m.attempts

	return updated, nil
}

func (m *mockStatusUpdater) RescheduleJob(
	id int,
	scheduledAt time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rescheduledAt = scheduledAt
	return nil
}

func (m *mockStatusUpdater) snapshot() (
	attempts int,
	running int,
	completed int,
	failed int,
	rescheduledAt time.Time,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.attempts, m.running, m.completed, m.failed, m.rescheduledAt
}

func runWorkerUntilTerminal(
	t *testing.T,
	j job.Job,
	exec *mockJobExecutor,
	updater *mockStatusUpdater,
) {
	t.Helper()

	q := NewQueue(1)
	ctx, cancel := context.WithCancel(context.Background())
	jobCtx, cancelJob := context.WithCancel(context.Background())

	t.Cleanup(func() {
		cancel()
		cancelJob()
		q.Close()
	})

	err := q.Enqueue(ctx, QueueJob{
		Job:     j,
		Context: jobCtx,
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		worker(ctx, 1, q, exec, updater)
	}()

	select {
	case result := <-updater.terminal:
		if result != "completed" && result != "failed" {
			t.Fatalf("unexpected terminal status: %s", result)
		}
		q.Close()

	case <-time.After(5 * time.Second):
		t.Fatal("worker did not reach a terminal status")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after queue closure")
	}
}

func TestWorkerCompletesSuccessfulJob(t *testing.T) {
	j := job.Job{
		ID:          1,
		Type:        "failing_job",
		MaxAttempts: 3,
	}

	exec := &mockJobExecutor{}
	updater := newMockStatusUpdater(j)

	runWorkerUntilTerminal(t, j, exec, updater)

	attempts, running, completed, failed, _ := updater.snapshot()

	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
	if running != 1 {
		t.Errorf("running = %d, want 1", running)
	}
	if completed != 1 {
		t.Errorf("completed = %d, want 1", completed)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
	if exec.Calls() != 1 {
		t.Errorf("executor calls = %d, want 1", exec.Calls())
	}
}

func TestWorkerRetriesThenSucceeds(t *testing.T) {
	j := job.Job{
		ID:          2,
		Type:        "failing_job",
		MaxAttempts: 3,
	}

	exec := &mockJobExecutor{
		errors: []error{
			errors.New("temporary failure"),
		},
	}
	updater := newMockStatusUpdater(j)

	runWorkerUntilTerminal(t, j, exec, updater)

	attempts, running, completed, failed, _ := updater.snapshot()

	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
	if running != 2 {
		t.Errorf("running = %d, want 2", running)
	}
	if completed != 1 {
		t.Errorf("completed = %d, want 1", completed)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
	if exec.Calls() != 2 {
		t.Errorf("executor calls = %d, want 2", exec.Calls())
	}
}

func TestWorkerMarksJobFailedAfterMaxAttempts(t *testing.T) {
	j := job.Job{
		ID:          3,
		Type:        "failing_job",
		MaxAttempts: 1,
	}

	exec := &mockJobExecutor{
		errors: []error{
			errors.New("permanent failure"),
		},
	}
	updater := newMockStatusUpdater(j)

	runWorkerUntilTerminal(t, j, exec, updater)

	attempts, running, completed, failed, _ := updater.snapshot()

	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
	if running != 1 {
		t.Errorf("running = %d, want 1", running)
	}
	if completed != 0 {
		t.Errorf("completed = %d, want 0", completed)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
	if exec.Calls() != 1 {
		t.Errorf("executor calls = %d, want 1", exec.Calls())
	}
}

func TestWorkerCancellationDuringRetryDelay(t *testing.T) {
	j := job.Job{
		ID:          4,
		Type:        "failing_job",
		MaxAttempts: 3,
	}

	exec := &mockJobExecutor{
		errors: []error{
			errors.New("temporary failure"),
		},
	}
	updater := newMockStatusUpdater(j)

	q := NewQueue(1)
	ctx, cancelWorker := context.WithCancel(context.Background())
	jobCtx, cancelJob := context.WithCancel(context.Background())

	if err := q.Enqueue(ctx, QueueJob{
		Job:     j,
		Context: jobCtx,
	}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker(ctx, 1, q, exec, updater)
	}()

	// Wait until the first attempt has failed and the retry delay begins.
	deadline := time.After(3 * time.Second)
	for {
		attempts, _, _, _, _ := updater.snapshot()
		if attempts >= 1 {
			break
		}

		select {
		case <-deadline:
			cancelWorker()
			cancelJob()
			q.Close()
			t.Fatal("worker did not start the first attempt")
		case <-time.After(10 * time.Millisecond):
		}
	}

	cancelJob()
	cancelWorker()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		cancelWorker()
		q.Close()
		t.Fatal("worker did not stop after job cancellation")
	}

	q.Close()

	if got := exec.Calls(); got != 1 {
		t.Errorf("executor calls = %d, want 1", got)
	}
}

func TestWorkerReschedulesRecurringJob(t *testing.T) {
	j := job.Job{
		ID:                 5,
		Type:               "failing_job",
		MaxAttempts:        3,
		RepeatEverySeconds: 3600,
	}

	exec := &mockJobExecutor{}
	updater := newMockStatusUpdater(j)

	q := NewQueue(1)
	ctx, cancelWorker := context.WithCancel(context.Background())
	jobCtx, cancelJob := context.WithCancel(context.Background())

	defer cancelWorker()
	defer cancelJob()
	defer q.Close()

	if err := q.Enqueue(ctx, QueueJob{
		Job:     j,
		Context: jobCtx,
	}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker(ctx, 1, q, exec, updater)
	}()

	deadline := time.After(3 * time.Second)
	for {
		_, _, _, _, scheduledAt := updater.snapshot()
		if !scheduledAt.IsZero() {
			break
		}

		select {
		case <-deadline:
			cancelWorker()
			t.Fatal("worker did not reschedule the recurring job")
		case <-time.After(10 * time.Millisecond):
		}
	}

	_, _, completed, failed, scheduledAt := updater.snapshot()

	if completed != 0 {
		t.Errorf("completed = %d, want 0 for a recurring job", completed)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
	if scheduledAt.Before(time.Now()) {
		t.Errorf("next scheduled time %v is in the past", scheduledAt)
	}
	if exec.Calls() != 1 {
		t.Errorf("executor calls = %d, want 1", exec.Calls())
	}

	// Stop the worker without waiting for the next hourly execution.
	cancelWorker()
	q.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop")
	}
}
