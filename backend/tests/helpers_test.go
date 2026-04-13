package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/dukedhal/taskflow/internal/config"
	"github.com/dukedhal/taskflow/internal/database"
	"github.com/dukedhal/taskflow/internal/handler"
	"github.com/dukedhal/taskflow/internal/repository"
	"github.com/dukedhal/taskflow/internal/service"
	"github.com/dukedhal/taskflow/migrations"
)

const testJWTSecret = "test_secret_for_integration_tests"

// testServer holds a running httptest.Server backed by a real Postgres container.
type testServer struct {
	server *httptest.Server
	token  string // populated after calling registerAndLogin
}

// newTestServer spins up a Postgres container, runs migrations, and builds the
// full handler stack backed by a real database. The container is cleaned up
// when the test ends.
func newTestServer(t *testing.T) *testServer {
	t.Helper()
	ctx := context.Background()

	// Start a real Postgres container.
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	host, err := pg.Host(ctx)
	if err != nil {
		t.Fatalf("get container host: %v", err)
	}
	mappedPort, err := pg.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("get mapped port: %v", err)
	}

	dbCfg := config.DBConfig{
		Host:     host,
		Port:     mappedPort.Port(),
		User:     "test",
		Password: "test",
		Name:     "testdb",
		SSLMode:  "disable",
	}

	db, err := database.NewPool(dbCfg)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.RunMigrations(dbCfg.MigrateURL(), migrations.FS); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	userRepo    := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	taskRepo    := repository.NewTaskRepository(db)

	authSvc    := service.NewAuthService(userRepo, testJWTSecret)
	projectSvc := service.NewProjectService(projectRepo)
	taskSvc    := service.NewTaskService(taskRepo, projectRepo)

	router := handler.NewRouter(authSvc, projectSvc, taskSvc, userRepo)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testServer{server: srv}
}

// do sends an HTTP request and returns the response.
func (ts *testServer) do(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, ts.server.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if ts.token != "" {
		req.Header.Set("Authorization", "Bearer "+ts.token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("execute request: %v", err)
	}
	return resp
}

// decode reads and decodes the response body into v.
func decode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// uniqueEmail generates a unique email for each test run.
func uniqueEmail() string {
	return fmt.Sprintf("user_%d@example.com", time.Now().UnixNano())
}
