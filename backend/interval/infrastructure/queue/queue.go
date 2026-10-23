package queue

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
)

type QueueJob struct {
	Job     job.Job
	Context context.Context
}

type priorityItem struct {
	job QueueJob
}

type JobExecutor interface {
	Execute(ctx context.Context, j job.Job) error
}

type priorityQueue []priorityItem

func (p priorityQueue) Len() int {
	return len(p)
}

func (p priorityQueue) Less(i, j int) bool {
	return p[i].job.Job.Priority > p[j].job.Job.Priority
}

func (p priorityQueue) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p *priorityQueue) Push(x any) {
	*p = append(*p, x.(priorityItem))
}

func (p *priorityQueue) Pop() any {
	old := *p
	n := len(old)

	item := old[n-1]
	*p = old[:n-1]

	return item
}

type Queue struct {
	mu     sync.Mutex
	jobs   priorityQueue
	closed bool
	wg     sync.WaitGroup
	wake   chan struct{}
}

func NewQueue(size int) *Queue {
	q := &Queue{
		wake: make(chan struct{}, 1),
	}

	heap.Init(&q.jobs)

	return q
}

func (q *Queue) Enqueue(ctx context.Context, job QueueJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	heap.Push(&q.jobs, priorityItem{
		job: job,
	})

	q.signal()

	return nil
}

func (q *Queue) signal() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func (q *Queue) Dequeue(ctx context.Context) (QueueJob, bool) {
	for {
		q.mu.Lock()

		if q.closed {
			q.mu.Unlock()
			return QueueJob{}, false
		}

		now := time.Now()

		readyIndex := -1
		highestPriority := -1 << 63

		var earliestScheduled time.Time

		for i, item := range q.jobs {
			j := item.job.Job

			// Cancelled jobs should never be executed.
			if item.job.Context.Err() != nil {
				continue
			}

			// Job is ready immediately.
			if j.ScheduledAt == nil || !j.ScheduledAt.After(now) {
				if readyIndex == -1 ||
					j.Priority > highestPriority {
					readyIndex = i
					highestPriority = j.Priority
				}

				continue
			}

			// Track the earliest future job.
			if earliestScheduled.IsZero() ||
				j.ScheduledAt.Before(earliestScheduled) {
				earliestScheduled = *j.ScheduledAt
			}
		}

		if readyIndex != -1 {
			item := heap.Remove(
				&q.jobs,
				readyIndex,
			).(priorityItem)

			q.mu.Unlock()

			return item.job, true
		}

		q.mu.Unlock()

		// No jobs at all.
		if earliestScheduled.IsZero() {
			select {
			case <-ctx.Done():
				return QueueJob{}, false

			case <-q.wake:
				continue
			}
		}

		// Jobs exist, but they're scheduled for later.
		delay := time.Until(earliestScheduled)

		if delay < 0 {
			continue
		}

		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			return QueueJob{}, false

		case <-q.wake:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			// A new job may have arrived, or the queue state changed.
			continue

		case <-timer.C:
			continue
		}
	}
}

func (q *Queue) StartWorker(
	ctx context.Context,
	id int,
	jobExecutor JobExecutor,
	updater JobStatusUpdater,
) {
	q.wg.Add(1)

	go func() {
		defer q.wg.Done()

		worker(ctx, id, q, jobExecutor, updater)
	}()
}

func (q *Queue) Close() {
	q.mu.Lock()

	if q.closed {
		q.mu.Unlock()
		return
	}

	q.closed = true
	q.mu.Unlock()

	q.signal()
}

func (q *Queue) Wait(ctx context.Context) bool {
	done := make(chan struct{})

	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true

	case <-ctx.Done():
		return false
	}
}
