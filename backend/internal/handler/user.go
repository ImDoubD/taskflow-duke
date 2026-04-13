package handler

import (
	"net/http"

	"github.com/dukedhal/taskflow/internal/repository"
)

type UserHandler struct {
	users *repository.UserRepository
}

func NewUserHandler(users *repository.UserRepository) *UserHandler {
	return &UserHandler{users: users}
}

// List handles GET /users — returns all registered users.
// Useful for populating assignee pickers in the frontend.
// Password is never included (json:"-" on the model field).
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.ListAll(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"users": users,
	})
}
