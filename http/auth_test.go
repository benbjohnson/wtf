package http_test

import (
	"net/http"
	"net/url"
	"testing"

	wtfhttp "github.com/benbjohnson/wtf/http"
)

// Ensure our OAuth route redirects to the correct GitHub URL and with the
// correct OAuth parameters (e.g. client_id & state).
func TestLogin_OAuth_GitHub(t *testing.T) {
	s := MustOpenServer(t)
	defer MustCloseServer(t, s)

	// Disable redirects for testing OAuth.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Fetch our OAuth redirection URL and ensure it is redirecting us (i.e. StatusFound).
	resp, err := client.Get(s.URL() + "/oauth/github")
	if err != nil {
		t.Fatal(err)
	} else if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	} else if got, want := resp.StatusCode, http.StatusFound; got != want {
		t.Fatalf("StatusCode=%v, want %v", got, want)
	}

	// Read session from cookie & ensure a state variable is set.
	var session wtfhttp.Session
	if err := s.UnmarshalSession(resp.Cookies()[0].Value, &session); err != nil {
		t.Fatal(err)
	} else if session.State == "" {
		t.Fatal("expected oauth state in session")
	}

	// Parse location & verify that the URL is correct and that we have the
	// client ID set to our configured ID and that state matches our session state.
	if loc, err := url.Parse(resp.Header.Get("Location")); err != nil {
		t.Fatal(err)
	} else if got, want := loc.Host, `github.com`; got != want {
		t.Fatalf("Location.Host=%v, want %v", got, want)
	} else if got, want := loc.Path, `/login/oauth/authorize`; got != want {
		t.Fatalf("Location.Path=%v, want %v", got, want)
	} else if got, want := loc.Query().Get("client_id"), TestGitHubClientID; got != want {
		t.Fatalf("Location.Query.client_id=%v, want %v", got, want)
	} else if got, want := loc.Query().Get("state"), session.State; got != want {
		t.Fatalf("Location.Query.state=%v, want %v", got, want)
	}
}
