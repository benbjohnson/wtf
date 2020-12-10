package mock

import (
	"context"
	"time"

	"github.com/benbjohnson/wtf"
)

var _ wtf.DialService = (*DialService)(nil)

// DialService represents a mock of wtf.DialService.
type DialService struct {
	FindDialByIDFn           func(ctx context.Context, id int) (*wtf.Dial, error)
	FindDialsFn              func(ctx context.Context, filter wtf.DialFilter) ([]*wtf.Dial, int, error)
	CreateDialFn             func(ctx context.Context, dial *wtf.Dial) error
	UpdateDialFn             func(ctx context.Context, id int, upd wtf.DialUpdate) (*wtf.Dial, error)
	DeleteDialFn             func(ctx context.Context, id int) error
	AverageDialValueReportFn func(ctx context.Context, start, end time.Time, interval time.Duration) (*wtf.DialValueReport, error)
}

func (s *DialService) FindDialByID(ctx context.Context, id int) (*wtf.Dial, error) {
	return s.FindDialByIDFn(ctx, id)
}

func (s *DialService) FindDials(ctx context.Context, filter wtf.DialFilter) ([]*wtf.Dial, int, error) {
	return s.FindDialsFn(ctx, filter)
}

func (s *DialService) CreateDial(ctx context.Context, dial *wtf.Dial) error {
	return s.CreateDialFn(ctx, dial)
}

func (s *DialService) UpdateDial(ctx context.Context, id int, upd wtf.DialUpdate) (*wtf.Dial, error) {
	return s.UpdateDialFn(ctx, id, upd)
}

func (s *DialService) DeleteDial(ctx context.Context, id int) error {
	return s.DeleteDialFn(ctx, id)
}

func (s *DialService) AverageDialValueReport(ctx context.Context, start, end time.Time, interval time.Duration) (*wtf.DialValueReport, error) {
	return s.AverageDialValueReportFn(ctx, start, end, interval)
}
