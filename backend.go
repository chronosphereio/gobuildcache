package main

import (
	"io"
	"time"
)

// CacheBackend defines the interface for cache storage backends.
// Implementations can be swapped to use different storage mechanisms.
type CacheBackend interface {
	// Put stores an object in the cache.
	// actionID is the cache key, outputID is stored with the body,
	// body is the content to store, and bodySize is the size in bytes.
	// Returns the absolute path to the stored file on disk.
	Put(actionID, outputID []byte, body io.Reader, bodySize int64) (diskPath string, err error)

	// Get retrieves an object from the cache.
	// actionID is the cache key to look up.
	// Returns outputID, diskPath, size, time, and whether it was a miss.
	Get(actionID []byte) (outputID []byte, diskPath string, size int64, putTime *time.Time, miss bool, err error)

	// Close performs any cleanup operations needed by the backend.
	Close() error

	// Clear removes all entries from the cache.
	Clear() error
}

