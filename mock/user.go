package mock

import (
	"context"

	"github.com/benbjohnson/wtf"
)

var _ wtf.UserService = (*UserService)(nil)

type UserService struct {
	FindUserByIDFn func(ctx context.Context, id int) (*wtf.User, error)
	FindUsersFn    func(ctx context.Context, filter wtf.UserFilter) ([]*wtf.User, int, error)
	CreateUserFn   func(ctx context.Context, user *wtf.User) error
	UpdateUserFn   func(ctx context.Context, id int, upd wtf.UserUpdate) (*wtf.User, error)
	DeleteUserFn   func(ctx context.Context, id int) error
}

func (s *UserService) FindUserByID(ctx context.Context, id int) (*wtf.User, error) {
	return s.FindUserByIDFn(ctx, id)
}

func (s *UserService) FindUsers(ctx context.Context, filter wtf.UserFilter) ([]*wtf.User, int, error) {
	return s.FindUsersFn(ctx, filter)
}

func (s *UserService) CreateUser(ctx context.Context, user *wtf.User) error {
	return s.CreateUserFn(ctx, user)
}

func (s *UserService) UpdateUser(ctx context.Context, id int, upd wtf.UserUpdate) (*wtf.User, error) {
	return s.UpdateUserFn(ctx, id, upd)
}

func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	return s.DeleteUserFn(ctx, id)
}
