package database

import (
	"context"
	"database/sql"
)

type HealthChecker interface {
	PingContext(ctx context.Context) error
}

func ReadinessProbe(checker HealthChecker) func(context.Context) error {
	if checker == nil {
		return nil
	}
	return func(ctx context.Context) error {
		return checker.PingContext(ctx)
	}
}

func NewSQLite(dbURL string) (*sql.DB, error) {
	return sql.Open("sqlite3", dbURL)
}
