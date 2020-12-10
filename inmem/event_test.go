package inmem_test

import (
	"context"
	"testing"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/inmem"
)

func TestEventService(t *testing.T) {
	t.Run("Subscribe", func(t *testing.T) {
		ctx := context.Background()
		ctx0 := wtf.NewContextWithUser(ctx, &wtf.User{ID: 1})
		ctx1 := wtf.NewContextWithUser(ctx, &wtf.User{ID: 2})

		s := inmem.NewEventService()
		sub0a, err := s.Subscribe(ctx0)
		if err != nil {
			t.Fatal(err)
		}
		sub0b, err := s.Subscribe(ctx0)
		if err != nil {
			t.Fatal(err)
		}
		sub1, err := s.Subscribe(ctx1)
		if err != nil {
			t.Fatal(err)
		}

		// Publish event to both users
		s.PublishEvent(1, wtf.Event{Type: "test1"})

		// Verify both of the first user's subscriptions received the event.
		select {
		case <-sub0a.C():
		default:
			t.Fatal("expected event")
		}

		select {
		case <-sub0b.C():
		default:
			t.Fatal("expected event")
		}

		// Ensure second user does not receive event.
		select {
		case <-sub1.C():
			t.Fatal("expected no event")
		default:
		}
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		ctx := context.Background()
		ctx0 := wtf.NewContextWithUser(ctx, &wtf.User{ID: 1})

		s := inmem.NewEventService()
		sub, err := s.Subscribe(ctx0)
		if err != nil {
			t.Fatal(err)
		}

		// Publish event & close.
		s.PublishEvent(1, wtf.Event{Type: "test1"})
		if err := sub.Close(); err != nil {
			t.Fatal(err)
		}

		// Verify event is still received.
		select {
		case <-sub.C():
		default:
			t.Fatal("expected event")
		}

		// Ensure channel is now closed.
		if _, ok := <-sub.C(); ok {
			t.Fatal("expected closed channel")
		}

		// Ensure unsubscribing twice is ok.
		if err := sub.Close(); err != nil {
			t.Fatal(err)
		}
	})
}
