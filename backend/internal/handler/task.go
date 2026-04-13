package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/dukedhal/taskflow/internal/middleware"
	"github.com/dukedhal/taskflow/internal/model"
	"github.com/dukedhal/taskflow/internal/repository"
	"github.com/dukedhal/taskflow/internal/service"
)

type TaskHandler struct {
	tasks *service.TaskService
}

func NewTaskHandler(tasks *service.TaskService) *TaskHandler {
	return &TaskHandler{tasks: tasks}
}

// List handles GET /projects/:id/tasks
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	page, limit := parsePagination(r)

	filters := repository.TaskFilters{}
	if s := r.URL.Query().Get("status"); s != "" {
		filters.Status = &s
	}
	if a := r.URL.Query().Get("assignee"); a != "" {
		filters.AssigneeID = &a
	}

	tasks, total, err := h.tasks.List(r.Context(), projectID, filters, page, limit)
	if errors.Is(err, model.ErrNotFound) {
		Error(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// Create handles POST /projects/:id/tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	userID := middleware.UserIDFromContext(r.Context())

	var in service.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if fields := validateStruct(in); fields != nil {
		ValidationError(w, fields)
		return
	}

	t, err := h.tasks.Create(r.Context(), projectID, userID, in)
	if errors.Is(err, model.ErrNotFound) {
		Error(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	JSON(w, http.StatusCreated, t)
}

// Update handles PATCH /tasks/:id
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var in service.UpdateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if fields := validateStruct(in); fields != nil {
		ValidationError(w, fields)
		return
	}

	t, err := h.tasks.Update(r.Context(), id, in)
	if errors.Is(err, model.ErrNotFound) {
		Error(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	JSON(w, http.StatusOK, t)
}

// Delete handles DELETE /tasks/:id
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := middleware.UserIDFromContext(r.Context())

	err := h.tasks.Delete(r.Context(), id, userID)
	if errors.Is(err, model.ErrNotFound) {
		Error(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, model.ErrForbidden) {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parsePagination extracts page/limit from query params
func parsePagination(r *http.Request) (page, limit int) {
	// sensible defaults
	page = 1
	limit = 20

	if p := r.URL.Query().Get("page"); p != "" {
		if v := parseInt(p); v > 0 {
			page = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v := parseInt(l); v > 0 && v <= 100 {
			limit = v
		}
	}
	return
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
