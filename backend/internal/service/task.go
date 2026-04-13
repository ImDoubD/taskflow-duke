package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dukedhal/taskflow/internal/model"
	"github.com/dukedhal/taskflow/internal/repository"
)

type TaskService struct {
	tasks    *repository.TaskRepository
	projects *repository.ProjectRepository
}

func NewTaskService(tasks *repository.TaskRepository, projects *repository.ProjectRepository) *TaskService {
	return &TaskService{tasks: tasks, projects: projects}
}

type CreateTaskInput struct {
	Title       string       `json:"title"       validate:"required,min=1,max=255"`
	Description *string      `json:"description"`
	Status      string       `json:"status"      validate:"omitempty,oneof=todo in_progress done"`
	Priority    string       `json:"priority"    validate:"omitempty,oneof=low medium high"`
	AssigneeID  *string      `json:"assignee_id"`
	DueDate     *string      `json:"due_date"    validate:"omitempty,datetime=2006-01-02"`
}

type UpdateTaskInput struct {
	Title       *string `json:"title"       validate:"omitempty,min=1,max=255"`
	Description *string `json:"description"`
	Status      *string `json:"status"      validate:"omitempty,oneof=todo in_progress done"`
	Priority    *string `json:"priority"    validate:"omitempty,oneof=low medium high"`
	AssigneeID  *string `json:"assignee_id"`
	DueDate     *string `json:"due_date"    validate:"omitempty,datetime=2006-01-02"`
}

// StatsResponse is the payload returned by GET /projects/:id/stats.
type StatsResponse struct {
	ByStatus   map[string]int               `json:"by_status"`
	ByAssignee []repository.AssigneeStatsRow `json:"by_assignee"`
	Total      int                          `json:"total"`
}

func (s *TaskService) Create(ctx context.Context, projectID, createdBy string, in CreateTaskInput) (*model.Task, error) {
	// Verify the project exists before creating a task in it.
	if _, err := s.projects.FindByID(ctx, projectID); err != nil {
		return nil, err
	}

	status := model.TaskStatus(in.Status)
	if status == "" {
		status = model.StatusTodo
	}
	priority := model.TaskPriority(in.Priority)
	if priority == "" {
		priority = model.PriorityMedium
	}

	t := &model.Task{
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		Status:      status,
		Priority:    priority,
		ProjectID:   projectID,
		AssigneeID:  in.AssigneeID,
		CreatedBy:   createdBy,
		DueDate:     in.DueDate,
	}

	if err := s.tasks.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return t, nil
}

func (s *TaskService) List(ctx context.Context, projectID string, filters repository.TaskFilters, page, limit int) ([]model.Task, int, error) {
	// Verify the project exists.
	if _, err := s.projects.FindByID(ctx, projectID); err != nil {
		return nil, 0, err
	}
	return s.tasks.List(ctx, projectID, filters, page, limit)
}

// Update applies a partial update. Any authenticated user may update any task
// (the assignment only restricts delete, not update).
func (s *TaskService) Update(ctx context.Context, id string, in UpdateTaskInput) (*model.Task, error) {
	t, err := s.tasks.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Title != nil {
		t.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		t.Description = in.Description
	}
	if in.Status != nil {
		t.Status = model.TaskStatus(*in.Status)
	}
	if in.Priority != nil {
		t.Priority = model.TaskPriority(*in.Priority)
	}
	if in.AssigneeID != nil {
		t.AssigneeID = in.AssigneeID
	}
	if in.DueDate != nil {
		t.DueDate = in.DueDate
	}

	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return t, nil
}

// Delete removes a task. Only the project owner or task creator may do this.
func (s *TaskService) Delete(ctx context.Context, id, callerID string) error {
	t, err := s.tasks.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Load the project to check ownership.
	p, err := s.projects.FindByID(ctx, t.ProjectID)
	if err != nil {
		return fmt.Errorf("find parent project: %w", err)
	}

	if t.CreatedBy != callerID && p.OwnerID != callerID {
		return model.ErrForbidden
	}

	return s.tasks.Delete(ctx, id)
}

// Stats returns task counts grouped by status and by assignee for a project.
func (s *TaskService) Stats(ctx context.Context, projectID string) (*StatsResponse, error) {
	if _, err := s.projects.FindByID(ctx, projectID); err != nil {
		return nil, err
	}

	statusRows, err := s.tasks.StatsByStatus(ctx, projectID)
	if err != nil {
		return nil, err
	}

	assigneeRows, err := s.tasks.StatsByAssignee(ctx, projectID)
	if err != nil {
		return nil, err
	}

	byStatus := map[string]int{
		"todo":        0,
		"in_progress": 0,
		"done":        0,
	}
	total := 0
	for _, r := range statusRows {
		byStatus[r.Status] = r.Count
		total += r.Count
	}

	if assigneeRows == nil {
		assigneeRows = []repository.AssigneeStatsRow{}
	}

	return &StatsResponse{
		ByStatus:   byStatus,
		ByAssignee: assigneeRows,
		Total:      total,
	}, nil
}
