package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/benbjohnson/hashfs"
	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/http/assets"
	"github.com/benbjohnson/wtf/http/html"
)

// main is the entry point to our application binary. However, it has some poor
// usability so we mainly use it to delegate out to our run() function.
func main() {
	if err := run(context.Background(), os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run executes our program.
func run(ctx context.Context, args []string) error {
	// Our flag set is very simple. It only includes a config path.
	fs := flag.NewFlagSet("wtf-storybook", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Handle embedded asset serving. This serves files embedded from http/assets.
	http.Handle("/assets/", http.StripPrefix("/assets/", hashfs.FileServer(assets.FS)))

	// Display list of routes.
	http.Handle("/", http.HandlerFunc(handleIndex))

	// Attach all routes.
	for _, route := range routes {
		http.Handle(route.Path, route)
	}

	fmt.Println("Listening on http://localhost:3001")
	return http.ListenAndServe(":3001", nil)
}

// handleIndex renders a list of all attached routes.
func handleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, `<html>`)
	fmt.Fprintln(w, `<body>`)
	fmt.Fprintln(w, `<h2>wtf-storybook</h2>`)
	fmt.Fprintln(w, `<ul>`)
	for _, route := range routes {
		fmt.Fprintf(w, `<li><a href="%s">%s</a></li>`+"\n", route.Path, route.Name)
	}
	fmt.Fprintln(w, `</ul>`)
	fmt.Fprintln(w, `</body>`)
	fmt.Fprintln(w, `</html>`)
}

// Renderer represents an HTML template renderer found in http/html.
type Renderer interface {
	Render(ctx context.Context, w io.Writer)
}

// Route represents a named reference to a renderer.
type Route struct {
	Name     string
	Path     string
	Context  context.Context
	Renderer Renderer
}

func (r *Route) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if r.Context != nil {
		ctx = r.Context
	}
	r.Renderer.Render(ctx, w)
}

var routes = []*Route{
	// Show dial listing when user has no dials.
	{
		Name: "Dial listing with no dials available",
		Path: "/dials-with-no-data",
		Renderer: &html.DialIndexTemplate{
			Dials: []*wtf.Dial{},
		},
	},

	// Show dial with existing data.
	{
		Name: "Dial listing with data",
		Path: "/dials-with-data",
		Renderer: &html.DialIndexTemplate{
			Dials: []*wtf.Dial{
				{
					ID:        1,
					Name:      "My dial",
					Value:     20,
					User:      &wtf.User{Name: "Susy Que"},
					UpdatedAt: time.Now(),
				},
				{
					ID:        2,
					Name:      "My other dial",
					Value:     55,
					User:      &wtf.User{Name: "Jim Bob"},
					UpdatedAt: time.Now().Add(-1 * time.Hour),
				},
			},
			N:   2,
			URL: url.URL{Path: "/dials"},
		},
	},

	// Show dial listing with pagination.
	{
		Name: "Dial listing with pagination",
		Path: "/dials-pagination",
		Renderer: &html.DialIndexTemplate{
			Dials: []*wtf.Dial{
				{
					ID:        1,
					Name:      "My dial",
					User:      &wtf.User{Name: "Susy Que"},
					UpdatedAt: time.Now(),
				},
			},
			N:      20,
			Filter: wtf.DialFilter{Offset: 6, Limit: 2},
			URL:    url.URL{Path: "/dials", RawQuery: "foo=bar&baz=bat"},
		},
	},
}
