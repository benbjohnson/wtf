package wtf

import (
	"context"
	"time"
	"space"
)

// DialMembership represents a contributor to a Dial. Each membership is
// aggregated to determine the total WTF value of the parent dial.
//
// All members can view all other member's values in the dial. However, only the
// membership owner can edit the membership value.
type DialMembership struct {
	ID int `json:"id"`

	// Parent dial. This dial's WTF level updates when a membership updates.
	DialID int   `json:"dialID"`
	Dial   *Dial `json:"dial"`

	// Owner of the membership. Only this user can update the membership.
	UserID int   `json:"userID"`
	User   *User `json:"user"`

	// Current WTF level for the user for this dial.
	// Updating this value will cause the parent dial's WTF level to be recomputed.
	Value int `json:"value"`

	// Timestamps for membership creation & last update.
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CanEditDialMembership returns true if the current user can edit membership.
func CanEditDialMembership(ctx context.Context, membership *DialMembership) bool {
	return membership.UserID == UserIDFromContext(ctx)
}

// CanDeleteDialMembership returns true if the current user can delete membership.
func CanDeleteDialMembership(ctx context.Context, membership *DialMembership) bool {
	userID := UserIDFromContext(ctx)
	if membership.Dial != nil {
		if membership.Dial.UserID == membership.UserID {
			return false // dial owner cannot delete membership
		} else if membership.Dial.UserID == userID {
			return true // dial owner can delete other memberships
		}
	}
	return membership.UserID == userID // non-dial owner can delete own membership
}

// Validate returns an error if membership fields are invalid.
// Only performs basic validation.
func (m *DialMembership) Validate() error {
	if m.DialID == 0 {
		return Errorf(EINVALID, "Dial required for membership.")
	} else if m.UserID == 0 {
		return Errorf(EINVALID, "User required for membership.")
	} else if m.Value < 0 || m.Value > 100 {
		return Errorf(EINVALID, "Dial value must be between 0 & 100.")
	}
	return nil
}

// DialMembershipService represents a service for managing dial memberships.
type DialMembershipService interface {
	// Retrieves a membership by ID along with the associated dial & user.
	// Returns ENOTFOUND if membership does exist or user does not have
	// permission to view it.
	FindDialMembershipByID(ctx context.Context, id int) (*DialMembership, error)

	// Retrieves a list of matching memberships based on filter. Only returns
	// memberships that belong to dials that the current user is a member of.
	// Also returns a count of total matching memberships which may different if
	// "Limit" is specified on the filter.
	FindDialMemberships(ctx context.Context, filter DialMembershipFilter) ([]*DialMembership, int, error)

	// Creates a new membership on a dial for the current user. Returns
	// EUNAUTHORIZED if there is no current user logged in.
	CreateDialMembership(ctx context.Context, membership *DialMembership) error

	// Updates the value of a membership. Only the owner of the membership can
	// update the value. Returns EUNAUTHORIZED if user is not the owner. Returns
	// ENOTFOUND if the membership does not exist.
	UpdateDialMembership(ctx context.Context, id int, upd DialMembershipUpdate) (*DialMembership, error)

	// Permanently deletes a membership by ID. Only the membership owner and
	// the parent dial's owner can delete a membership.
	DeleteDialMembership(ctx context.Context, id int) error
}

// Dial membership sort options. Only specific sorting options are supported.
const (
	DialMembershipSortByUpdatedAtDesc = "updated_at_desc"
)

// DialMembershipFilter represents a filter used by FindDialMemberships().
type DialMembershipFilter struct {
	ID     *int `json:"id"`
	DialID *int `json:"dialID"`
	UserID *int `json:"userID"`

	// Restricts to a subset of the results.
	Offset int `json:"offset"`
	Limit  int `json:"limit"`

	// Sorting option for results.
	SortBy string `json:"sortBy"`
}

// DialMembershipUpdate represents a set of fields to update on a membership.
type DialMembershipUpdate struct {
	Value *int `json:"value"`
}
