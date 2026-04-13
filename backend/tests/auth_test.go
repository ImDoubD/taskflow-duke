package tests

import (
	"net/http"
	"testing"
)

// TestAuth_RegisterAndLogin covers the full auth flow end-to-end:
// register → receive JWT → login → receive JWT → use JWT on protected route.
func TestAuth_RegisterAndLogin(t *testing.T) {
	ts := newTestServer(t)
	email := uniqueEmail()

	// 1. Register a new user.
	resp := ts.do(t, http.MethodPost, "/auth/register", map[string]any{
		"name":     "Jane Doe",
		"email":    email,
		"password": "securepassword",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", resp.StatusCode)
	}

	var registerBody map[string]any
	decode(t, resp, &registerBody)

	if _, ok := registerBody["token"]; !ok {
		t.Fatal("register response missing token")
	}
	if _, ok := registerBody["user"]; !ok {
		t.Fatal("register response missing user")
	}

	// 2. Login with the same credentials.
	resp = ts.do(t, http.MethodPost, "/auth/login", map[string]any{
		"email":    email,
		"password": "securepassword",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d", resp.StatusCode)
	}

	var loginBody map[string]any
	decode(t, resp, &loginBody)

	token, ok := loginBody["token"].(string)
	if !ok || token == "" {
		t.Fatal("login response missing or empty token")
	}
	ts.token = token

	// 3. Use the token to hit a protected route.
	resp = ts.do(t, http.MethodGet, "/projects", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get projects with valid token: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestAuth_Unauthenticated ensures protected routes return 401, not 403.
func TestAuth_Unauthenticated(t *testing.T) {
	ts := newTestServer(t) // no token set

	resp := ts.do(t, http.MethodGet, "/projects", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}
}

// TestAuth_DuplicateEmail ensures registering the same email twice returns 409.
func TestAuth_DuplicateEmail(t *testing.T) {
	ts := newTestServer(t)
	email := uniqueEmail()
	payload := map[string]any{
		"name":     "Alice",
		"email":    email,
		"password": "password123",
	}

	resp := ts.do(t, http.MethodPost, "/auth/register", payload)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first register: expected 201, got %d", resp.StatusCode)
	}

	resp = ts.do(t, http.MethodPost, "/auth/register", payload)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate register: expected 409, got %d", resp.StatusCode)
	}
}

// TestAuth_WrongPassword ensures wrong credentials return 401.
func TestAuth_WrongPassword(t *testing.T) {
	ts := newTestServer(t)
	email := uniqueEmail()

	resp := ts.do(t, http.MethodPost, "/auth/register", map[string]any{
		"name": "Bob", "email": email, "password": "correct_password",
	})
	resp.Body.Close()

	resp = ts.do(t, http.MethodPost, "/auth/login", map[string]any{
		"email": email, "password": "wrong_password",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password: expected 401, got %d", resp.StatusCode)
	}
}

// TestAuth_ValidationErrors ensures missing fields return 400 with field details.
func TestAuth_ValidationErrors(t *testing.T) {
	ts := newTestServer(t)

	resp := ts.do(t, http.MethodPost, "/auth/register", map[string]any{
		"name": "No Email",
		// email and password missing
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing fields: expected 400, got %d", resp.StatusCode)
	}

	var body map[string]any
	decode(t, resp, &body)
	if body["error"] != "validation failed" {
		t.Fatalf("expected 'validation failed', got %v", body["error"])
	}
	if _, ok := body["fields"]; !ok {
		t.Fatal("expected 'fields' in validation error response")
	}
}
