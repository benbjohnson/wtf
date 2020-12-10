package sqlite_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/sqlite"
)

func TestDialMembershipService_CreateDialMembership(t *testing.T) {
	// Ensure we can create a dial membership.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})
		_, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jim", Email: "jim@gmail.com"})
		dial := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL"})

		s := sqlite.NewDialMembershipService(db)
		membership := &wtf.DialMembership{
			DialID: dial.ID,
			Value:  50,
		}

		// Create new membership. One membership should already exist (1) since it
		// is automatically created for the dial owner.
		if err := s.CreateDialMembership(ctx1, membership); err != nil {
			t.Fatal(err)
		} else if got, want := membership.ID, 2; got != want {
			t.Fatalf("ID=%v, want %v", got, want)
		} else if got, want := membership.Dial.Value, 25; got != want {
			t.Fatalf("Dial.Value=%v, want %v", got, want)
		}

		// Fetch membership & compare.
		if other, err := s.FindDialMembershipByID(ctx1, membership.ID); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(membership, other) {
			t.Fatalf("mismatch: %#v != %#v", membership, other)
		}
	})

	// Ensure an error is returned if we do not have an associated dial.
	t.Run("ErrDialRequired", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})

		if err := s.CreateDialMembership(ctx0, &wtf.DialMembership{}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EINVALID || wtf.ErrorMessage(err) != `Dial required for membership.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	// Ensure an error is returned if user is not currently logged in.
	t.Run("ErrUserRequired", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})
		dial := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL"})

		if err := s.CreateDialMembership(ctx, &wtf.DialMembership{DialID: dial.ID}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EUNAUTHORIZED || wtf.ErrorMessage(err) != `You must be logged in to join a dial.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestDialMembershipService_UpdateDialMembership(t *testing.T) {
	// Ensure a membership value can be updated by owner.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})
		_, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jim", Email: "jim@gmail.com"})

		dial := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL"})
		membership := MustCreateDialMembership(t, ctx1, db, &wtf.DialMembership{
			DialID: dial.ID,
			Value:  50,
		})

		// Update membership value.
		// New aggregate dial value is the integer average of the owner
		// membership (0) and the new value (25).
		newValue := 25
		var err error
		if membership, err = s.UpdateDialMembership(ctx1, membership.ID, wtf.DialMembershipUpdate{Value: &newValue}); err != nil {
			t.Fatal(err)
		} else if got, want := membership.Value, 25; got != want {
			t.Fatalf("Value=%v, want %v", got, want)
		} else if got, want := membership.Dial.Value, 13; got != want {
			t.Fatalf("Dial.Value=%v, want %v", got, want)
		}

		// Fetch membership & compare.
		if other, err := s.FindDialMembershipByID(ctx1, membership.ID); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(membership, other) {
			t.Fatalf("mismatch: %#v != %#v", membership, other)
		}
	})

	// Ensure historical values are stored with a resolution of 1 minute.
	t.Run("DialValueRollup", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialService(db)

		db.Now = func() time.Time {
			return time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
		}

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})
		dial0 := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL0"})
		membership0 := MustFindDialMembershipByID(t, ctx0, db, 1)

		// Update value after one minute.
		db.Now = func() time.Time {
			return time.Date(2000, time.January, 1, 0, 1, 0, 0, time.UTC)
		}
		MustSetDialMembershipValue(t, ctx0, db, membership0.ID, 50)
		MustSetDialMembershipValue(t, ctx0, db, membership0.ID, 60)

		// Update value after 1m30s.
		db.Now = func() time.Time {
			return time.Date(2000, time.January, 1, 0, 1, 30, 0, time.UTC)
		}
		MustSetDialMembershipValue(t, ctx0, db, membership0.ID, 10)

		// Update value after 5 minutes.
		db.Now = func() time.Time {
			return time.Date(2000, time.January, 1, 0, 5, 0, 0, time.UTC)
		}
		MustSetDialMembershipValue(t, ctx0, db, membership0.ID, 100)

		// Ensure only 4 values are stored.
		if values, err := s.DialValues(ctx, dial0.ID); err != nil {
			t.Fatal(err)
		} else if got, want := values, []int{0, 10, 100}; !reflect.DeepEqual(got, want) {
			t.Fatalf("DialValues()=%#v, want %#v", got, want)
		}
	})

	// Ensure an error is returned if another user tries to update a membership.
	t.Run("ErrUnauthorized", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})
		_, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jim", Email: "jim@gmail.com"})
		dial := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL"})
		membership := MustCreateDialMembership(t, ctx1, db, &wtf.DialMembership{
			DialID: dial.ID,
			Value:  50,
		})

		newValue := 25
		if _, err := s.UpdateDialMembership(ctx0, membership.ID, wtf.DialMembershipUpdate{Value: &newValue}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EUNAUTHORIZED || wtf.ErrorMessage(err) != `You do not have permission to update the dial membership.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	// Ensure membership value is between 0 & 100.
	t.Run("ErrValueOutOfRange", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})
		MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL"})

		newValue := -1
		if _, err := s.UpdateDialMembership(ctx0, 1, wtf.DialMembershipUpdate{Value: &newValue}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EINVALID || wtf.ErrorMessage(err) != `Dial value must be between 0 & 100.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestDialMembershipService_FindDialMemberships(t *testing.T) {
	// Ensure dial member can see all memberships in dial.
	t.Run("RestrictToDialMember", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})
		_, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "john"})
		_, ctx2 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jill"})

		// Dials will automatically create memberships for the owner.
		dial0 := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL0"})
		membership0 := MustFindDialMembershipByID(t, ctx0, db, 1)
		membership1 := MustCreateDialMembership(t, ctx1, db, &wtf.DialMembership{DialID: dial0.ID, Value: 10})
		membership2 := MustCreateDialMembership(t, ctx2, db, &wtf.DialMembership{DialID: dial0.ID, Value: 20})

		dial1 := MustCreateDial(t, ctx1, db, &wtf.Dial{Name: "DIAL1"})
		MustCreateDialMembership(t, ctx0, db, &wtf.DialMembership{DialID: dial1.ID, Value: 30})

		a, n, err := s.FindDialMemberships(ctx2, wtf.DialMembershipFilter{})
		if err != nil {
			t.Fatal(err)
		} else if got, want := len(a), 3; got != want {
			t.Fatalf("len=%v, want %v", got, want)
		} else if got, want := n, 3; got != want {
			t.Fatalf("n=%v, want %v", got, want)
		}

		// Self membership should appear first.
		if got, want := a[0], membership2; got.ID != want.ID {
			t.Fatalf("[].ID=%v, want %v", got.ID, want.ID)
		} else if got.Value != want.Value {
			t.Fatalf("[].Value=%v, want %v", got.Value, want.Value)
		}

		// Remaining memberships should appear sorted by user name.
		if got, want := a[1], membership0; got.ID != want.ID {
			t.Fatalf("[].ID=%v, want %v", got.ID, want.ID)
		} else if got.Value != want.Value {
			t.Fatalf("[].Value=%v, want %v", got.Value, want.Value)
		}
		if got, want := a[2], membership1; got.ID != want.ID {
			t.Fatalf("[].ID=%v, want %v", got.ID, want.ID)
		} else if got.Value != want.Value {
			t.Fatalf("[].Value=%v, want %v", got.Value, want.Value)
		}
	})

	// Ensure memberships can be filtered by dial.
	t.Run("DialID", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})

		// These dials will automatically create memberships for the owner (1,2).
		dial0 := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL0"})
		MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL1"})

		a, n, err := s.FindDialMemberships(ctx0, wtf.DialMembershipFilter{DialID: &dial0.ID})
		if err != nil {
			t.Fatal(err)
		} else if got, want := len(a), 1; got != want {
			t.Fatalf("len=%v, want %v", got, want)
		} else if got, want := n, 1; got != want {
			t.Fatalf("n=%v, want %v", got, want)
		} else if got, want := a[0].ID, 1; got != want {
			t.Fatalf("[].ID=%v, want %v", got, want)
		}
	})

	// Ensure memberships can be filtered by user.
	t.Run("DialID", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})
		user1, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jill"})
		dial0 := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL0"})
		membership0 := MustCreateDialMembership(t, ctx1, db, &wtf.DialMembership{DialID: dial0.ID, Value: 10})

		a, n, err := s.FindDialMemberships(ctx0, wtf.DialMembershipFilter{UserID: &user1.ID})
		if err != nil {
			t.Fatal(err)
		} else if got, want := len(a), 1; got != want {
			t.Fatalf("len=%v, want %v", got, want)
		} else if got, want := n, 1; got != want {
			t.Fatalf("n=%v, want %v", got, want)
		} else if got, want := a[0].ID, membership0.ID; got != want {
			t.Fatalf("[].ID=%v, want %v", got, want)
		}
	})
}

