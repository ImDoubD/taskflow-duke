package database

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver with database/sql
	"github.com/jmoiron/sqlx"

	"github.com/dukedhal/taskflow/internal/config"
)

// NewPool opens a connection pool to PostgreSQL and verifies connectivity.
func NewPool(cfg config.DBConfig) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// max allowed connection is 25, can be tuned as per workload.
	db.SetMaxOpenConns(25)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
