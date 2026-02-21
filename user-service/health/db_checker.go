package health

import (
	"context"
	"database/sql"
)

// DBHealthChecker wraps *sql.DB to implement the HealthChecker interface.
type DBHealthChecker struct {
	db *sql.DB
}

// NewDBHealthChecker creates a new database health checker.
func NewDBHealthChecker(db *sql.DB) HealthChecker {
	return &DBHealthChecker{db: db}
}

// HealthCheck pings the database to verify connectivity.
func (h *DBHealthChecker) HealthCheck(ctx context.Context) error {
	return h.db.PingContext(ctx)
}
