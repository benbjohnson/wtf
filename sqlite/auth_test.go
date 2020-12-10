package sqlite_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/sqlite"
)

func TestAuthService_CreateAuth(t *testing.T) {
	// Ensure we can create a new auth object and associated user.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewAuthService(db)

		expiry := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
		auth := &wtf.Auth{
			Source:       wtf.AuthSourceGitHub,
			SourceID:     "SOURCEID",
			AccessToken:  "ACCESS",
			RefreshToken: "REFRESH",
			Expiry:       &expiry,
			User: &wtf.User{
				Name:  "jill",
				Email: "jill@gmail.com",
			},
		}

		// Create new auth object & ensure ID and timestamps are returned.
		if err := s.CreateAuth(context.Background(), auth); err != nil {
			t.Fatal(err)
		} else if got, want := auth.ID, 1; got != want {
			t.Fatalf("ID=%v, want %v", got, want)
		} else if auth.CreatedAt.IsZero() {
			t.Fatal("expected created at")
		} else if auth.UpdatedAt.IsZero() {
			t.Fatal("expected updated at")
		}

		// Fetch auth from database & compare.
		if other, err := s.FindAuthByID(context.Background(), 1); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(auth, other) {
			t.Fatalf("mismatch: %#v != %#v", auth, other)
		}

		// Fetching user should return auths.
		if user, err := sqlite.NewUserService(db).FindUserByID(context.Background(), 1); err != nil {
			t.Fatal(err)
		} else if len(user.Auths) != 1 {
			t.Fatal("expected auths")
		} else if auth := user.Auths[0]; auth.ID != 1 {
			t.Fatalf("unexpected auth: %#v", auth)
		}
	})

	// Ensure that a blank source field returns an error.
	t.Run("ErrSourceRequired", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		if err := sqlite.NewAuthService(db).CreateAuth(context.Background(), &wtf.Auth{
			User: &wtf.User{Name: "NAME"},
		}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EINVALID || wtf.ErrorMessage(err) != `Source required.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	// Ensure that a blank source ID field returns an error.
	t.Run("ErrSourceIDRequired", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		if err := sqlite.NewAuthService(db).CreateAuth(context.Background(), &wtf.Auth{
			Source: wtf.AuthSourceGitHub,
			User:   &wtf.User{Name: "NAME"},
		}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EINVALID || wtf.ErrorMessage(err) != `Source ID required.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	// Ensure that a blank access token field returns an error.
	t.Run("ErrAccessTokenRequired", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewAuthService(db)
		if err := s.CreateAuth(context.Background(), &wtf.Auth{
			Source:   wtf.AuthSourceGitHub,
			SourceID: "X",
			User:     &wtf.User{Name: "NAME"},
		}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EINVALID || wtf.ErrorMessage(err) != `Access token required.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	// Ensure that a user object is required when creating an auth.
	t.Run("ErrUserRequired", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewAuthService(db)
		if err := s.CreateAuth(context.Background(), &wtf.Auth{}); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EINVALID || wtf.ErrorMessage(err) != `User required.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestAuthService_DeleteAuth(t *testing.T) {
	// Ensure an auth object can be deleted by its owner.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewAuthService(db)
		auth0, ctx0 := MustCreateAuth(t, context.Background(), db, &wtf.Auth{
			Source:      wtf.AuthSourceGitHub,
			SourceID:    "X",
			AccessToken: "X", User: &wtf.User{Name: "X"},
		})

		// Delete auth & ensure it is actually gone.
		if err := s.DeleteAuth(ctx0, auth0.ID); err != nil {
			t.Fatal(err)
		} else if _, err := s.FindAuthByID(ctx0, auth0.ID); wtf.ErrorCode(err) != wtf.ENOTFOUND {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	// Ensure an error is returned if fetching a non-existent auth object.
	t.Run("ErrNotFound", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewAuthService(db)
		if err := s.DeleteAuth(context.Background(), 1); wtf.ErrorCode(err) != wtf.ENOTFOUND {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	// Ensure deleting a auth is restricted only to the owner.
	t.Run("ErrUnauthorized", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewAuthService(db)

		// We use test helpers to avoid redundant error checks that are not specific to our test.
		auth0, _ := MustCreateAuth(t, context.Background(), db, &wtf.Auth{
			Source:      wtf.AuthSourceGitHub,
			SourceID:    "X",
			AccessToken: "X", User: &wtf.User{Name: "X"},
		})
		_, ctx1 := MustCreateAuth(t, context.Background(), db, &wtf.Auth{
			Source:      wtf.AuthSourceGitHub,
			SourceID:    "Y",
			AccessToken: "Y", User: &wtf.User{Name: "Y"},
		})

		if err := s.DeleteAuth(ctx1, auth0.ID); err == nil {
			t.Fatal("expected error")
		} else if wtf.ErrorCode(err) != wtf.EUNAUTHORIZED || wtf.ErrorMessage(err) != `You are not allowed to delete this auth.` {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestAuthService_FindAuth(t *testing.T) {
	// Ensure we receive an error if fetching a non-existent auth.
	t.Run("ErrNotFound", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewAuthService(db)
		if _, err := s.FindAuthByID(context.Background(), 1); wtf.ErrorCode(err) != wtf.ENOTFOUND {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestAuthService_FindAuths(t *testing.T) {
	// Ensure we can fetch all auths for a single user.
	t.Run("User", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		s := sqlite.NewAuthService(db)

		ctx := context.Background()

		// Create two auths with the same user email which will group them.
		// The third auth is for a separate user.
		MustCreateAuth(t, context.Background(), db, &wtf.Auth{
			Source:      "SRCA",
			SourceID:    "X1",
			AccessToken: "ACCESSX1",
			User:        &wtf.User{Name: "X", Email: "x@y.com"},
		})
		MustCreateAuth(t, context.Background(), db, &wtf.Auth{
			Source:      "SRCB",
			SourceID:    "X2",
			AccessToken: "ACCESSX2",
			User:        &wtf.User{Name: "X", Email: "x@y.com"},
		})
		MustCreateAuth(t, context.Background(), db, &wtf.Auth{
			Source:      wtf.AuthSourceGitHub,
			SourceID:    "Y",
			AccessToken: "ACCESSY",
			User:        &wtf.User{Name: "Y"},
		})

		// Fetch auths and compare results.
		userID := 1
		if a, n, err := s.FindAuths(ctx, wtf.AuthFilter{UserID: &userID}); err != nil {
			t.Fatal(err)
		} else if got, want := len(a), 2; got != want {
			t.Fatalf("len=%v, want %v", got, want)
		} else if got, want := a[0].SourceID, "X1"; got != want {
			t.Fatalf("[]=%v, want %v", got, want)
		} else if got, want := a[1].SourceID, "X2"; got != want {
			t.Fatalf("[]=%v, want %v", got, want)
		} else if got, want := n, 2; got != want {
			t.Fatalf("n=%v, want %v", got, want)
		}
	})
}

// MustCreateAuth creates a auth in the database. Fatal on error.
func MustCreateAuth(tb testing.TB, ctx context.Context, db *sqlite.DB, auth *wtf.Auth) (*wtf.Auth, context.Context) {
	tb.Helper()
	if err := sqlite.NewAuthService(db).CreateAuth(ctx, auth); err != nil {
		tb.Fatal(err)
	}
	return auth, wtf.NewContextWithUser(ctx, auth.User)
}
