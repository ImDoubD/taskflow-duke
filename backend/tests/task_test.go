package tests

import (
	"net/http"
	"testing"
)

// createProject is a test helper that creates a project and returns its ID.
func createProject(t *testing.T, ts *testServer, name string) string {
	t.Helper()
	resp := ts.do(t, http.MethodPost, "/projects", map[string]any{"name": name})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project: expected 201, got %d", resp.StatusCode)
	}
	var p map[string]any
	decode(t, resp, &p)
	return p["id"].(string)
}

// TestTask_Lifecycle tests create → list → update status → delete.
func TestTask_Lifecycle(t *testing.T) {
	ts := newTestServer(t)
	loginUser(t, ts)
	projectID := createProject(t, ts, "Task Test Project")

	// Create task
	resp := ts.do(t, http.MethodPost, "/projects/"+projectID+"/tasks", map[string]any{
		"title":    "Write tests",
		"priority": "high",
		"due_date": "2026-12-31",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create task: expected 201, got %d", resp.StatusCode)
	}
	var task map[string]any
	decode(t, resp, &task)

	taskID := task["id"].(string)
	if task["status"] != "todo" {
		t.Fatalf("default status: expected 'todo', got %v", task["status"])
	}
	if task["priority"] != "high" {
		t.Fatalf("priority: expected 'high', got %v", task["priority"])
	}

	// List tasks — should contain our task
	resp = ts.do(t, http.MethodGet, "/projects/"+projectID+"/tasks", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list tasks: expected 200, got %d", resp.StatusCode)
	}
	var listBody map[string]any
	decode(t, resp, &listBody)
	tasks := listBody["tasks"].([]any)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	// Update status to in_progress
	resp = ts.do(t, http.MethodPatch, "/tasks/"+taskID, map[string]any{
		"status": "in_progress",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update task: expected 200, got %d", resp.StatusCode)
	}
	var updated map[string]any
	decode(t, resp, &updated)
	if updated["status"] != "in_progress" {
		t.Fatalf("updated status: expected 'in_progress', got %v", updated["status"])
	}

	// Filter by status — should find our task
	resp = ts.do(t, http.MethodGet, "/projects/"+projectID+"/tasks?status=in_progress", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("filter tasks: expected 200, got %d", resp.StatusCode)
	}
	var filtered map[string]any
	decode(t, resp, &filtered)
	filteredTasks := filtered["tasks"].([]any)
	if len(filteredTasks) != 1 {
		t.Fatalf("status filter: expected 1 task, got %d", len(filteredTasks))
	}

	// Delete task
	resp = ts.do(t, http.MethodDelete, "/tasks/"+taskID, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete task: expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Confirm task deleted
	resp = ts.do(t, http.MethodGet, "/projects/"+projectID+"/tasks", nil)
	var afterDelete map[string]any
	decode(t, resp, &afterDelete)
	remaining := afterDelete["tasks"].([]any)
	if len(remaining) != 0 {
		t.Fatalf("after delete: expected 0 tasks, got %d", len(remaining))
	}
}

// TestTask_Stats verifies the bonus stats endpoint.
func TestTask_Stats(t *testing.T) {
	ts := newTestServer(t)
	loginUser(t, ts)
	projectID := createProject(t, ts, "Stats Project")

	// Create tasks with different statuses
	for _, payload := range []map[string]any{
		{"title": "Task A", "status": "todo"},
		{"title": "Task B", "status": "in_progress"},
		{"title": "Task C", "status": "done"},
	} {
		resp := ts.do(t, http.MethodPost, "/projects/"+projectID+"/tasks", payload)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create task: expected 201, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	resp := ts.do(t, http.MethodGet, "/projects/"+projectID+"/stats", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats: expected 200, got %d", resp.StatusCode)
	}

	var stats map[string]any
	decode(t, resp, &stats)

	if stats["total"].(float64) != 3 {
		t.Fatalf("stats total: expected 3, got %v", stats["total"])
	}

	byStatus, ok := stats["by_status"].(map[string]any)
	if !ok {
		t.Fatal("stats missing by_status")
	}
	for _, status := range []string{"todo", "in_progress", "done"} {
		if byStatus[status].(float64) != 1 {
			t.Fatalf("by_status[%s]: expected 1, got %v", status, byStatus[status])
		}
	}
}

// TestTask_DeleteAuthorization ensures non-creator/non-owner gets 403.
func TestTask_DeleteAuthorization(t *testing.T) {
	ts := newTestServer(t)
	loginUser(t, ts)
	projectID := createProject(t, ts, "Auth Project")

	// Task creator creates a task
	resp := ts.do(t, http.MethodPost, "/projects/"+projectID+"/tasks", map[string]any{
		"title": "Owned task",
	})
	var task map[string]any
	decode(t, resp, &task)
	taskID := task["id"].(string)

	// A different user tries to delete it
	loginUser(t, ts) // overwrites ts.token with a new user's token

	resp = ts.do(t, http.MethodDelete, "/tasks/"+taskID, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unauthorized delete: expected 403, got %d", resp.StatusCode)
	}
}
