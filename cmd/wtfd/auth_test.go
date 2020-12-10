package main_test

import (
	"context"
	"log"
	"testing"

	"github.com/chromedp/chromedp"
)

// Ensure that navigating to a page that requires authentication will redirect
// the user to the home page.
func TestRedirectToLogin(t *testing.T) {
	t.Parallel()

	// Begin running our test program.
	m := MustRunMain(t)
	defer MustCloseMain(t, m)

	// Create Chrome testing context.
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf))
	defer cancel()

	// Navigate to the home page, expect to be redirected to login.
	var title string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(m.HTTPServer.URL()),
		chromedp.WaitVisible(`body > footer`),
		chromedp.Title(&title),
	); err != nil {
		t.Fatal(err)
	} else if got, want := title, `Log in`; got != want {
		t.Fatalf("title=%q, want %q", got, want)
	}
}
