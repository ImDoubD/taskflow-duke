package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dukedhal/taskflow/internal/middleware"
	"github.com/dukedhal/taskflow/internal/service"
)

// NewRouter wires all routes, middleware, and handlers into a single http.Handler.
func NewRouter(
	authSvc *service.AuthService,
	projectSvc *service.ProjectService,
	taskSvc *service.TaskService,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())

	authH    := NewAuthHandler(authSvc)
	projectH := NewProjectHandler(projectSvc, taskSvc)
	taskH    := NewTaskHandler(taskSvc)

	// Public — no auth required
	r.Post("/auth/register", authH.Register)
	r.Post("/auth/login", authH.Login)

	// Protected — JWT required
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(authSvc))

		// Projects
		r.Get("/projects", projectH.List)
		r.Post("/projects", projectH.Create)
		r.Get("/projects/{id}", projectH.Get)
		r.Patch("/projects/{id}", projectH.Update)
		r.Delete("/projects/{id}", projectH.Delete)
		r.Get("/projects/{id}/stats", projectH.Stats)

		// Tasks
		r.Get("/projects/{id}/tasks", taskH.List)
		r.Post("/projects/{id}/tasks", taskH.Create)
		r.Patch("/tasks/{id}", taskH.Update)
		r.Delete("/tasks/{id}", taskH.Delete)
	})

	return r
}
