package http_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/benbjohnson/wtf"
	wtfhttp "github.com/benbjohnson/wtf/http"
	"github.com/google/go-cmp/cmp"
)

// Ensure the HTTP server can return the dial listing in a variety of formats.
func TestDialIndex(t *testing.T) {
	// Start the mocked HTTP test server.
	s := MustOpenServer(t)
	defer MustCloseServer(t, s)

	// Create a single user and build a context with them.
	user0 := &wtf.User{ID: 1, Name: "USER1", APIKey: "APIKEY"}
	ctx0 := wtf.NewContextWithUser(context.Background(), user0)

	// Mock dial data.
	dial := &wtf.Dial{
		ID:         1,
		UserID:     1,
		User:       &wtf.User{ID: 1, Name: "USER1"},
		Name:       "DIAL1",
		Value:      50,
		InviteCode: "INVITECODE",
		CreatedAt:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}

	// Mock the fetch of dials.
	s.DialService.FindDialsFn = func(ctx context.Context, filter wtf.DialFilter) ([]*wtf.Dial, int, error) {
		return []*wtf.Dial{dial}, 1, nil
	}

	// Ensure server can generate HTML output.
	t.Run("HTML", func(t *testing.T) {
		// Mock user look up by ID for loading session data.
		s.UserService.FindUserByIDFn = func(ctx context.Context, id int) (*wtf.User, error) {
			return user0, nil
		}

		// Issue request with client session data.
		resp, err := http.DefaultClient.Do(s.MustNewRequest(t, ctx0, "GET", "/dials", nil))
		if err != nil {
			t.Fatal(err)
		} else if got, want := resp.StatusCode, http.StatusOK; got != want {
			t.Fatalf("StatusCode=%v, want %v", got, want)
		}

		// Load the HTML document using goquery so we can validate HTML nodes using CSS selectors.
		// This validates the title of the HTML page as well as some cells in the first row.
		if doc, err := goquery.NewDocumentFromReader(resp.Body); err != nil {
			t.Fatal(err)
		} else if got, want := strings.TrimSpace(doc.Find("title").Text()), `Your Dials`; got != want {
			t.Fatalf("title=%q, want %q", got, want)
		} else if got, want := strings.TrimSpace(doc.Find(".table-dials tbody th.dial-name").Text()), `DIAL1`; got != want {
			t.Fatalf("name=%q, want %q", got, want)
		} else if got, want := strings.TrimSpace(doc.Find(".table-dials tbody td.dial-user-name").Text()), `USER1`; got != want {
			t.Fatalf("user=%q, want %q", got, want)
		}
	})

	// Ensure server can generate JSON output.
	t.Run("JSON", func(t *testing.T) {
		// Mock user look up by API key for API calls.
		s.UserService.FindUsersFn = func(ctx context.Context, filter wtf.UserFilter) ([]*wtf.User, int, error) {
			if filter.APIKey == nil || *filter.APIKey != "APIKEY" {
				t.Fatalf("unexpected api key: %#v", filter.APIKey)
			}
			return []*wtf.User{user0}, 1, nil
		}

		// Instantiate HTTP service and fetch dials.
		dialService := wtfhttp.NewDialService(wtfhttp.NewClient(s.URL()))
		if dials, n, err := dialService.FindDials(ctx0, wtf.DialFilter{}); err != nil {
			t.Fatal(err)
		} else if got, want := len(dials), 1; got != want {
			t.Fatalf("len(dials)=%d, want %d", got, want)
		} else if diff := cmp.Diff(dials[0], dial); diff != "" {
			t.Fatal(diff)
		} else if got, want := n, 1; got != want {
			t.Fatalf("n=%d, want %d", got, want)
		}
	})
}
