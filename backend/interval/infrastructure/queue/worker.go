package queue

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
	"github.com/robfig/cron/v3"
)

type JobStatusUpdater interface {
	MarkRunning(id int) error
	MarkCompleted(id int) error
	MarkFailed(id int) error
	IncrementAttempt(id int) (job.Job, error)
	RescheduleJob(id int, scheduledAt time.Time) error
	SaveResult(id int, result map[string]any) error
}

func worker(
	ctx context.Context,
	id int,
	q *Queue,
	jobExecutor JobExecutor,
	updater JobStatusUpdater,
) {
	fmt.Printf("Worker %d started\n", id)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d shutting down\n", id)
			return
		default:
		}

		queueJob, ok := q.Dequeue(ctx)
		if !ok {
			fmt.Printf("Worker %d shutting down\n", id)
			return
		}

		j := queueJob.Job
		jobCtx := queueJob.Context

		if jobCtx.Err() != nil {
			fmt.Printf("Worker %d skipped cancelled job %d\n", id, j.ID)
			continue
		}

		fmt.Printf("Worker %d picked up job %d\n", id, j.ID)

		for {
			if ctx.Err() != nil {
				return
			}

			if jobCtx.Err() != nil {
				fmt.Printf("Worker %d cancelled job %d\n", id, j.ID)
				break
			}

			updatedJob, err := updater.IncrementAttempt(j.ID)
			if err != nil {
				fmt.Printf(
					"Worker %d could not increment attempt for job %d: %v\n",
					id, j.ID, err,
				)
				break
			}

			if err := updater.MarkRunning(j.ID); err != nil {
				fmt.Printf(
					"Worker %d failed to mark job %d as running: %v\n",
					id, j.ID, err,
				)
				break
			}

			var executionResult map[string]any

			if resultExecutor, ok := jobExecutor.(interface {
				ExecuteWithResult(context.Context, job.Job) (map[string]any, error)
			}); ok {
				executionResult, err = resultExecutor.ExecuteWithResult(jobCtx, updatedJob)
			} else {
				err = jobExecutor.Execute(jobCtx, updatedJob)
			}

			if err == nil && executionResult != nil {
				err = updater.SaveResult(j.ID, executionResult)
			}

			if err == nil {
				if isRecurring(updatedJob) {
					nextRun, err := nextScheduledTime(updatedJob, time.Now())
					if err != nil {
						fmt.Printf(
							"Worker %d could not calculate next run for job %d: %v\n",
							id, j.ID, err,
						)

						if markErr := updater.MarkFailed(j.ID); markErr != nil {
							fmt.Printf(
								"Worker %d failed to mark job %d as failed: %v\n",
								id, j.ID, markErr,
							)
						}
						break
					}

					if err := updater.RescheduleJob(j.ID, nextRun); err != nil {
						fmt.Printf(
							"Worker %d completed job %d but could not persist next run: %v\n",
							id, j.ID, err,
						)
						break
					}

					updatedJob.Status = job.StatusPending
					updatedJob.ScheduledAt = &nextRun
					updatedJob.Attempts = 0
					updatedJob.StartedAt = nil
					updatedJob.EndedAt = nil

					if jobCtx.Err() != nil || ctx.Err() != nil {
						fmt.Printf(
							"Worker %d did not re-enqueue recurring job %d because its context was cancelled\n",
							id, j.ID,
						)
						break
					}

					if err := q.Enqueue(
						ctx,
						QueueJob{
							Job:     updatedJob,
							Context: jobCtx,
						},
					); err != nil {
						fmt.Printf(
							"Worker %d persisted next run for job %d but could not enqueue it: %v\n",
							id, j.ID, err,
						)
						break
					}

					fmt.Printf(
						"Worker %d completed recurring job %d; next run at %s\n",
						id, j.ID, nextRun.Format(time.RFC3339),
					)
					break
				}

				if err := updater.MarkCompleted(j.ID); err != nil {
					fmt.Printf(
						"Worker %d executed job %d successfully but could not mark it completed: %v\n",
						id, j.ID, err,
					)
					break
				}

				fmt.Printf(
					"Worker %d completed job %d on attempt %d\n",
					id, j.ID, updatedJob.Attempts,
				)
				break
			}

			if jobCtx.Err() != nil {
				fmt.Printf("Worker %d cancelled job %d\n", id, j.ID)
				break
			}

			fmt.Printf(
				"Worker %d failed job %d on attempt %d: %v\n",
				id, j.ID, updatedJob.Attempts, err,
			)

			if updatedJob.Attempts >= updatedJob.MaxAttempts {
				if markErr := updater.MarkFailed(j.ID); markErr != nil {
					fmt.Printf(
						"Worker %d failed to mark job %d as failed: %v\n",
						id, j.ID, markErr,
					)
				} else {
					fmt.Printf(
						"Job %d permanently failed after %d attempts\n",
						j.ID, updatedJob.Attempts,
					)
				}
				break
			}

			delay := time.Duration(1<<uint(updatedJob.Attempts-1)) * time.Second

			fmt.Printf("Retrying job %d in %v...\n", j.ID, delay)

			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				// Continue retrying.
			case <-jobCtx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				fmt.Printf("Worker %d cancelled job %d during retry delay\n", id, j.ID)
				break
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			}

			if jobCtx.Err() != nil || ctx.Err() != nil {
				break
			}
		}
	}
}

func isRecurring(j job.Job) bool {
	return j.RepeatEverySeconds > 0 ||
		strings.TrimSpace(j.CronExpression) != ""
}

func nextScheduledTime(j job.Job, now time.Time) (time.Time, error) {
	if strings.TrimSpace(j.CronExpression) != "" {
		schedule, err := cron.ParseStandard(j.CronExpression)
		if err != nil {
			return time.Time{}, fmt.Errorf(
				"parse cron expression for job %d: %w",
				j.ID, err,
			)
		}

		return schedule.Next(now), nil
	}

	if j.RepeatEverySeconds > 0 {
		return now.Add(
			time.Duration(j.RepeatEverySeconds) * time.Second,
		), nil
	}

	return time.Time{}, fmt.Errorf(
		"job %d has no recurring schedule",
		j.ID,
	)
}
