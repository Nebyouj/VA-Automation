package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nebyoudamtew/go-job-queue/interval/application/jobs"
	"github.com/nebyoudamtew/go-job-queue/interval/domain/job"
)

type JobHandler struct {
	service *jobs.Service
}

func NewJobHandler(service *jobs.Service) *JobHandler {
	return &JobHandler{
		service: service,
	}
}

type CreateJobRequest struct {
	Type               string         `json:"type"`
	Priority           int            `json:"priority"`
	MaxAttempts        int            `json:"max_attempts"`
	ScheduledAt        *time.Time     `json:"scheduled_at"`
	RepeatEverySeconds int            `json:"repeat_every_seconds,omitempty"`
	CronExpression     string         `json:"cron_expression,omitempty"`
	Payload            map[string]any `json:"payload"`
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var request CreateJobRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	request.Type = strings.TrimSpace(request.Type)

	if request.Type == "" {
		http.Error(w, "type is required", http.StatusBadRequest)
		return
	}

	createdJob, err := h.service.CreateJob(
		r.Context(),
		request.Type,
		request.Priority,
		request.MaxAttempts,
		request.ScheduledAt,
		request.RepeatEverySeconds,
		request.CronExpression,
		request.Payload,
	)

	if err != nil {
		fmt.Printf("Failed to create job: %v\n", err)

		if errors.Is(err, job.ErrInvalidJob) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to create job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(createdJob)

}

func (h *JobHandler) GetJobs(
	w http.ResponseWriter,
	r *http.Request,
) {
	status := r.URL.Query().Get("status")
	jobType := r.URL.Query().Get("type")

	page := 1
	limit := 10

	if value := r.URL.Query().Get("page"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			http.Error(w, "invalid page", http.StatusBadRequest)
			return
		}
		page = parsed
	}

	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	jobs, err := h.service.GetJobs(r.Context(), status, jobType, page, limit)

	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(jobs)
}

func (h *JobHandler) GetJob(
	w http.ResponseWriter,
	r *http.Request,
) {
	idString := r.PathValue("id")

	id, err := strconv.Atoi(idString)
	if err != nil {
		http.Error(
			w,
			"invalid job id",
			http.StatusBadRequest,
		)
		return
	}

	job, err := h.service.GetJob(r.Context(), id)

	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusNotFound,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(job)
}

func (h *JobHandler) CancelJob(
	w http.ResponseWriter,
	r *http.Request,
) {
	idString := r.PathValue("id")

	id, err := strconv.Atoi(idString)
	if err != nil {
		http.Error(
			w,
			"invalid job id",
			http.StatusBadRequest,
		)

		return
	}

	err = h.service.CancelJob(r.Context(), id)

	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

		return
	}

	job, err := h.service.GetJob(r.Context(), id)

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(job)
}

func (h *JobHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		http.Error(
			w,
			"failed to get job statistics",
			http.StatusInternalServerError,
		)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
	}
}
