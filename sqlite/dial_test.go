package sqlite_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/sqlite"
)

func TestDialService_CreateDial(t *testing.T) {
	// Ensure a dial can be created by a user & a membership for the user is automatically created.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})

		s := sqlite.NewDialService(db)
		dial := &wtf.Dial{Name: "mydial"}

		// Create new dial. Ensure the current user is the owner & an invite code is generated.
		if err := s.CreateDial(ctx0, dial); err != nil {
			t.Fatal(err)
		} else if got, want := dial.ID, 1; got != want {
			t.Fatalf("ID=%v, want %v", got, want)
		} else if got, want := dial.UserID, 1; got != want {
			t.Fatalf("UserID=%v, want %v", got, want)
		} else if dial.InviteCode == "" {
			t.Fatal("expected invite code generation")
		} else if dial.CreatedAt.IsZero() {
			t.Fatal("expected created at")
		} else if dial.UpdatedAt.IsZero() {
			t.Fatal("expected updated at")
		} else if dial.User == nil {
			t.Fatal("expected user")
		}

		// Fetch dial from database & compare.
		if other, err := s.FindDialByID(ctx0, 1); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(dial, other) {
			t.Fatalf("mismatch: %#v != %#v", dial, other)
		}

		// Ensure membership for owner automatically created.
		if _, n, err := sqlite.NewDialMembershipService(db).FindDialMemberships(ctx0, wtf.DialMembershipFilter{DialID: &dial.ID}); err != nil {
			t.Fatal(err)
		} else if n != 1 {
			t.Fatal("expected owner membership auto-creation")
		}
	})

	// Ensure that creating a nameless dial returns an error.
	t.Run("ErrNameRequired", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		_, ctx0 := MustCreateUser(t, context.Background(), db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})

		if err := sqlite.NewDialService(db).CreateDial(ctx0, &wtf.Dial{}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EINVALID || wtf.ErrorMessage(err) != "Dial name required." {
			t.Fatal(err)
		}
	})

	// Ensure that creating a dial with a long name returns an error.
	t.Run("ErrNameTooLong", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		_, ctx0 := MustCreateUser(t, context.Background(), db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})

		if err := sqlite.NewDialService(db).CreateDial(ctx0, &wtf.Dial{Name: strings.Repeat("X", wtf.MaxDialNameLen+1)}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EINVALID || wtf.ErrorMessage(err) != "Dial name too long." {
			t.Fatal(err)
		}
	})

	// Ensure user is logged in when creating a dial.
	t.Run("ErrUserRequired", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		if err := sqlite.NewDialService(db).CreateDial(context.Background(), &wtf.Dial{}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EUNAUTHORIZED || wtf.ErrorMessage(err) != "You must be logged in to create a dial." {
			t.Fatal(err)
		}
	})
}

func TestDialService_UpdateDial(t *testing.T) {
	// Ensure a dial name can be updated.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialService(db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})
		dial := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "NAME"})

		// Update dial.
		newName := "mydial2"
		uu, err := s.UpdateDial(ctx0, dial.ID, wtf.DialUpdate{Name: &newName})
		if err != nil {
			t.Fatal(err)
		} else if got, want := uu.Name, "mydial2"; got != want {
			t.Fatalf("Name=%v, want %v", got, want)
		}

		// Fetch dial from database & compare.
		if other, err := s.FindDialByID(ctx0, 1); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(uu, other) {
			t.Fatalf("mismatch: %#v != %#v", uu, other)
		}
	})
}

