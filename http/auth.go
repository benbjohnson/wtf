package http

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/http/html"
	"github.com/google/go-github/v32/github"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

// registerAuthRoutes is a helper function to register routes to a router.
func (s *Server) registerAuthRoutes(r *mux.Router) {
	r.HandleFunc("/login", s.handleLogin).Methods("GET")
	r.HandleFunc("/logout", s.handleLogout).Methods("DELETE")
	r.HandleFunc("/oauth/github", s.handleOAuthGitHub).Methods("GET")
	r.HandleFunc("/oauth/github/callback", s.handleOAuthGitHubCallback).Methods("GET")
}

// handleLogin handles the "GET /login" route. It simply renders an HTML login form.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var tmpl html.LoginTemplate
	tmpl.Render(r.Context(), w)
}

// handleLogout handles the "DELETE /logout" route. It clears the session
// cookie and redirects the user to the home page.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear session cookie on HTTP response.
	if err := s.setSession(w, Session{}); err != nil {
		Error(w, r, err)
		return
	}

	// Send user to the home page.
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleOAuthGitHub handles the "GET /oauth/github" route. It generates a
// random state variable and redirects the user to the GitHub OAuth endpoint.
//
// After authentication, user will be redirected back to the callback page
// where we can store the returned OAuth tokens.
func (s *Server) handleOAuthGitHub(w http.ResponseWriter, r *http.Request) {
	// Read session from request's cookies.
	session, err := s.session(r)
	if err != nil {
		Error(w, r, err)
		return
	}

	// Generate new OAuth state for the session to prevent CSRF attacks.
	state := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, state); err != nil {
		Error(w, r, err)
		return
	}
	session.State = hex.EncodeToString(state)

	// Store the state to the session in the response cookie.
	if err := s.setSession(w, session); err != nil {
		Error(w, r, err)
		return
	}

	// Redirect to OAuth2 provider.
	http.Redirect(w, r, s.OAuth2Config().AuthCodeURL(session.State), http.StatusFound)
}

// handleOAuthGitHubCallback handles the "GET /oauth/github/callback" route.
// It validates the returned OAuth state that we generated previously, looks up
// the current user's information, and creates an "Auth" object in the database.
func (s *Server) handleOAuthGitHubCallback(w http.ResponseWriter, r *http.Request) {
	// Read form variables passed in from GitHub.
	state, code := r.FormValue("state"), r.FormValue("code")

	// Read session from request.
	session, err := s.session(r)
	if err != nil {
		Error(w, r, fmt.Errorf("cannot read session: %s", err))
		return
	}

	// Validate that state matches session state.
	if state != session.State {
		Error(w, r, fmt.Errorf("oauth state mismatch"))
		return
	}

	// Exchange code for OAuth tokens.
	tok, err := s.OAuth2Config().Exchange(r.Context(), code)
	if err != nil {
		Error(w, r, fmt.Errorf("oauth exchange error: %s", err))
		return
	}

	// Create a new GitHub API client.
	client := github.NewClient(oauth2.NewClient(r.Context(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: tok.AccessToken},
	)))

	// Fetch user information for the currently authenticated user.
	u, _, err := client.Users.Get(r.Context(), "")
	if err != nil {
		Error(w, r, fmt.Errorf("cannot fetch github user: %s", err))
		return
	}

	// Email is not necessarily available for all accounts. If it is, store it
	// so we can link together multiple OAuth providers in the future
	// (e.g. GitHub, Google, etc).
	var name string
	if u.Name != nil {
		name = *u.Name
	} else if u.Login != nil {
		name = *u.Login
	}
	var email string
	if u.Email != nil {
		email = *u.Email
	}

	// Create an authentication object with an associated user.
	auth := &wtf.Auth{
		Source:       wtf.AuthSourceGitHub,
		SourceID:     strconv.FormatInt(*u.ID, 10),
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		User: &wtf.User{
			Name:  name,
			Email: email,
		},
	}
	if !tok.Expiry.IsZero() {
		auth.Expiry = &tok.Expiry
	}

	// Create the "Auth" object in the database. The AuthService will lookup
	// the user by email if they already exist. Otherwise, a new user will be
	// created and the user's ID will be set to auth.UserID.
	if err := s.AuthService.CreateAuth(r.Context(), auth); err != nil {
		Error(w, r, fmt.Errorf("cannot create auth: %s", err))
		return
	}

	// Restore redirect URL stored on login.
	redirectURL := session.RedirectURL

	// Update browser session to store the user's ID and clear OAuth state.
	session.UserID = auth.UserID
	session.RedirectURL = ""
	session.State = ""
	if err := s.setSession(w, session); err != nil {
		Error(w, r, fmt.Errorf("cannot set session cookie: %s", err))
		return
	}

	// Redirect to stored URL or, if not available, to the home page.
	if redirectURL == "" {
		redirectURL = "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
