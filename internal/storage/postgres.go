// Package storage manages the PostgreSQL connection pool used as the
// system of record for users, food logs, and food items.

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

// DB wraps a *sql.DB connection pool to PostgreSQL.
type DB struct {
	*sql.DB
}

// NewPostgres opens a connection pool and verifies connectivity with a ping.
func NewPostgres(ctx context.Context, cfg config.PostgresConfig) (*DB, error) {
	sqlDB, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening postgres connection: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}

	return &DB{sqlDB}, nil
}

// HealthCheck verifies the database is reachable within the given context.
func (d *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return d.PingContext(ctx)
}