func TestDialService_FindDials(t *testing.T) {
	// Ensure all dials that are owned by user can be fetched.
	t.Run("Owned", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "john", Email: "john@gmail.com"})
		_, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})

		MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "dial0"})
		MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "dial1"})
		MustCreateDial(t, ctx1, db, &wtf.Dial{Name: "dial2"})

		s := sqlite.NewDialService(db)
		if a, n, err := s.FindDials(ctx0, wtf.DialFilter{}); err != nil {
			t.Fatal(err)
		} else if got, want := len(a), 2; got != want {
			t.Fatalf("len=%v, want %v", got, want)
		} else if got, want := a[0].Name, "dial0"; got != want {
			t.Fatalf("[0]=%v, want %v", got, want)
		} else if got, want := a[1].Name, "dial1"; got != want {
			t.Fatalf("[1]=%v, want %v", got, want)
		} else if got, want := n, 2; got != want {
			t.Fatalf("n=%v, want %v", got, want)
		}
	})

	// Ensure all dials that user is a member of can be fetched.
	t.Run("MemberOf", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "john", Email: "john@gmail.com"})
		user1, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})

		dial0 := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "dial0"})
		MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "dial1"})
		MustCreateDialMembership(t, ctx1, db, &wtf.DialMembership{DialID: dial0.ID, UserID: user1.ID})

		s := sqlite.NewDialService(db)
		if a, n, err := s.FindDials(ctx1, wtf.DialFilter{}); err != nil {
			t.Fatal(err)
		} else if got, want := len(a), 1; got != want {
			t.Fatalf("len=%v, want %v", got, want)
		} else if got, want := a[0].Name, "dial0"; got != want {
			t.Fatalf("[0]=%v, want %v", got, want)
		} else if got, want := n, 1; got != want {
			t.Fatalf("n=%v, want %v", got, want)
		}
	})

	// Ensure dial can be found by invite code even if not logged in.
	t.Run("InviteCode", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "john", Email: "john@gmail.com"})

		dial0 := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "dial0"})
		MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "dial1"})

		s := sqlite.NewDialService(db)
		if a, n, err := s.FindDials(context.Background(), wtf.DialFilter{InviteCode: &dial0.InviteCode}); err != nil {
			t.Fatal(err)
		} else if got, want := len(a), 1; got != want {
			t.Fatalf("len=%v, want %v", got, want)
		} else if got, want := a[0].Name, "dial0"; got != want {
			t.Fatalf("[0]=%v, want %v", got, want)
		} else if got, want := n, 1; got != want {
			t.Fatalf("n=%v, want %v", got, want)
		}
	})
}

func TestDialService_DeleteDial(t *testing.T) {
	// Ensure a dial can be deleted by the owner.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialService(db)

		_, ctx0 := MustCreateUser(t, context.Background(), db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})
		dial := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "NAME"})

		if err := s.DeleteDial(ctx0, dial.ID); err != nil {
			t.Fatal(err)
		} else if _, err := s.FindDialByID(ctx0, dial.ID); wtf.ErrorCode(err) != wtf.ENOTFOUND {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestDialService_AverageDialValueReport(t *testing.T) {
	// Ensure we can compute the average dial value across time for one dial.
	t.Run("SingleDial", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewDialService(db)

		db.Now = func() time.Time {
			return time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
		}

		ctx := context.Background()
		_, ctx0 := MustCreateUser(t, ctx, db, &wtf.User{Name: "jane"})
		_, ctx1 := MustCreateUser(t, ctx, db, &wtf.User{Name: "joe"})

		dial0 := MustCreateDial(t, ctx0, db, &wtf.Dial{Name: "DIAL0"})
		membership0 := MustFindDialMembershipByID(t, ctx0, db, 1)
		MustCreateDialMembership(t, ctx1, db, &wtf.DialMembership{DialID: dial0.ID})

		// Update value after one hour (avg 25).
		db.Now = func() time.Time {
			return time.Date(2000, time.January, 1, 1, 0, 0, 0, time.UTC)
		}
		MustSetDialMembershipValue(t, ctx0, db, membership0.ID, 50)

		// Update value after 4 hour (avg 50).
		db.Now = func() time.Time {
			return time.Date(2000, time.January, 1, 4, 0, 0, 0, time.UTC)
		}
		MustSetDialMembershipValue(t, ctx0, db, membership0.ID, 100)

		// Update value after 6 hours (avg 55).
		db.Now = func() time.Time {
			return time.Date(2000, time.January, 1, 6, 0, 0, 0, time.UTC)
		}
		MustSetDialMembershipValue(t, ctx0, db, membership0.ID, 100)

		// Generate hourly report.
		start := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2000, time.January, 1, 5, 0, 0, 0, time.UTC)
		report, err := s.AverageDialValueReport(ctx0, start, end, time.Hour)
		if err != nil {
			t.Fatal(err)
		} else if got, want := report.Records[0], (&wtf.DialValueRecord{Value: 0, Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)}); !reflect.DeepEqual(got, want) {
			t.Fatalf("[]=%#v, want %#v", got, want)
		}
	})
}

// MustFindDialByID finds a dial by ID. Fatal on error.
func MustFindDialByID(tb testing.TB, ctx context.Context, db *sqlite.DB, id int) *wtf.Dial {
	tb.Helper()
	dial, err := sqlite.NewDialService(db).FindDialByID(ctx, id)
	if err != nil {
		tb.Fatal(err)
	}
	return dial
}

// MustCreateDial creates a dial in the database. Fatal on error.
func MustCreateDial(tb testing.TB, ctx context.Context, db *sqlite.DB, dial *wtf.Dial) *wtf.Dial {
	tb.Helper()
	if err := sqlite.NewDialService(db).CreateDial(ctx, dial); err != nil {
		tb.Fatal(err)
	}
	return dial
}
