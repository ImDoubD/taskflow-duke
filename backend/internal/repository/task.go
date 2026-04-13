package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/dukedhal/taskflow/internal/model"
)

// TaskFilters holds optional query-string filters for listing tasks.
type TaskFilters struct {
	Status     *string
	AssigneeID *string
}

type TaskRepository struct {
	db *sqlx.DB
}

func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

const taskSelectCols = `
	id, title, description, status, priority, project_id,
	assignee_id, created_by, due_date::TEXT AS due_date,
	created_at, updated_at`

func (r *TaskRepository) Create(ctx context.Context, t *model.Task) error {
	query := `
		INSERT INTO tasks (title, description, status, priority, project_id, assignee_id, created_by, due_date)
		VALUES ($1, $2, $3::task_status, $4::task_priority, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query,
		t.Title, t.Description, t.Status, t.Priority,
		t.ProjectID, t.AssigneeID, t.CreatedBy, t.DueDate,
	)
	if err := row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (r *TaskRepository) FindByID(ctx context.Context, id string) (*model.Task, error) {
	var t model.Task
	query := `SELECT ` + taskSelectCols + ` FROM tasks WHERE id = $1`
	err := r.db.GetContext(ctx, &t, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find task by id: %w", err)
	}
	return &t, nil
}

// List returns paginated tasks for a project with optional filters.
func (r *TaskRepository) List(ctx context.Context, projectID string, f TaskFilters, page, limit int) ([]model.Task, int, error) {
	// Build WHERE clause dynamically based on provided filters.
	where := "project_id = $1"
	args := []any{projectID}
	n := 2

	if f.Status != nil {
		where += fmt.Sprintf(" AND status = $%d::task_status", n)
		args = append(args, *f.Status)
		n++
	}
	if f.AssigneeID != nil {
		where += fmt.Sprintf(" AND assignee_id = $%d", n)
		args = append(args, *f.AssigneeID)
		n++
	}

	countQuery := "SELECT COUNT(*) FROM tasks WHERE " + where
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	listQuery := fmt.Sprintf(
		"SELECT %s FROM tasks WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		taskSelectCols, where, n, n+1,
	)
	args = append(args, limit, (page-1)*limit)

	var tasks []model.Task
	if err := r.db.SelectContext(ctx, &tasks, listQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	if tasks == nil {
		tasks = []model.Task{}
	}
	return tasks, total, nil
}

// Update applies only the supplied fields (partial update).
func (r *TaskRepository) Update(ctx context.Context, t *model.Task) error {
	query := `
		UPDATE tasks
		SET title       = $1,
		    description = $2,
		    status      = $3::task_status,
		    priority    = $4::task_priority,
		    assignee_id = $5,
		    due_date    = $6,
		    updated_at  = NOW()
		WHERE id = $7
		RETURNING updated_at`

	row := r.db.QueryRowContext(ctx, query,
		t.Title, t.Description, t.Status, t.Priority,
		t.AssigneeID, t.DueDate, t.ID,
	)
	if err := row.Scan(&t.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrNotFound
		}
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// StatsRow holds per-status task counts for the stats endpoint.
type StatsRow struct {
	Status string `db:"status"`
	Count  int    `db:"count"`
}

// AssigneeStatsRow holds per-assignee task counts.
type AssigneeStatsRow struct {
	UserID *string `db:"assignee_id" json:"user_id"`
	Name   string  `db:"name"        json:"name"`
	Count  int     `db:"count"       json:"count"`
}

func (r *TaskRepository) StatsByStatus(ctx context.Context, projectID string) ([]StatsRow, error) {
	var rows []StatsRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT status::TEXT, COUNT(*) AS count
		FROM tasks
		WHERE project_id = $1
		GROUP BY status`, projectID)
	if err != nil {
		return nil, fmt.Errorf("stats by status: %w", err)
	}
	return rows, nil
}

func (r *TaskRepository) StatsByAssignee(ctx context.Context, projectID string) ([]AssigneeStatsRow, error) {
	var rows []AssigneeStatsRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT t.assignee_id,
		       COALESCE(u.name, 'Unassigned') AS name,
		       COUNT(t.id) AS count
		FROM tasks t
		LEFT JOIN users u ON u.id = t.assignee_id
		WHERE t.project_id = $1
		GROUP BY t.assignee_id, u.name
		ORDER BY count DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("stats by assignee: %w", err)
	}
	return rows, nil
}
