// Package traefik_plugin_authheader transforms Authorization header into signed headers.
package traefik_plugin_authheader

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config holds the plugin configuration.
type Config struct {
	SharedSecretEnvVar string `json:"sharedSecretEnvVar,omitempty"`
	AuthHeaderPrefix   string `json:"authHeaderPrefix,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		SharedSecretEnvVar: "X_AUTH_SIG_SECRET",
		AuthHeaderPrefix:   "Bearer ",
	}
}

// AuthHeaderMiddleware is the middleware handler.
type AuthHeaderMiddleware struct {
	next             http.Handler
	sharedSecret     string
	authHeaderPrefix string
	name             string
}

// New creates a new AuthHeaderMiddleware instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.SharedSecretEnvVar == "" {
		config.SharedSecretEnvVar = "AUTH_SHARED_SECRET"
	}

	if config.AuthHeaderPrefix == "" {
		config.AuthHeaderPrefix = "Bearer "
	}

	secret := os.Getenv(config.SharedSecretEnvVar)
	if secret == "" {
		return nil, fmt.Errorf("environment variable %s not set", config.SharedSecretEnvVar)
	}

	return &AuthHeaderMiddleware{
		next:             next,
		sharedSecret:     secret,
		authHeaderPrefix: config.AuthHeaderPrefix,
		name:             name,
	}, nil
}

// ServeHTTP implements the http.Handler interface.
func (a *AuthHeaderMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract Authorization header
	authHeader := r.Header.Get("Authorization")

	// ALWAYS strip Authorization and any X-Auth-* headers from incoming request
	// to prevent header injection attacks
	r.Header.Del("Authorization")
	r.Header.Del("X-Auth-Claims")
	r.Header.Del("X-Auth-Ts")
	r.Header.Del("X-Auth-Sig")

	if authHeader == "" {
		// If no auth header, pass through without adding new headers
		a.next.ServeHTTP(w, r)
		return
	}

	// Extract substring (e.g., token part after "Bearer ")
	claimsStr := authHeader
	if strings.HasPrefix(authHeader, a.authHeaderPrefix) {
		claimsStr = authHeader[len(a.authHeaderPrefix):]
	}

	// Create X-Auth-Claims (base64 of substring)
	authClaims := base64.RawStdEncoding.EncodeToString([]byte(claimsStr))

	// Create X-Auth-Ts (UNIX epoch timestamp)
	authTs := fmt.Sprintf("%d", time.Now().Unix())

	// Create X-Auth-Sig (HMAC_SHA256)
	message := authTs + "." + authClaims
	sig := hmac.New(sha256.New, []byte(a.sharedSecret))
	sig.Write([]byte(message))
	authSig := hex.EncodeToString(sig.Sum(nil))

	// Add new headers
	r.Header.Set("X-Auth-Claims", authClaims)
	r.Header.Set("X-Auth-Ts", authTs)
	r.Header.Set("X-Auth-Sig", authSig)

	a.next.ServeHTTP(w, r)
}