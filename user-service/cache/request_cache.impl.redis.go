package cache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConnectionConfig struct {
	RedisHost     string
	RedisPort     int
	RedisPassword string
}

func (cfg *RedisConnectionConfig) DSN() string {
	return fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)
}

var (
	ErrNotFound = errors.New("refresh token not found")
	ErrExpired  = errors.New("refresh token expired")
)

type RedisRefreshStore struct {
	Client *redis.Client
	keyNS  string // namespace prefix e.g. "refresh"
}

// NewRedisRefreshStore creates a Redis-backed RefreshTokenStore using values from Config.
// Returns an error if required Redis connection details are missing.
func NewRedisRefreshStore(ctx context.Context, cfg *RedisConnectionConfig) (*RedisRefreshStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.DSN(),
		Password: cfg.RedisPassword,
	})
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		err := client.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	log.Println("Pinged your deployment. You successfully connected to Redis!")

	return &RedisRefreshStore{Client: client, keyNS: "refresh"}, nil
}

func (s *RedisRefreshStore) buildKey(token string) string {
	return s.keyNS + ":" + token
}

// Set stores the refresh token with TTL.
func (s *RedisRefreshStore) Set(ctx context.Context, token string, userID string, expiresIn time.Duration) error {
	if token == "" || userID == "" {
		return errors.New("token and userID required")
	}
	key := s.buildKey(token)
	// Store userID as value; expiration managed by Redis.
	return s.Client.Set(ctx, key, userID, expiresIn).Err()
}

// Get retrieves the userID and calculates expiresAt using TTL.
func (s *RedisRefreshStore) Get(ctx context.Context, token string) (string, time.Time, error) {
	key := s.buildKey(token)
	val, err := s.Client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", time.Time{}, ErrNotFound
		}
		return "", time.Time{}, err
	}
	ttl, err := s.Client.TTL(ctx, key).Result()
	if err != nil {
		return "", time.Time{}, err
	}
	if ttl <= 0 { // key exists but no TTL or expired
		return "", time.Time{}, ErrExpired
	}
	expiresAt := time.Now().Add(ttl)
	return val, expiresAt, nil
}

// Delete removes the refresh token key.
func (s *RedisRefreshStore) Delete(ctx context.Context, token string) error {
	key := s.buildKey(token)
	return s.Client.Del(ctx, key).Err()
}

// HealthCheck checks if the Redis connection is healthy.
func (s *RedisRefreshStore) HealthCheck(ctx context.Context) error {
	return s.Client.Ping(ctx).Err()
}
