package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/csv"
	"github.com/benbjohnson/wtf/http/html"
	"github.com/gorilla/mux"
)

// registerDialRoutes is a helper function for registering all dial routes.
func (s *Server) registerDialRoutes(r *mux.Router) {
	// Listing of all dials user is a member of.
	r.HandleFunc("/dials", s.handleDialIndex).Methods("GET")

	// API endpoint for creating dials.
	r.HandleFunc("/dials", s.handleDialCreate).Methods("POST")

	// HTML form for creating dials.
	r.HandleFunc("/dials/new", s.handleDialNew).Methods("GET")
	r.HandleFunc("/dials/new", s.handleDialCreate).Methods("POST")

	// View a single dial.
	r.HandleFunc("/dials/{id}", s.handleDialView).Methods("GET")

	// HTML form for updating an existing dial.
	r.HandleFunc("/dials/{id}/edit", s.handleDialEdit).Methods("GET")
	r.HandleFunc("/dials/{id}/edit", s.handleDialUpdate).Methods("PATCH")

	// Removing a dial.
	r.HandleFunc("/dials/{id}", s.handleDialDelete).Methods("DELETE")
}

// handleDialIndex handles the "GET /dials" route. This route can optionally
// accept filter arguments and outputs a list of all dials that the current
// user is a member of.
//
// The endpoint works with HTML, JSON, & CSV formats.
func (s *Server) handleDialIndex(w http.ResponseWriter, r *http.Request) {
	// Parse optional filter object.
	var filter wtf.DialFilter
	switch r.Header.Get("Content-type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
			Error(w, r, wtf.Errorf(wtf.EINVALID, "Invalid JSON body"))
			return
		}
	default:
		// TODO: Dial pagination
	}

	// Fetch dials from database.
	dials, n, err := s.DialService.FindDials(r.Context(), filter)
	if err != nil {
		Error(w, r, err)
		return
	}

	// Render output based on HTTP accept header.
	switch r.Header.Get("Accept") {
	case "application/json":
		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(findDialsResponse{
			Dials: dials,
			N:     n,
		}); err != nil {
			LogError(r, err)
			return
		}

	case "text/csv":
		w.Header().Set("Content-type", "text/csv")
		enc := csv.NewDialEncoder(w)
		for _, dial := range dials {
			if err := enc.EncodeDial(dial); err != nil {
				LogError(r, err)
				return
			}
		}
		if err := enc.Close(); err != nil {
			LogError(r, err)
			return
		}

	default:
		tmpl := html.DialIndexTemplate{Dials: dials, N: n, Filter: filter, URL: *r.URL}
		tmpl.Render(r.Context(), w)
	}
}

// findDialsResponse represents the output JSON struct for "GET /dials".
type findDialsResponse struct {
	Dials []*wtf.Dial `json:"dials"`
	N     int         `json:"n"`
}

// handleDialView handles the "GET /dials/:id" route. It updates
func (s *Server) handleDialView(w http.ResponseWriter, r *http.Request) {
	// Parse ID from path.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, wtf.Errorf(wtf.EINVALID, "Invalid ID format"))
		return
	}

	// Fetch dial from the database.
	dial, err := s.DialService.FindDialByID(r.Context(), id)
	if err != nil {
		Error(w, r, err)
		return
	}

	// Fetch associated memberships from the database.
	dial.Memberships, _, err = s.DialMembershipService.FindDialMemberships(r.Context(), wtf.DialMembershipFilter{DialID: &dial.ID})
	if err != nil {
		Error(w, r, err)
		return
	}

	// Format returned data based on HTTP accept header.
	switch r.Header.Get("Accept") {
	case "application/json":
		w.Header().Set("Content-type", "application/json")
		if err := json.NewEncoder(w).Encode(dial); err != nil {
			LogError(r, err)
			return
		}

	default:
		tmpl := html.DialViewTemplate{
			Dial:      dial,
			InviteURL: fmt.Sprintf("%s/invite/%s", s.URL(), dial.InviteCode),
		}
		tmpl.Render(r.Context(), w)
	}
}

