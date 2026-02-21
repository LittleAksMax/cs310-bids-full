package api

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/LittleAksMax/bids-user-service/config"
	"github.com/LittleAksMax/bids-user-service/health"
	"github.com/LittleAksMax/bids-user-service/repository"
	"github.com/LittleAksMax/bids-user-service/service"
	"github.com/LittleAksMax/bids-util/requests"
)

// NewRouter constructs the main API router by wiring middleware and routes defined elsewhere.
func NewRouter(pool *sql.DB, cfg *config.Config, secureMode bool) http.Handler {
	r := chi.NewRouter()

	RegisterMiddleware(r)

	requests.ApplyCORS(
		r,
		cfg.AllowedOrigins,
		[]string{"GET", "POST", "PUT", "DELETE"},
		[]string{"Accept", "Authorization", "Content-Type", "X-Auth-Claims", "X-Auth-Ts", "X-Auth-Sig"},
		[]string{"Set-Cookie"},
		true,
		300,
	)

	// Initialise authentication layers
	userRepo := repository.NewUserRepository()
	credRepo := repository.NewPasswordCredentialRepository()
	authService := service.NewAuthService(pool, userRepo, credRepo, cfg.PasswordPepper)

	// Initialise token management layers
	refreshTokenRepo := repository.NewRefreshTokenRepository()
	tokenService := service.NewTokenService(
		pool,
		refreshTokenRepo,
		userRepo,
		cfg.AccessTokenSecret, cfg.RefreshTokenSecret,
		cfg.AccessTokenTTL, cfg.RefreshTokenTTL, cfg.TokenIssuer, cfg.TokenAudience)

	// Initialise cookie management services
	cookieService := service.NewCookieService(
		"/auth/refresh",
		"refresh_token",
		int(cfg.RefreshTokenTTL.Seconds()),
		http.SameSiteStrictMode,
		secureMode)

	// Initialise controllers
	authController := NewAuthController(authService, tokenService, cookieService)

	// Create health checkers map
	healthCheckers := map[string]health.HealthChecker{
		"database": health.NewDBHealthChecker(pool),
	}

	RegisterRoutes(r, authController, healthCheckers)

	return r
}
