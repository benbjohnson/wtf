package main_test

import (
	"log"
	"net/url"
	"strings"
	"testing"

	"github.com/benbjohnson/wtf"
	"github.com/chromedp/chromedp"
)

// TestCreateDial creates a new dial by navigating from the dials page, clicking
// the "Create Dial" button, filling out the form, and submitting it.
func TestCreateDial(t *testing.T) {
	t.Parallel()

	// Start our test program. Defer close to clean up execution.
	m := MustRunMain(t)
	defer MustCloseMain(t, m)

	// Generate a user directly in the database and attach the user to the context.
	_, ctx0 := MustCreateUser(t, m, &wtf.User{Name: "USER0"})
	ctx0, cancel := chromedp.NewContext(ctx0, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Navigate to dial list page & click "Create new dial" button
	if err := chromedp.Run(ctx0,
		Login(ctx0, m),
		chromedp.Navigate(m.HTTPServer.URL()+`/dials`),
		chromedp.WaitVisible(`body > footer`),
		chromedp.Click(`.btn-new-dial`, chromedp.NodeVisible),
	); err != nil {
		t.Fatal(err)
	}

	// Fill out 'Create Dial' form & submit.
	if err := chromedp.Run(ctx0,
		chromedp.WaitVisible(`body > footer`),
		chromedp.SendKeys(`#name`, "NEWDIAL", chromedp.NodeVisible),
		chromedp.Submit(`#name`),
	); err != nil {
		t.Fatal(err)
	}

	// Verify that we see the dial page and the dial exists.
	var location, title, name string
	if err := chromedp.Run(ctx0,
		chromedp.WaitVisible(`body > footer`),
		chromedp.Location(&location),
		chromedp.Title(&title),
		chromedp.Text(`h2 .dial-name`, &name),
	); err != nil {
		t.Fatal(err)
	} else if u, err := url.Parse(location); err != nil {
		t.Fatal(err)
	} else if got, want := u.Path, `/dials/1`; got != want {
		t.Fatalf("location.path=%q, want %q", got, want)
	} else if got, want := title, `NEWDIAL Dial`; got != want {
		t.Fatalf("title=%q, want %q", got, want)
	} else if got, want := strings.TrimSpace(name), `NEWDIAL`; got != want {
		t.Fatalf("name=%q, want %q", got, want)
	}

	// Navigate to dial list page and verify that new dial is listed in table.
	var tableName, tableUserName, tableValue string
	if err := chromedp.Run(ctx0,
		// Return to dials page & verify dial is shown.
		chromedp.Navigate(m.HTTPServer.URL()+`/dials`),
		chromedp.WaitVisible(`body > footer`),
		chromedp.Text(`th.dial-name`, &tableName),
		chromedp.Text(`td.dial-user-name`, &tableUserName),
		chromedp.Text(`td.dial-value`, &tableValue),
	); err != nil {
		t.Fatal(err)
	} else if got, want := strings.TrimSpace(tableName), `NEWDIAL`; got != want {
		t.Fatalf("table name=%q, want %q", got, want)
	} else if got, want := strings.TrimSpace(tableUserName), `USER0`; got != want {
		t.Fatalf("table user=%q, want %q", got, want)
	} else if got, want := strings.TrimSpace(tableValue), `0`; got != want {
		t.Fatalf("table value=%q, want %q", got, want)
	}
}
