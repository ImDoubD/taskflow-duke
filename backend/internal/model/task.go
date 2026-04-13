package model

import "time"

type TaskStatus string
type TaskPriority string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

// Task belongs to a Project and can be assigned to a User.
type Task struct {
	ID          string       `db:"id"          json:"id"`
	Title       string       `db:"title"       json:"title"`
	Description *string      `db:"description" json:"description"`
	Status      TaskStatus   `db:"status"      json:"status"`
	Priority    TaskPriority `db:"priority"    json:"priority"`
	ProjectID   string       `db:"project_id"  json:"project_id"`
	AssigneeID  *string      `db:"assignee_id" json:"assignee_id"`
	CreatedBy   string       `db:"created_by"  json:"created_by"`
	// DueDate is stored as a DATE in Postgres and returned as "YYYY-MM-DD".
	// We cast it to TEXT in every SELECT to avoid time-zone drift.
	DueDate   *string   `db:"due_date"   json:"due_date"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
