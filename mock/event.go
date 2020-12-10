package mock

import (
	"context"

	"github.com/benbjohnson/wtf"
)

var _ wtf.EventService = (*EventService)(nil)

type EventService struct {
	PublishEventFn func(userID int, event wtf.Event)
	SubscribeFn    func(ctx context.Context) (wtf.Subscription, error)
}

func (s *EventService) PublishEvent(userID int, event wtf.Event) {
	s.PublishEventFn(userID, event)
}

func (s *EventService) Subscribe(ctx context.Context) (wtf.Subscription, error) {
	return s.SubscribeFn(ctx)
}

type Subscription struct {
	CloseFn func() error
	CFn     func() <-chan wtf.Event
}

func (s *Subscription) Close() error {
	return s.CloseFn()
}

func (s *Subscription) C() <-chan wtf.Event {
	return s.CFn()
}
