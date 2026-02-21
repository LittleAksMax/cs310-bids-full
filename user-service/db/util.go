package db

import (
	"fmt"
	"net"
	"net/url"

	"golang.org/x/crypto/bcrypt"

	"github.com/LittleAksMax/bids-user-service/config"
)

// DSN constructs a Postgres connection string from environment variables.
// Required env vars: DATABASE_HOST, DATABASE_PORT, DATABASE_USER, DATABASE_PASSWORD, DATABASE_NAME.
func DSN(cfg *config.Config) string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		url.QueryEscape(cfg.DBUser),
		url.QueryEscape(cfg.DBPassword),
		net.JoinHostPort(cfg.DBHost, cfg.DBPort),
		cfg.DBName,
	)
}

// HashPassword generates a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MaxCost)
	return string(bytes), err
}

// CheckPassword verifies if the provided password matches the hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
