package user

import "github.com/google/uuid"

type User struct {
	ID    string
	Email string
	Name  string
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetUser(id string) (User, error) {
	return s.store.Get(id)
}

func (s *Service) CreateUser(email, name string) (User, error) {
	u := User{
		ID:    uuid.NewString(),
		Email: email,
		Name:  name,
	}
	return s.store.Create(u)
}