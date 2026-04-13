package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dukedhal/taskflow/internal/config"
	"github.com/dukedhal/taskflow/internal/database"
	"github.com/dukedhal/taskflow/internal/handler"
	"github.com/dukedhal/taskflow/internal/repository"
	"github.com/dukedhal/taskflow/internal/service"
	"github.com/dukedhal/taskflow/migrations"
)

// main() steps:
//1. Set up logging
//2. Load config from environment variables
//3. Connect to the database
//4. Run migrations
//5. Insert seed data (if RUN_SEED=true)
//6. Create repositories
//7. Create services
//8. Create router (with handlers)
//9. Start HTTP server
//10. Wait for shutdown signal
//11. Gracefully shut down

func main() {
	// JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := database.NewPool(cfg.DB)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("connected to database")

	if err := database.RunMigrations(cfg.DB.MigrateURL(), migrations.FS); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	if cfg.RunSeed {
		if err := database.RunSeed(db); err != nil {
			slog.Error("failed to run seed", "error", err)
			os.Exit(1)
		}
		slog.Info("seed data applied")
	}

	// Dependency injection — explicit wiring.
	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	authSvc := service.NewAuthService(userRepo, cfg.JWT.Secret)
	projectSvc := service.NewProjectService(projectRepo)
	taskSvc := service.NewTaskService(taskRepo, projectRepo)

	router := handler.NewRouter(authSvc, projectSvc, taskSvc)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGTERM or SIGINT.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("server starting", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	slog.Info("server stopped")
}
