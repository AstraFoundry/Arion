// Package application answers update, release and download requests from
// the objects the platform publishes in the release bucket.
package application

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned by a Bucket for a missing object.
var ErrNotFound = errors.New("object not found")

// Bucket reads the release bucket. Keys are relative to its prefix.
type Bucket interface {
	// Get returns the object at key, at most limit bytes.
	Get(ctx context.Context, key string, limit int64) ([]byte, error)
	// PresignGet returns a URL that downloads key until ttl has passed.
	PresignGet(key string, ttl time.Duration, now time.Time) (string, error)
}
