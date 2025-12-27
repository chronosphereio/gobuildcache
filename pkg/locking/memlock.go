package locking

import "sync"

// MemLock is a Group implementation that uses in-memory locks (mutexes) for mutual
// exclusuion. It only works within a single gobuildcache process and doesn't work
// if there are multiple gobuildcache processes running concurrently on the same
// filesystem. It's used primarily in tests.
type MemLock struct {
	sync.Mutex
	locks map[string]*sync.Mutex
}

func NewMemLock() *MemLock {
	return &MemLock{
		locks: make(map[string]*sync.Mutex),
	}
}

func (s *MemLock) DoWithLock(key string, fn func() (interface{}, error)) (v interface{}, err error) {
	s.Lock()
	lock, ok := s.locks[key]
	if !ok {
		lock = &sync.Mutex{}
		s.locks[key] = lock
	}
	s.Unlock()
	lock.Lock()
	defer lock.Unlock()
	return fn()
}
