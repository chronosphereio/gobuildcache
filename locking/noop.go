package locking

// NoOpGroup is a Group implementation that performs no locking.
// Every call executes the function immediately. This is useful for testing
// or scenarios where locking is not needed.
type NoOpGroup struct{}

// NewNoOpGroup creates a new NoOpGroup.
func NewNoOpGroup() *NoOpGroup {
	return &NoOpGroup{}
}

func (n *NoOpGroup) DoWithLock(key string, fn func() (interface{}, error)) (v interface{}, err error) {
	v, err = fn()
	return v, err
}
