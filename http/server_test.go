package http_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/benbjohnson/wtf"
	wtfhttp "github.com/benbjohnson/wtf/http"
	"github.com/benbjohnson/wtf/mock"
)

// Default configuration settings for the test server.
const (
	TestHashKey            = "0000000000000000"
	TestBlockKey           = "00000000000000000000000000000000"
	TestGitHubClientID     = "00000000000000000000"
	TestGitHubClientSecret = "0000000000000000000000000000000000000000"
)

// Server represents a test wrapper for wtfhttp.Server.
// It attaches mocks to the server & initializes on a random port.
type Server struct {
	*wtfhttp.Server

	// Mock services.
	AuthService           mock.AuthService
	DialService           mock.DialService
	DialMembershipService mock.DialMembershipService
	EventService          mock.EventService
	UserService           mock.UserService
}

// MustOpenServer is a test helper function for starting a new test HTTP server.
// Fail on error.
func MustOpenServer(tb testing.TB) *Server {
	tb.Helper()

	// Initialize wrapper and set test configuration settings.
	s := &Server{Server: wtfhttp.NewServer()}
	s.HashKey = TestHashKey
	s.BlockKey = TestBlockKey
	s.GitHubClientID = TestGitHubClientID
	s.GitHubClientSecret = TestGitHubClientSecret

	// Assign mocks to actual server's services.
	s.Server.AuthService = &s.AuthService
	s.Server.DialService = &s.DialService
	s.Server.DialMembershipService = &s.DialMembershipService
	s.Server.EventService = &s.EventService
	s.Server.UserService = &s.UserService

	// Begin running test server.
	if err := s.Open(); err != nil {
		tb.Fatal(err)
	}
	return s
}

// MustCloseServer is a test helper function for shutting down the server.
// Fail on error.
func MustCloseServer(tb testing.TB, s *Server) {
	tb.Helper()
	if err := s.Close(); err != nil {
		tb.Fatal(err)
	}
}

// MustNewRequest creates a new HTTP request using the server's base URL and
// attaching a user session based on the context.
func (s *Server) MustNewRequest(tb testing.TB, ctx context.Context, method, url string, body io.Reader) *http.Request {
	tb.Helper()

	// Create new net/http request with server's base URL.
	r, err := http.NewRequest(method, s.URL()+url, body)
	if err != nil {
		tb.Fatal(err)
	}

	// Generate session cookie for user, if logged in.
	if user := wtf.UserFromContext(ctx); user != nil {
		data, err := s.MarshalSession(wtfhttp.Session{UserID: user.ID})
		if err != nil {
			tb.Fatal(err)
		}
		r.AddCookie(&http.Cookie{
			Name:  wtfhttp.SessionCookieName,
			Value: data,
			Path:  "/",
		})
	}

	return r
}
