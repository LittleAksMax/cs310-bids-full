package cache

import (
	"context"
	"time"

	"github.com/LittleAksMax/bids-user-service/health"
)

// RequestCache is a key-value store for caching web requests for policies.
type RequestCache interface {
	health.HealthChecker
	Set(ctx context.Context, key string, value string, expiresIn time.Duration) error
	Get(ctx context.Context, key string) (value string, expiresAt time.Time, err error)
	Delete(ctx context.Context, key string) error
}
