package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dukedhal/taskflow/internal/middleware"
	"github.com/dukedhal/taskflow/internal/model"
	"github.com/dukedhal/taskflow/internal/service"
)

type ProjectHandler struct {
	projects *service.ProjectService
	tasks    *service.TaskService
}

func NewProjectHandler(projects *service.ProjectService, tasks *service.TaskService) *ProjectHandler {
	return &ProjectHandler{projects: projects, tasks: tasks}
}

// List handles GET /projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	page, limit := parsePagination(r)

	projects, total, err := h.projects.List(r.Context(), userID, page, limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"projects": projects,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// Create handles POST /projects
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var in service.CreateProjectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if fields := validateStruct(in); fields != nil {
		ValidationError(w, fields)
		return
	}

	p, err := h.projects.Create(r.Context(), userID, in)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	JSON(w, http.StatusCreated, p)
}

// Get handles GET /projects/:id
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, err := h.projects.Get(r.Context(), id)
	if errors.Is(err, model.ErrNotFound) {
		Error(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	JSON(w, http.StatusOK, p)
}

// Update handles PATCH /projects/:id
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := middleware.UserIDFromContext(r.Context())

	var in service.UpdateProjectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if fields := validateStruct(in); fields != nil {
		ValidationError(w, fields)
		return
	}

	p, err := h.projects.Update(r.Context(), id, userID, in)
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

	JSON(w, http.StatusOK, p)
}

// Delete handles DELETE /projects/:id
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := middleware.UserIDFromContext(r.Context())

	err := h.projects.Delete(r.Context(), id, userID)
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

// Stats handles GET /projects/:id/stats (bonus)
func (h *ProjectHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	stats, err := h.tasks.Stats(r.Context(), id)
	if errors.Is(err, model.ErrNotFound) {
		Error(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	JSON(w, http.StatusOK, stats)
}