// handleDialNew handles the "GET /dials/new" route.
// It renders an HTML form for editing a new dial.
func (s *Server) handleDialNew(w http.ResponseWriter, r *http.Request) {
	tmpl := html.DialEditTemplate{Dial: &wtf.Dial{}}
	tmpl.Render(r.Context(), w)
}

// handleDialCreate handles the "POST /dials" and "POST /dials/new" route.
// It reads & writes data using with HTML or JSON.
func (s *Server) handleDialCreate(w http.ResponseWriter, r *http.Request) {
	// Unmarshal data based on HTTP request's content type.
	var dial wtf.Dial
	switch r.Header.Get("Content-type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(&dial); err != nil {
			Error(w, r, wtf.Errorf(wtf.EINVALID, "Invalid JSON body"))
			return
		}
	default:
		dial.Name = r.PostFormValue("name")
	}

	// Create dial in the database.
	err := s.DialService.CreateDial(r.Context(), &dial)

	// Write new dial content to response based on accept header.
	switch r.Header.Get("Accept") {
	case "application/json":
		if err != nil {
			Error(w, r, err)
			return
		}

		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(dial); err != nil {
			LogError(r, err)
			return
		}

	default:
		// If we have an internal error, display the standard error page.
		// Otherwise it's probably a validation error so we can display the
		// error on the edit page with the user's dial data that was passed in.
		if wtf.ErrorCode(err) == wtf.EINTERNAL {
			Error(w, r, err)
			return
		} else if err != nil {
			tmpl := html.DialEditTemplate{Dial: &dial, Err: err}
			tmpl.Render(r.Context(), w)
			return
		}

		// Set a message to the user and redirect to the dial's new page.
		SetFlash(w, "Dial successfully created.")
		http.Redirect(w, r, fmt.Sprintf("/dials/%d", dial.ID), http.StatusFound)
	}
}

// handleDialEdit handles the "GET /dials/:id/edit" route. This route fetches
// the underlying dial and renders it in an HTML form.
func (s *Server) handleDialEdit(w http.ResponseWriter, r *http.Request) {
	// Parse dial ID from the path.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, wtf.Errorf(wtf.EINVALID, "Invalid ID format"))
		return
	}

	// Fetch dial from the database.
	dial, err := s.DialService.FindDialByID(r.Context(), id)
	if err != nil {
		Error(w, r, err)
		return
	}

	// Render dial in the HTML form.
	tmpl := html.DialEditTemplate{Dial: dial}
	tmpl.Render(r.Context(), w)
}

// handleDialUpdate handles the "PATCH /dials/:id/edit" route. This route
// reads in the updated fields and issues an update in the database. On success,
// it redirects to the dial's view page.
func (s *Server) handleDialUpdate(w http.ResponseWriter, r *http.Request) {
	// Parse dial ID from the path.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, wtf.Errorf(wtf.EINVALID, "Invalid ID format"))
		return
	}

	// Parse fields into an update object.
	var upd wtf.DialUpdate
	name := r.PostFormValue("name")
	upd.Name = &name

	// Update the dial in the database.
	dial, err := s.DialService.UpdateDial(r.Context(), id, upd)
	if wtf.ErrorCode(err) == wtf.EINTERNAL {
		Error(w, r, err)
		return
	} else if err != nil {
		tmpl := html.DialEditTemplate{Dial: dial, Err: err}
		tmpl.Render(r.Context(), w)
		return
	}

	// Save a message to display to the user on the next page.
	// Then redirect them to the dial's view page.
	SetFlash(w, "Dial successfully updated.")
	http.Redirect(w, r, fmt.Sprintf("/dials/%d", dial.ID), http.StatusFound)
}

// handleDialDelete handles the "DELETE /dials/:id" route. This route
// permanently deletes the dial and all its members and redirects to the
// dial listing page.
func (s *Server) handleDialDelete(w http.ResponseWriter, r *http.Request) {
	// Parse dial ID from path.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, wtf.Errorf(wtf.EINVALID, "Invalid ID format"))
		return
	}

	// Delete the dial from the database.
	if err := s.DialService.DeleteDial(r.Context(), id); err != nil {
		Error(w, r, err)
		return
	}

	// Render output to the client based on HTTP accept header.
	switch r.Header.Get("Accept") {
	case "application/json":
		w.Header().Set("Content-type", "application/json")
		w.Write([]byte(`{}`))

	default:
		SetFlash(w, "Dial successfully deleted.")
		http.Redirect(w, r, "/dials", http.StatusFound)
	}
}

