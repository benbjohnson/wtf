package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/http/html"
	"github.com/gorilla/mux"
)

// registerDialMembershipRoutes is a helper function for registering membership routes.
func (s *Server) registerDialMembershipRoutes(r *mux.Router) {
	// Create membership via invite code.
	r.HandleFunc("/invite/{code}", s.handleDialMembershipNew).Methods("GET")
	r.HandleFunc("/invite/{code}", s.handleDialMembershipCreate).Methods("POST")

	// Update membership WTF level.
	r.HandleFunc("/dial-memberships/{id}", s.handleDialMembershipUpdate).Methods("PATCH")

	// Remove membership.
	r.HandleFunc("/dial-memberships/{id}", s.handleDialMembershipDelete).Methods("DELETE")
}

// handleDialMembershipNew handles the "GET /invite/:code" route. This route
// uses the dial's invite code to allow users to join an existing dial.
func (s *Server) handleDialMembershipNew(w http.ResponseWriter, r *http.Request) {
	// Read user ID for currently logged in user.
	userID := wtf.UserIDFromContext(r.Context())

	// Read invite code from the URL path.
	code := mux.Vars(r)["code"]

	// Find dial by invite code.
	// The invite code is unique so at most one dial will be returned.
	dials, _, err := s.DialService.FindDials(r.Context(), wtf.DialFilter{InviteCode: &code})
	if err != nil {
		Error(w, r, err)
		return
	} else if len(dials) == 0 {
		Error(w, r, wtf.Errorf(wtf.ENOTFOUND, "Invalid invitation URL."))
		return
	}

	// Check if user is already a member. If so, redirect them to the dial's
	// page automatically and add a flash message letting them know.
	if memberships, _, err := s.DialMembershipService.FindDialMemberships(r.Context(), wtf.DialMembershipFilter{
		DialID: &dials[0].ID,
		UserID: &userID,
	}); err != nil {
		Error(w, r, err)
		return
	} else if len(memberships) != 0 {
		SetFlash(w, "You are already a member of this dial.")
		http.Redirect(w, r, fmt.Sprintf("/dials/%d", memberships[0].DialID), http.StatusFound)
		return
	}

	// Render HTML page asking user to confirm they want to join the dial.
	tmpl := html.DialMembershipCreateTemplate{Dial: dials[0]}
	tmpl.Render(r.Context(), w)
}

// handleDialMembershipCreate handles the "POST /invite/:code" route.
// This route adds a new membership for the current user to a dial.
func (s *Server) handleDialMembershipCreate(w http.ResponseWriter, r *http.Request) {
	// Read user ID for currently logged in user.
	userID := wtf.UserIDFromContext(r.Context())

	// Read invite code from URL path.
	code := mux.Vars(r)["code"]

	// Look up dial by invite code.
	dials, _, err := s.DialService.FindDials(r.Context(), wtf.DialFilter{InviteCode: &code})
	if err != nil {
		Error(w, r, err)
		return
	} else if len(dials) == 0 {
		Error(w, r, wtf.Errorf(wtf.ENOTFOUND, "Invalid invitation URL."))
		return
	}

	// Create a new membership between the current user and the dial associated
	// with the invite code.
	membership := &wtf.DialMembership{
		DialID: dials[0].ID,
		UserID: userID,
	}
	if err := s.DialMembershipService.CreateDialMembership(r.Context(), membership); err != nil {
		Error(w, r, err)
		return
	}

	// Let the user know they've joined the dial and then redirect them to the
	// dial's page.
	SetFlash(w, fmt.Sprintf("You have now joined the %q dial.", membership.Dial.Name))
	http.Redirect(w, r, fmt.Sprintf("/dials/%d", membership.DialID), http.StatusFound)
}

// handleDialMembershipUpdate handles the "PATCH /dial-memberships/:id" route.
// This route is only called via JSON API on the dial view page.
func (s *Server) handleDialMembershipUpdate(w http.ResponseWriter, r *http.Request) {
	// Parse membership ID from URL path.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, wtf.Errorf(wtf.EINVALID, "Invalid ID format"))
		return
	}

	// Force application/json output.
	r.Header.Set("Accept", "application/json")

	// Parse update object from JSON request body.
	var upd wtf.DialMembershipUpdate
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		Error(w, r, wtf.Errorf(wtf.EINVALID, "Invalid JSON body"))
		return
	}

	// Update membership.
	membership, err := s.DialMembershipService.UpdateDialMembership(r.Context(), id, upd)
	if err != nil {
		Error(w, r, err)
		return
	}

	// Write new membership state back as JSON response.
	w.Header().Set("Content-type", "application/json")
	if err := json.NewEncoder(w).Encode(membership); err != nil {
		LogError(r, err)
		return
	}
}

// handleDialMembershipDelete handles the "DELETE /dial-memberships/:id" route.
// This route deletes the given membership and redirects the user.
func (s *Server) handleDialMembershipDelete(w http.ResponseWriter, r *http.Request) {
	// Parse membership ID from the URL.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, wtf.Errorf(wtf.EINVALID, "Invalid ID format"))
		return
	}

	// Look up membership by ID.
	membership, err := s.DialMembershipService.FindDialMembershipByID(r.Context(), id)
	if err != nil {
		Error(w, r, err)
		return
	}

	// Delete membership.
	if err := s.DialMembershipService.DeleteDialMembership(r.Context(), id); err != nil {
		Error(w, r, err)
		return
	}

	// Let user know the membership has been deleted.
	SetFlash(w, "Dial membership successfully deleted.")

	// If user is the owner then redirect back to the dial's view page. However,
	// if user was just a member then they won't be able to see the dial anymore
	// so redirect them to the home page.
	if membership.Dial.UserID == wtf.UserIDFromContext(r.Context()) {
		http.Redirect(w, r, fmt.Sprintf("/dials/%d", membership.DialID), http.StatusFound)
	} else {
		http.Redirect(w, r, "/dials", http.StatusFound)
	}
}
