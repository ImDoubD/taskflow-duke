package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/dukedhal/taskflow/internal/model"
)

type ProjectRepository struct {
	db *sqlx.DB
}

func NewProjectRepository(db *sqlx.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, p *model.Project) error {
	query := `
		INSERT INTO projects (name, description, owner_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	row := r.db.QueryRowContext(ctx, query, p.Name, p.Description, p.OwnerID)
	if err := row.Scan(&p.ID, &p.CreatedAt); err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

func (r *ProjectRepository) FindByID(ctx context.Context, id string) (*model.Project, error) {
	var p model.Project
	err := r.db.GetContext(ctx, &p, `SELECT * FROM projects WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find project by id: %w", err)
	}
	return &p, nil
}

// ListAccessibleByUser returns projects the user owns or has tasks assigned in,
// with pagination. Returns the projects, total count, and any error.
func (r *ProjectRepository) ListAccessibleByUser(ctx context.Context, userID string, page, limit int) ([]model.Project, int, error) {
	offset := (page - 1) * limit

	const countQuery = `
		SELECT COUNT(DISTINCT p.id)
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.owner_id = $1 OR t.assignee_id = $1`

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, userID); err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}

	const listQuery = `
		SELECT DISTINCT p.*
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.owner_id = $1 OR t.assignee_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3`

	var projects []model.Project
	if err := r.db.SelectContext(ctx, &projects, listQuery, userID, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}

	if projects == nil {
		projects = []model.Project{}
	}
	return projects, total, nil
}

// FindByIDWithTasks returns a project and all its tasks in two queries.
func (r *ProjectRepository) FindByIDWithTasks(ctx context.Context, id string) (*model.ProjectWithTasks, error) {
	p, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	const taskQuery = `
		SELECT id, title, description, status, priority, project_id,
		       assignee_id, created_by, due_date::TEXT AS due_date,
		       created_at, updated_at
		FROM tasks
		WHERE project_id = $1
		ORDER BY created_at DESC`

	var tasks []model.Task
	if err := r.db.SelectContext(ctx, &tasks, taskQuery, id); err != nil {
		return nil, fmt.Errorf("list tasks for project: %w", err)
	}
	if tasks == nil {
		tasks = []model.Task{}
	}

	return &model.ProjectWithTasks{Project: *p, Tasks: tasks}, nil
}

func (r *ProjectRepository) Update(ctx context.Context, p *model.Project) error {
	query := `
		UPDATE projects
		SET name = $1, description = $2
		WHERE id = $3
		RETURNING created_at`

	row := r.db.QueryRowContext(ctx, query, p.Name, p.Description, p.ID)
	if err := row.Scan(&p.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrNotFound
		}
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}
