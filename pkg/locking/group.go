package locking

// locking.Group is an abstraction for running functions with mutual exclusion
// over sets of keys.
type Group interface {
	// DoWithLock runs the given function with mutual exclusion over the given key.
	DoWithLock(key string, fn func() (interface{}, error)) (v interface{}, err error)
}
