package html

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/fs"
	"net/url"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/http/assets"
)

// Alert displays an error message.
type Alert struct {
	Err error
}

func (r *Alert) Render(ctx context.Context, w io.Writer) {
	if r.Err == nil {
		return
	}

	fmt.Fprint(w, `<div class="card bg-light mb-3">`)
	fmt.Fprint(w, `<div class="card-body p-3">`)
	fmt.Fprint(w, `<p class="fs--1 mb-0 text-danger">`)
	fmt.Fprint(w, `<i class="fas fa-exclamation-circle mr-2"></i>`)
	fmt.Fprint(w, html.EscapeString(wtf.ErrorMessage(r.Err)))
	fmt.Fprint(w, `</p>`)
	fmt.Fprint(w, `</div>`)
	fmt.Fprint(w, `</div>`)
}

// Flash displays the flash message, if available.
type Flash struct{}

func (r *Flash) Render(ctx context.Context, w io.Writer) {
	s := wtf.FlashFromContext(ctx)
	if s == "" {
		return
	}

	fmt.Fprint(w, `<div class="card bg-light mb-3">`)
	fmt.Fprint(w, `<div class="card-body p-3">`)
	fmt.Fprint(w, `<p class="fs--1 mb-0">`)
	fmt.Fprint(w, `<i class="fas fa-exclamation-circle mr-2"></i>`)
	fmt.Fprint(w, html.EscapeString(s))
	fmt.Fprint(w, `</p>`)
	fmt.Fprint(w, `</div>`)
	fmt.Fprint(w, `</div>`)
}

type Pagination struct {
	URL    url.URL
	Offset int
	Limit  int
	N      int
}

func (r *Pagination) Render(ctx context.Context, w io.Writer) {
	// Do not render if no limit exists or there is only one page.
	if r.Limit == 0 || r.N <= r.Limit {
		return
	}

	// Determine the page range & current page.
	current := (r.Offset / r.Limit) + 1
	pageN := ((r.N - 1) / r.Limit) + 1

	prev := current - 1
	if prev <= 0 {
		prev = 1
	}
	next := current + 1
	if next >= pageN {
		next = pageN
	}

	// Print container & "previous" link.
	fmt.Fprint(w, `<nav aria-label="Page navigation">`)
	fmt.Fprint(w, `<ul class="pagination pagination-sm justify-content-end mb-0">`)
	fmt.Fprintf(w, `<li class="page-item"><a class="page-link" href="%s">Previous</a></li>`, r.pageURL(current-1))

	// Print a button for each page number.
	for page := 1; page <= pageN; page++ {
		className := ""
		if page == current {
			className = " active"
		}
		fmt.Fprintf(w, `<li class="page-item %s"><a class="page-link" href="%s">%d</a></li>`, className, r.pageURL(page), page)
	}

	// Print "next" link & close container.
	fmt.Fprintf(w, `<li class="page-item"><a class="page-link" href="%s">Next</a></li>`, r.pageURL(current+1))
	fmt.Fprint(w, `</ul>`)
	fmt.Fprint(w, `</nav>`)
}

func (r *Pagination) pageURL(page int) string {
	// Ensure page number is within min/max.
	pageN := ((r.N - 1) / r.Limit) + 1
	if page < 1 {
		page = 1
	} else if page > pageN {
		page = pageN
	}

	q := r.URL.Query()
	q.Set("offset", fmt.Sprint((page-1)*r.Limit))
	u := url.URL{Path: r.URL.Path, RawQuery: q.Encode()}
	return u.String()
}

type WTFBadge struct {
	DialID           int
	DialMembershipID int

	Value int
}

func (r *WTFBadge) Render(ctx context.Context, w io.Writer) {
	prefix := "bg-"
	if HasTheme {
		prefix = "badge-soft-"
	}

	var class string
	switch {
	case r.Value < 25:
		class = prefix + "success"
	case r.Value < 50:
		class = prefix + "info"
	case r.Value < 75:
		class = prefix + "warning"
	default:
		class = prefix + "danger"
	}

	fmt.Fprintf(w, `<span`)
	fmt.Fprintf(w, ` class="wtf-badge wtf-value badge rounded-pill %s"`, class)
	if r.DialID != 0 {
		fmt.Fprintf(w, ` data-dial-id="%d"`, r.DialID)
	}
	if r.DialMembershipID != 0 {
		fmt.Fprintf(w, ` data-dial-membership-id="%d"`, r.DialMembershipID)
	}
	fmt.Fprint(w, `>`)

	fmt.Fprint(w, r.Value)

	fmt.Fprint(w, `</span>`)
}

func marshalJSONTo(w io.Writer, v interface{}) {
	json.NewEncoder(w).Encode(v)
}

const ThemePath = "css/theme.css"

var HasTheme bool

func init() {
	_, err := fs.Stat(assets.FS, ThemePath)
	HasTheme = (err == nil)
}