func TestDialMembershipService_DeleteDialMembership(t *testing.T) {
	// Ensure a membership owner can delete their membership.
	t.Run("ByMembershipOwner", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})
		_, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jim"})
		dial := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL"})
		membership := MustCreateDialMembership(t, ctx1, db, &wtf.DialMembership{DialID: dial.ID, Value: 50})

		if err := s.DeleteDialMembership(ctx1, membership.ID); err != nil {
			t.Fatal(err)
		}

		// Ensure membership has been deleted.
		if _, err := s.FindDialMembershipByID(ctx1, membership.ID); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.ENOTFOUND || wtf.ErrorMessage(err) != `Dial membership not found.` {
			t.Fatalf("unexpected error: %#v", err)
		}

		// Ensure dial aggregate value is updated.
		if other := MustFindDialByID(t, ctx0, db, dial.ID); other.Value != 0 {
			t.Fatalf("unexpected dial value: %d", other.Value)
		}
	})

	// Ensure a dial owner can delete another user's membership.
	t.Run("ByDialOwner", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})
		_, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jim"})
		dial := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL"})
		membership := MustCreateDialMembership(t, ctx1, db, &wtf.DialMembership{DialID: dial.ID, Value: 50})

		if err := s.DeleteDialMembership(ctx0, membership.ID); err != nil {
			t.Fatal(err)
		}

		// Ensure membership has been deleted.
		if _, err := s.FindDialMembershipByID(ctx1, membership.ID); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.ENOTFOUND || wtf.ErrorMessage(err) != `Dial membership not found.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	// Ensure owner's membership cannot be deleted.
	t.Run("ErrCannotDeleteOwnerMembership", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})
		dial := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL"})

		if err := s.DeleteDialMembership(ctx0, 1); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.ECONFLICT || wtf.ErrorMessage(err) != `Dial owner may not delete their own membership.` {
			t.Fatalf("unexpected error: %#v", err)
		}

		// Ensure dial aggregate value is updated.
		if other := MustFindDialByID(t, ctx0, db, dial.ID); other.Value != 0 {
			t.Fatalf("unexpected dial value: %d", other.Value)
		}
	})

	// Ensure a non-owner (of dial or membership) cannot delete a membership.
	t.Run("ErrUnauthorized", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialMembershipService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})
		_, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jim"})
		_, ctx2 := MustCreateUser(t, ctx, db, &wtf.User{Name: "bob"})
		dial := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL"})
		membership0 := MustCreateDialMembership(t, ctx1, db, &wtf.DialMembership{DialID: dial.ID, Value: 50})
		MustCreateDialMembership(t, ctx2, db, &wtf.DialMembership{DialID: dial.ID, Value: 50})

		if err := s.DeleteDialMembership(ctx2, membership0.ID); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EUNAUTHORIZED || wtf.ErrorMessage(err) != `You do not have permission to delete the dial membership.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

// MustFindDialMembershipByID finds a membership in the database. Fatal on error.
func MustFindDialMembershipByID(tb testing.TB, ctx context.Context, db *sqlite.DB, id int) *wtf.DialMembership {
	tb.Helper()
	membership, err := sqlite.NewDialMembershipService(db).FindDialMembershipByID(ctx, id)
	if err != nil {
		tb.Fatal(err)
	}
	return membership
}

// MustCreateDialMembership creates a membership in the database. Fatal on error.
func MustCreateDialMembership(tb testing.TB, ctx context.Context, db *sqlite.DB, membership *wtf.DialMembership) *wtf.DialMembership {
	tb.Helper()
	if err := sqlite.NewDialMembershipService(db).CreateDialMembership(ctx, membership); err != nil {
		tb.Fatal(err)
	}
	return membership
}

// MustSetDialMembershipValue updates the membership value. Fatal on error.
func MustSetDialMembershipValue(tb testing.TB, ctx context.Context, db *sqlite.DB, id, value int) {
	tb.Helper()
	if _, err := sqlite.NewDialMembershipService(db).UpdateDialMembership(ctx, id, wtf.DialMembershipUpdate{Value: &value}); err != nil {
		tb.Fatal(err)
	}
}
