package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dukedhal/taskflow/internal/model"
	"github.com/dukedhal/taskflow/internal/repository"
)

type ProjectService struct {
	projects *repository.ProjectRepository
}

func NewProjectService(projects *repository.ProjectRepository) *ProjectService {
	return &ProjectService{projects: projects}
}

type CreateProjectInput struct {
	Name        string  `json:"name"        validate:"required,min=1,max=255"`
	Description *string `json:"description"`
}

type UpdateProjectInput struct {
	Name        *string `json:"name"        validate:"omitempty,min=1,max=255"`
	Description *string `json:"description"`
}

func (s *ProjectService) Create(ctx context.Context, ownerID string, in CreateProjectInput) (*model.Project, error) {
	p := &model.Project{
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		OwnerID:     ownerID,
	}

	if err := s.projects.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return p, nil
}

func (s *ProjectService) List(ctx context.Context, userID string, page, limit int) ([]model.Project, int, error) {
	return s.projects.ListAccessibleByUser(ctx, userID, page, limit)
}

func (s *ProjectService) Get(ctx context.Context, id string) (*model.ProjectWithTasks, error) {
	return s.projects.FindByIDWithTasks(ctx, id)
}

// Update applies a partial update. Only the owner may update.
func (s *ProjectService) Update(ctx context.Context, id, callerID string, in UpdateProjectInput) (*model.Project, error) {
	p, err := s.projects.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if p.OwnerID != callerID {
		return nil, model.ErrForbidden
	}

	// Apply only the fields that were supplied.
	if in.Name != nil {
		p.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		p.Description = in.Description
	}

	if err := s.projects.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return p, nil
}

// Delete removes the project and cascades to all its tasks. Owner only.
func (s *ProjectService) Delete(ctx context.Context, id, callerID string) error {
	p, err := s.projects.FindByID(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return model.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find project: %w", err)
	}

	if p.OwnerID != callerID {
		return model.ErrForbidden
	}

	return s.projects.Delete(ctx, id)
}
