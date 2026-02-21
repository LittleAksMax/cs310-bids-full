package api

import (
	"net/http"

	"github.com/LittleAksMax/bids-util/requests"
	"github.com/go-chi/chi/v5"

	"github.com/LittleAksMax/bids-user-service/health"
)

// Health handler implementation that checks all registered services.
// Always returns success=true since the request itself was fulfilled.
// Individual service health is reported in the data field.
func Health(checkers map[string]health.HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statuses := make(map[string]interface{})
		allHealthy := true

		for name, checker := range checkers {
			if err := checker.HealthCheck(r.Context()); err != nil {
				statuses[name] = map[string]interface{}{
					"status": "unhealthy",
					"error":  err.Error(),
				}
				allHealthy = false
			} else {
				statuses[name] = map[string]interface{}{
					"status": "healthy",
				}
			}
		}

		// Determine HTTP status code based on health
		statusCode := http.StatusOK
		if !allHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		requests.WriteJSON(w, statusCode, requests.APIResponse{
			Success: true,
			Data:    statuses,
		})
	}
}

// RegisterRoutes registers all endpoint handlers using the controller methods.
func RegisterRoutes(r chi.Router, healthCheckers map[string]health.HealthChecker) {
	// Health
	r.Get("/health", Health(healthCheckers))

	// Auth routes
	r.Route("/users", func(r chi.Router) {

	})
}
