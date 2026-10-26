package redisstore

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const visitsKey = "visits"

// Store provides access to the Redis-backed visit counter.
type Store struct {
	client *redis.Client
}

// New creates a Redis store from a URL such as
// redis://localhost:6379/0.
func New(redisURL string) (*Store, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse Redis URL: %w", err)
	}

	client := redis.NewClient(options)

	return &Store{
		client: client,
	}, nil
}

// Ping verifies that Redis is reachable.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping Redis: %w", err)
	}

	return nil
}

// Increment atomically increments and returns the visit counter.
func (s *Store) Increment(ctx context.Context) (int64, error) {
	value, err := s.client.Incr(ctx, visitsKey).Result()
	if err != nil {
		return 0, fmt.Errorf("increment visit counter: %w", err)
	}

	return value, nil
}

// Reset sets the visit counter to zero.
func (s *Store) Reset(ctx context.Context) error {
	if err := s.client.Set(ctx, visitsKey, 0, 0).Err(); err != nil {
		return fmt.Errorf("reset visit counter: %w", err)
	}

	return nil
}

// Close releases the client's resources.
func (s *Store) Close() error {
	return s.client.Close()
}
