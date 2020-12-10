package main_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/cmd/wtfd"
	"github.com/benbjohnson/wtf/http"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	// "github.com/chromedp/cdproto/cdp"
)

// MustRunMain is a test helper function that executes Main in a temporary path.
// The HTTP server binds to ":0" so it will start on a random port. This allows
// our end-to-end tests to be run in parallel. Fail on error.
func MustRunMain(tb testing.TB) *main.Main {
	tb.Helper()

	m := main.NewMain()
	m.Config.DSN = filepath.Join(tb.TempDir(), "db")
	m.Config.HTTP.Addr = ":0"
	m.Config.GitHub.ClientID = strings.Repeat("00", 10)
	m.Config.GitHub.ClientSecret = strings.Repeat("00", 20)
	m.Config.HTTP.HashKey = strings.Repeat("00", 64)
	m.Config.HTTP.BlockKey = strings.Repeat("00", 32)

	if err := m.Run(context.Background()); err != nil {
		tb.Fatal(err)
	}
	return m
}

// MustCloseMain closes the program. Fail on error.
func MustCloseMain(tb testing.TB, m *main.Main) {
	tb.Helper()
	if err := m.Close(); err != nil {
		tb.Fatal(err)
	}
}

// MustCreateUser is a test helper for creating a new user in the system by
// calling the underlying DB service directly.
func MustCreateUser(tb testing.TB, m *main.Main, user *wtf.User) (*wtf.User, context.Context) {
	tb.Helper()
	if err := m.UserService.CreateUser(context.Background(), user); err != nil {
		tb.Fatal(err)
	}
	return user, wtf.NewContextWithUser(context.Background(), user)
}

// Login returns a Chrome action that generates a secure cookie and attaches it
// to the browser. This approach is used to avoid OAuth communication with GitHub.
func Login(ctx context.Context, m *main.Main) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Generate cookie value from the server.
		value, err := m.HTTPServer.MarshalSession(http.Session{
			UserID: wtf.UserIDFromContext(ctx),
		})
		if err != nil {
			return err
		}

		// Add cookie to browser.
		if ok, err := network.SetCookie(http.SessionCookieName, value).WithDomain("localhost").Do(ctx); err != nil {
			return err
		} else if !ok {
			return fmt.Errorf("could not set session cookie")
		}
		return nil
	})
}
