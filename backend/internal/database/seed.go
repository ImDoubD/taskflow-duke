package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

const (
	seedUserID    = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	seedProjectID = "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22"
)

// RunSeed inserts a known test user, one project, and three tasks.
// All inserts use ON CONFLICT DO NOTHING so repeated calls are safe.
func RunSeed(db *sqlx.DB) error {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck — rollback on error path only

	if _, err := tx.Exec(`
		INSERT INTO users (id, name, email, password)
		VALUES ($1, 'Test User', 'test@example.com', $2)
		ON CONFLICT (email) DO NOTHING`,
		seedUserID, string(hash),
	); err != nil {
		return fmt.Errorf("seed user: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO projects (id, name, description, owner_id)
		VALUES ($1, 'Demo Project', 'A sample project for testing', $2)
		ON CONFLICT (id) DO NOTHING`,
		seedProjectID, seedUserID,
	); err != nil {
		return fmt.Errorf("seed project: %w", err)
	}

	tasks := []struct {
		id, title, description, status, priority, dueDate string
	}{
		{"c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a33", "Set up project structure", "Initialize the repository and tooling", "done", "high", "2026-04-10"},
		{"d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a44", "Implement authentication", "Register and login endpoints with JWT", "in_progress", "high", "2026-04-20"},
		{"e0eebc99-9c0b-4ef8-bb6d-6bb9bd380a55", "Write API documentation", "Document all endpoints with examples", "todo", "medium", "2026-04-30"},
	}

	for _, t := range tasks {
		if _, err := tx.Exec(`
			INSERT INTO tasks (id, title, description, status, priority, project_id, created_by, due_date)
			VALUES ($1, $2, $3, $4::task_status, $5::task_priority, $6, $7, $8)
			ON CONFLICT (id) DO NOTHING`,
			t.id, t.title, t.description, t.status, t.priority,
			seedProjectID, seedUserID, t.dueDate,
		); err != nil {
			return fmt.Errorf("seed task %q: %w", t.title, err)
		}
	}

	return tx.Commit()
}