// DialService implements the wtf.DialService over the HTTP protocol.
type DialService struct {
	Client *Client
}

// NewDialService returns a new instance of DialService.
func NewDialService(client *Client) *DialService {
	return &DialService{Client: client}
}

// FindDialByID retrieves a single dial by ID along with associated memberships.
// Only the dial owner & members can see a dial. Returns ENOTFOUND if dial does
// not exist or user does not have permission to view it.
func (s *DialService) FindDialByID(ctx context.Context, id int) (*wtf.Dial, error) {
	// Create request with API key attached.
	req, err := s.Client.newRequest(ctx, "GET", fmt.Sprintf("/dials/%d", id), nil)
	if err != nil {
		return nil, err
	}

	// Issue request. If any other status besides 200, then treats as an error.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, parseResponseError(resp)
	}
	defer resp.Body.Close()

	// Unmarshal the returned dial data.
	var dial wtf.Dial
	if err := json.NewDecoder(resp.Body).Decode(&dial); err != nil {
		return nil, err
	}
	return &dial, nil
}

// FindDials retrieves a list of dials based on a filter. Only returns dials
// that the user owns or is a member of. Also returns a count of total matching
// dials which may different from the number of returned dials if the
// "Limit" field is set.
func (s *DialService) FindDials(ctx context.Context, filter wtf.DialFilter) ([]*wtf.Dial, int, error) {
	// Marshal filter into JSON format.
	body, err := json.Marshal(filter)
	if err != nil {
		return nil, 0, err
	}

	// Create request with API key.
	req, err := s.Client.newRequest(ctx, "GET", "/dials", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}

	// Issue request. Any non-200 status code is considered an error.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, 0, parseResponseError(resp)
	}
	defer resp.Body.Close()

	// Unmarshal result set of dials & total dial count.
	var jsonResponse findDialsResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResponse); err != nil {
		return nil, 0, err
	}
	return jsonResponse.Dials, jsonResponse.N, nil
}

// CreateDial creates a new dial and assigns the current user as the owner.
// The owner will automatically be added as a member of the new dial.
func (s *DialService) CreateDial(ctx context.Context, dial *wtf.Dial) error {
	// Marshal dial data into JSON format.
	body, err := json.Marshal(dial)
	if err != nil {
		return err
	}

	// Create request with API key.
	req, err := s.Client.newRequest(ctx, "POST", "/dials", bytes.NewReader(body))
	if err != nil {
		return err
	}

	// Issue request. Treat non-201 status codes as errors.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusCreated {
		return parseResponseError(resp)
	}
	defer resp.Body.Close()

	// Unmarshal returned dial data.
	if err := json.NewDecoder(resp.Body).Decode(&dial); err != nil {
		return err
	}
	return nil
}

// UpdateDial is not implemented by the HTTP service.
func (s *DialService) UpdateDial(ctx context.Context, id int, upd wtf.DialUpdate) (*wtf.Dial, error) {
	return nil, wtf.Errorf(wtf.ENOTIMPLEMENTED, "Not implemented.")
}

// DeleteDial permanently removes a dial by ID. Only the dial owner may delete
// a dial. Returns ENOTFOUND if dial does not exist. Returns EUNAUTHORIZED if
// user is not the dial owner.
func (s *DialService) DeleteDial(ctx context.Context, id int) error {
	// Create a request with API key.
	req, err := s.Client.newRequest(ctx, "DELETE", fmt.Sprintf("/dials/%d", id), nil)
	if err != nil {
		return err
	}

	// Issue request. Any non-200 response is considered an error.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return parseResponseError(resp)
	}
	defer resp.Body.Close()

	return nil
}

// AverageDialValueReport is not implemented by the HTTP service.
func (s *DialService) AverageDialValueReport(ctx context.Context, start, end time.Time, interval time.Duration) (*wtf.DialValueReport, error) {
	return nil, wtf.Errorf(wtf.ENOTIMPLEMENTED, "Not implemented.")
}
