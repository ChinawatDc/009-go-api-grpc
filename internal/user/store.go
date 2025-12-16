package user

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("user not found")

type Store interface {
	Get(id string) (User, error)
	Create(u User) (User, error)
}

type InMemoryStore struct {
	mu    sync.RWMutex
	items map[string]User
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{items: make(map[string]User)}
}

func (s *InMemoryStore) Get(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.items[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (s *InMemoryStore) Create(u User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[u.ID] = u
	return u, nil
}