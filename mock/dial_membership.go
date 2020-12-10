package mock

import (
	"context"

	"github.com/benbjohnson/wtf"
)

var _ wtf.DialMembershipService = (*DialMembershipService)(nil)

type DialMembershipService struct {
	FindDialMembershipByIDFn func(ctx context.Context, id int) (*wtf.DialMembership, error)
	FindDialMembershipsFn    func(ctx context.Context, filter wtf.DialMembershipFilter) ([]*wtf.DialMembership, int, error)
	CreateDialMembershipFn   func(ctx context.Context, membership *wtf.DialMembership) error
	UpdateDialMembershipFn   func(ctx context.Context, id int, upd wtf.DialMembershipUpdate) (*wtf.DialMembership, error)
	DeleteDialMembershipFn   func(ctx context.Context, id int) error
}

func (s *DialMembershipService) FindDialMembershipByID(ctx context.Context, id int) (*wtf.DialMembership, error) {
	return s.FindDialMembershipByIDFn(ctx, id)
}

func (s *DialMembershipService) FindDialMemberships(ctx context.Context, filter wtf.DialMembershipFilter) ([]*wtf.DialMembership, int, error) {
	return s.FindDialMembershipsFn(ctx, filter)
}

func (s *DialMembershipService) CreateDialMembership(ctx context.Context, membership *wtf.DialMembership) error {
	return s.CreateDialMembershipFn(ctx, membership)
}

func (s *DialMembershipService) UpdateDialMembership(ctx context.Context, id int, upd wtf.DialMembershipUpdate) (*wtf.DialMembership, error) {
	return s.UpdateDialMembershipFn(ctx, id, upd)
}

func (s *DialMembershipService) DeleteDialMembership(ctx context.Context, id int) error {
	return s.DeleteDialMembershipFn(ctx, id)
}
