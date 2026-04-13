package tests

import (
	"net/http"
	"testing"
)

// loginUser registers a new user, logs in, sets the token, and returns the user ID.
func loginUser(t *testing.T, ts *testServer) string {
	t.Helper()
	email := uniqueEmail()

	resp := ts.do(t, http.MethodPost, "/auth/register", map[string]any{
		"name": "Test User", "email": email, "password": "password123",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", resp.StatusCode)
	}

	var body map[string]any
	decode(t, resp, &body)

	ts.token = body["token"].(string)

	user := body["user"].(map[string]any)
	return user["id"].(string)
}

// TestProject_CRUD tests the full project lifecycle.
func TestProject_CRUD(t *testing.T) {
	ts := newTestServer(t)
	loginUser(t, ts)

	// Create
	resp := ts.do(t, http.MethodPost, "/projects", map[string]any{
		"name":        "My Project",
		"description": "A test project",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project: expected 201, got %d", resp.StatusCode)
	}

	var project map[string]any
	decode(t, resp, &project)
	projectID := project["id"].(string)

	// Get
	resp = ts.do(t, http.MethodGet, "/projects/"+projectID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get project: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Update
	resp = ts.do(t, http.MethodPatch, "/projects/"+projectID, map[string]any{
		"name": "Renamed Project",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update project: expected 200, got %d", resp.StatusCode)
	}
	var updated map[string]any
	decode(t, resp, &updated)
	if updated["name"] != "Renamed Project" {
		t.Fatalf("update: expected name 'Renamed Project', got %v", updated["name"])
	}

	// Delete
	resp = ts.do(t, http.MethodDelete, "/projects/"+projectID, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete project: expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Confirm deleted
	resp = ts.do(t, http.MethodGet, "/projects/"+projectID, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("after delete: expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestProject_AuthorizationEnforced ensures non-owners get 403 on update/delete.
func TestProject_AuthorizationEnforced(t *testing.T) {
	ts := newTestServer(t)
	loginUser(t, ts)

	// Owner creates a project.
	resp := ts.do(t, http.MethodPost, "/projects", map[string]any{"name": "Owner's Project"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}
	var p map[string]any
	decode(t, resp, &p)
	projectID := p["id"].(string)

	// Switch to a different user.
	loginUser(t, ts)

	// Non-owner update → 403
	resp = ts.do(t, http.MethodPatch, "/projects/"+projectID, map[string]any{"name": "Hijacked"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-owner update: expected 403, got %d", resp.StatusCode)
	}

	// Non-owner delete → 403
	resp = ts.do(t, http.MethodDelete, "/projects/"+projectID, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-owner delete: expected 403, got %d", resp.StatusCode)
	}
}

// TestProject_List verifies pagination fields are present.
func TestProject_List(t *testing.T) {
	ts := newTestServer(t)
	loginUser(t, ts)

	resp := ts.do(t, http.MethodGet, "/projects?page=1&limit=10", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	decode(t, resp, &body)

	for _, field := range []string{"projects", "total", "page", "limit"} {
		if _, ok := body[field]; !ok {
			t.Fatalf("list response missing field %q", field)
		}
	}
}
