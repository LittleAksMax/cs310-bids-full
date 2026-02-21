package health

import "context"

// HealthChecker defines a common interface for health check operations.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}
