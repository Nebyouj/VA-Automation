package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	workflowApp "github.com/nebyoudamtew/go-job-queue/interval/application/workflows"
)

type WorkflowHandler struct {
	service *workflowApp.Service
}

func NewWorkflowHandler(service *workflowApp.Service) *WorkflowHandler {
	return &WorkflowHandler{service: service}
}

func (h *WorkflowHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var request workflowApp.CreateRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	created, err := h.service.CreateWorkflow(r.Context(), request)
	if err != nil {
		message := err.Error()

		// Validation errors originate from the submitted workflow definition.
		if isWorkflowValidationError(message) {
			http.Error(w, message, http.StatusBadRequest)
			return
		}

		fmt.Printf("Failed to create workflow: %v\n", err)
		http.Error(w, "failed to create workflow", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(created); err != nil {
		fmt.Printf("Failed to encode workflow response: %v\n", err)
	}
}

func isWorkflowValidationError(message string) bool {
	prefixes := []string{
		"workflow name is required",
		"workflow must contain at least one job",
		"job at position ",
		"duplicate workflow job key ",
		"recurring job ",
		"max_attempts cannot be negative",
		"invalid job ",
		"invalid workflow cron_expression",
		"job ",
		"workflow contains a dependency cycle",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}

	return false
}

func (h *WorkflowHandler) GetWorkflows(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

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

	items, err := h.service.GetWorkflows(r.Context(), status, page, limit)
	if err != nil {
		fmt.Printf("Failed to list workflows: %v\n", err)
		http.Error(w, "failed to list workflows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		fmt.Printf("Failed to encode workflows response: %v\n", err)
	}
}

func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid workflow id", http.StatusBadRequest)
		return
	}

	item, err := h.service.GetWorkflow(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "workflow not found", http.StatusNotFound)
			return
		}

		fmt.Printf("Failed to get workflow %d: %v\n", id, err)
		http.Error(w, "failed to get workflow", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(item); err != nil {
		fmt.Printf("Failed to encode workflow response: %v\n", err)
	}
}
