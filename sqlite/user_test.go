package sqlite_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/sqlite"
)

func TestUserService_CreateUser(t *testing.T) {
	// Ensure user can be created.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewUserService(db)

		u := &wtf.User{
			Name:  "susy",
			Email: "susy@gmail.com",
		}

		// Create new user & verify ID and timestamps are set.
		if err := s.CreateUser(context.Background(), u); err != nil {
			t.Fatal(err)
		} else if got, want := u.ID, 1; got != want {
			t.Fatalf("ID=%v, want %v", got, want)
		} else if u.CreatedAt.IsZero() {
			t.Fatal("expected created at")
		} else if u.UpdatedAt.IsZero() {
			t.Fatal("expected updated at")
		}

		// Create second user with email.
		u2 := &wtf.User{Name: "jane"}
		if err := s.CreateUser(context.Background(), u2); err != nil {
			t.Fatal(err)
		} else if got, want := u2.ID, 2; got != want {
			t.Fatalf("ID=%v, want %v", got, want)
		}

		// Fetch user from database & compare.
		if other, err := s.FindUserByID(context.Background(), 1); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(u, other) {
			t.Fatalf("mismatch: %#v != %#v", u, other)
		}
	})

	// Ensure an error is returned if user name is not set.
	t.Run("ErrNameRequired", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewUserService(db)
		if err := s.CreateUser(context.Background(), &wtf.User{}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EINVALID || wtf.ErrorMessage(err) != `User name required.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	// Ensure user name & email can be updated by current user.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewUserService(db)
		user0, ctx0 := MustCreateUser(t, context.Background(), db, &wtf.User{
			Name:  "susy",
			Email: "susy@gmail.com",
		})

		// Update user.
		newName, newEmail := "jill", "jill@gmail.com"
		uu, err := s.UpdateUser(ctx0, user0.ID, wtf.UserUpdate{
			Name:  &newName,
			Email: &newEmail,
		})
		if err != nil {
			t.Fatal(err)
		} else if got, want := uu.Name, "jill"; got != want {
			t.Fatalf("Name=%v, want %v", got, want)
		} else if got, want := uu.Email, "jill@gmail.com"; got != want {
			t.Fatalf("Email=%v, want %v", got, want)
		}

		// Fetch user from database & compare.
		if other, err := s.FindUserByID(context.Background(), 1); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(uu, other) {
			t.Fatalf("mismatch: %#v != %#v", uu, other)
		}
	})

	// Ensure updating a user is restricted only to the current user.
	t.Run("ErrUnauthorized", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewUserService(db)
		user0, _ := MustCreateUser(t, context.Background(), db, &wtf.User{Name: "NAME0"})
		_, ctx1 := MustCreateUser(t, context.Background(), db, &wtf.User{Name: "NAME1"})

		// Update user as another user.
		newName := "NEWNAME"
		if _, err := s.UpdateUser(ctx1, user0.ID, wtf.UserUpdate{Name: &newName}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EUNAUTHORIZED || wtf.ErrorMessage(err) != `You are not allowed to update this user.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	// Ensure user can delete self.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewUserService(db)
		user0, ctx0 := MustCreateUser(t, context.Background(), db, &wtf.User{Name: "john"})

		// Delete user & ensure it is actually gone.
		if err := s.DeleteUser(ctx0, user0.ID); err != nil {
			t.Fatal(err)
		} else if _, err := s.FindUserByID(ctx0, user0.ID); wtf.ErrorCode(err) != wtf.ENOTFOUND {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	// Ensure an error is returned if deleting a non-existent user.
	t.Run("ErrNotFound", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewUserService(db)
		if err := s.DeleteUser(context.Background(), 1); wtf.ErrorCode(err) != wtf.ENOTFOUND {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	// Ensure deleting a user is restricted only to the current user.
	t.Run("ErrUnauthorized", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewUserService(db)
		user0, _ := MustCreateUser(t, context.Background(), db, &wtf.User{Name: "NAME0"})
		_, ctx1 := MustCreateUser(t, context.Background(), db, &wtf.User{Name: "NAME1"})

		if err := s.DeleteUser(ctx1, user0.ID); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EUNAUTHORIZED || wtf.ErrorMessage(err) != `You are not allowed to delete this user.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestUserService_FindUser(t *testing.T) {
	// Ensure an error is returned if fetching a non-existent user.
	t.Run("ErrNotFound", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewUserService(db)
		if _, err := s.FindUserByID(context.Background(), 1); wtf.ErrorCode(err) != wtf.ENOTFOUND {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestUserService_FindUsers(t *testing.T) {
	// Ensure users can be fetched by email address.
	t.Run("Email", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewUserService(db)

		ctx := context.Background()
		MustCreateUser(t, ctx, db, &wtf.User{Name: "john", Email: "john@gmail.com"})
		MustCreateUser(t, ctx, db, &wtf.User{Name: "jane", Email: "jane@gmail.com"})
		MustCreateUser(t, ctx, db, &wtf.User{Name: "frank", Email: "frank@gmail.com"})
		MustCreateUser(t, ctx, db, &wtf.User{Name: "sue", Email: "sue@gmail.com"})

		email := "jane@gmail.com"
		if a, n, err := s.FindUsers(ctx, wtf.UserFilter{Email: &email}); err != nil {
			t.Fatal(err)
		} else if got, want := len(a), 1; got != want {
			t.Fatalf("len=%v, want %v", got, want)
		} else if got, want := a[0].Name, "jane"; got != want {
			t.Fatalf("name=%v, want %v", got, want)
		} else if got, want := n, 1; got != want {
			t.Fatalf("n=%v, want %v", got, want)
		}
	})
}

// MustCreateUser creates a user in the database. Fatal on error.
func MustCreateUser(tb testing.TB, ctx context.Context, db *sqlite.DB, user *wtf.User) (*wtf.User, context.Context) {
	tb.Helper()
	if err := sqlite.NewUserService(db).CreateUser(ctx, user); err != nil {
		tb.Fatal(err)
	}
	return user, wtf.NewContextWithUser(ctx, user)
}
