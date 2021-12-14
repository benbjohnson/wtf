package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/benbjohnson/wtf"
)

// DialMembershipService represents a service for managing dial memberships in SQLite.
type DialMembershipService struct {
	db *DB
}

// NewDialMembershipService returns a new instance of DialMembershipService.
func NewDialMembershipService(db *DB) *DialMembershipService {
	return &DialMembershipService{db: db}
}

// FindDialMembershipByID retrieves a membership by ID along with the associated
// dial & user. Returns ENOTFOUND if membership does exist or user does not have
// permission to view it.
func (s *DialMembershipService) FindDialMembershipByID(ctx context.Context, id int) (*wtf.DialMembership, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Fetch membership object by ID and attach associated user & dial.
	membership, err := findDialMembershipByID(ctx, tx, id)
	if err != nil {
		return nil, err
	} else if err := attachDialMembershipAssociations(ctx, tx, membership); err != nil {
		return nil, err
	}
	return membership, nil
}

// FindDialMemberships retrieves a list of matching memberships based on filter.
// Only returns memberships that belong to dials that the current user is a member of.
//
// Also returns a count of total matching memberships which may different if
// "Limit" is specified on the filter.
func (s *DialMembershipService) FindDialMemberships(ctx context.Context, filter wtf.DialMembershipFilter) ([]*wtf.DialMembership, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	// Fetch a list of matching membership objects.
	memberships, n, err := findDialMemberships(ctx, tx, filter)
	if err != nil {
		return memberships, n, err
	}

	// Attach dial & user to each returned membership.
	// This should be batched up if you were to use a remote database server.
	for _, membership := range memberships {
		if err := attachDialMembershipAssociations(ctx, tx, membership); err != nil {
			return memberships, n, err
		}
	}
	return memberships, n, nil
}

// CreateDialMembership creates a new membership on a dial for the current user.
// Returns EUNAUTHORIZED if there is no current user logged in.
func (s *DialMembershipService) CreateDialMembership(ctx context.Context, membership *wtf.DialMembership) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ensure user is logged in & assign membership to current user.
	userID := wtf.UserIDFromContext(ctx)
	if userID == 0 {
		return wtf.Errorf(wtf.EUNAUTHORIZED, "You must be logged in to join a dial.")
	}
	membership.UserID = wtf.UserIDFromContext(ctx)

	// Create new membership and attach associated user & dial to returned data.
	if err := createDialMembership(ctx, tx, membership); err != nil {
		return err
	} else if err := attachDialMembershipAssociations(ctx, tx, membership); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateDialMembership updates the value of a membership. Only the owner of
// the membership can update the value. Returns EUNAUTHORIZED if user is not the
// owner. Returns ENOTFOUND if the membership does not exist.
func (s *DialMembershipService) UpdateDialMembership(ctx context.Context, id int, upd wtf.DialMembershipUpdate) (*wtf.DialMembership, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Update a membership and attach associated user & dial to returned data.
	membership, err := updateDialMembership(ctx, tx, id, upd)
	if err != nil {
		return membership, err
	} else if err := attachDialMembershipAssociations(ctx, tx, membership); err != nil {
		return membership, err
	}
	return membership, tx.Commit()
}

// DeleteDialMembership permanently deletes a membership by ID. Only the
// membership owner and the parent dial's owner can delete a membership.
func (s *DialMembershipService) DeleteDialMembership(ctx context.Context, id int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := deleteDialMembership(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// findDialMembershipByID returns a membership object by ID.
// Returns ENOTFOUND if membership does not exist.
func findDialMembershipByID(ctx context.Context, tx *Tx, id int) (*wtf.DialMembership, error) {
	memberships, _, err := findDialMemberships(ctx, tx, wtf.DialMembershipFilter{ID: &id})
	if err != nil {
		return nil, err
	} else if len(memberships) == 0 {
		return nil, &wtf.Error{Code: wtf.ENOTFOUND, Message: "Dial membership not found."}
	}
	return memberships[0], nil
}

func findDialMemberships(ctx context.Context, tx *Tx, filter wtf.DialMembershipFilter) (_ []*wtf.DialMembership, n int, err error) {
	// Build WHERE clause. Each segment of the clause is AND-ed together.
	// Values are appended to args so we can avoid SQL injection.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "dm.id = ?"), append(args, *v)
	}
	if v := filter.DialID; v != nil {
		where, args = append(where, "dm.dial_id = ?"), append(args, *v)
	}
	if v := filter.UserID; v != nil {
		where, args = append(where, "dm.user_id = ?"), append(args, *v)
	}

	// Limit to user's memberships or memberships of dials they belong to.
	userID := wtf.UserIDFromContext(ctx)
	where = append(where, `(
		d.user_id = ? OR
		dm.dial_id IN (SELECT dm1.dial_id FROM dial_memberships dm1 WHERE dm1.user_id = ?)
	)`)
	args = append(args, userID, userID, userID)

	// Determine sorting.
	var sortBy string
	switch filter.SortBy {
	case wtf.DialMembershipSortByUpdatedAtDesc:
		sortBy = "dm.updated_at DESC"
	default:
		// Sort current user's membership first and then order by user name.
		sortBy = `CASE dm.user_id WHEN ? THEN 0 ELSE 1 END ASC, u.name ASC`
		args = append(args, userID)
	}

	// Query for all matching membership rows.
	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    dm.id,
		    dm.dial_id,
		    dm.user_id,
		    dm.value,
		    dm.created_at,
		    dm.updated_at,
		    d.user_id AS dial_user_id,
		    COUNT(*) OVER()
		FROM dial_memberships dm
		INNER JOIN dials d ON dm.dial_id = d.id
		INNER JOIN users u ON dm.user_id = u.id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY `+sortBy+`
		`+FormatLimitOffset(filter.Limit, filter.Offset),
		args...,
	)
	if err != nil {
		return nil, n, FormatError(err)
	}
	defer rows.Close()

	// Iterate over rows and deserialized into DialMembership objects.
	memberships := make([]*wtf.DialMembership, 0)
	for rows.Next() {
		var dialUserID int
		var membership wtf.DialMembership
		if err := rows.Scan(
			&membership.ID,
			&membership.DialID,
			&membership.UserID,
			&membership.Value,
			(*NullTime)(&membership.CreatedAt),
			(*NullTime)(&membership.UpdatedAt),
			&dialUserID,
			&n,
		); err != nil {
			return nil, 0, err
		}

		memberships = append(memberships, &membership)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return memberships, n, nil
}

// createDialMembership creates a new membership. Assigns the new database ID
// to membership.ID and updates the timestamps.
func createDialMembership(ctx context.Context, tx *Tx, membership *wtf.DialMembership) error {
	// Update timestamps to current time.
	membership.CreatedAt = tx.now
	membership.UpdatedAt = membership.CreatedAt

	// Perform basic field validation.
	if err := membership.Validate(); err != nil {
		return err
	}

	// Lookup dial & user to ensure they exist.
	//
	// We need a special function for the dial as we have permissions checks
	// we need to avoid since we are not yet a member. Normally we could use
	// FOREIGN KEY errors to report a non-existent dial but SQLite FOREIGN KEY
	// errors are not descriptive enough.
	if err := checkDialExists(ctx, tx, membership.DialID); err != nil {
		return err
	} else if _, err := findUserByID(ctx, tx, membership.UserID); err != nil {
		return err
	}

	// Execute query to insert membership.
	result, err := tx.ExecContext(ctx, `
		INSERT INTO dial_memberships (
			dial_id,
			user_id,
			value,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		membership.DialID,
		membership.UserID,
		membership.Value,
		(*NullTime)(&membership.CreatedAt),
		(*NullTime)(&membership.UpdatedAt),
	)
	if err != nil {
		return FormatError(err)
	}

	// Assign new database ID to the caller's arg.
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	membership.ID = int(id)

	// Ensure computed parent dial value is up to date.
	if err := refreshDialValue(ctx, tx, membership.DialID); err != nil {
		return fmt.Errorf("refresh dial value: %w", err)
	}

	return nil
}

// updateDialMembership updates the value of a membership.
// Returns EUNAUTHORIZED if user is not the membership owner.
func updateDialMembership(ctx context.Context, tx *Tx, id int, upd wtf.DialMembershipUpdate) (*wtf.DialMembership, error) {
	// Fetch current object state. Return error if current user is not owner.
	membership, err := findDialMembershipByID(ctx, tx, id)
	if err != nil {
		return membership, err
	} else if membership.UserID != wtf.UserIDFromContext(ctx) {
		return membership, wtf.Errorf(wtf.EUNAUTHORIZED, "You do not have permission to update the dial membership.")
	}

	// Save state of membership to compare later in the function.
	prev := *membership

	// Update fields.
	if v := upd.Value; v != nil {
		membership.Value = *v
	}

	// Exit if membership did not change.
	if prev.Value == membership.Value {
		return membership, nil
	}

	// Set last updated date to current time.
	membership.UpdatedAt = tx.now

	// Perform basic field validation.
	if err := membership.Validate(); err != nil {
		return membership, err
	}

	// Execute query to update membership value.
	if _, err := tx.ExecContext(ctx, `
		UPDATE dial_memberships
		SET value = ?,
		    updated_at = ?
		WHERE id = ?
	`,
		membership.Value,
		(*NullTime)(&membership.UpdatedAt),
		id,
	); err != nil {
		return membership, FormatError(err)
	}

	// Ensure computed dial value is up to date.
	if err := refreshDialValue(ctx, tx, membership.DialID); err != nil {
		return membership, fmt.Errorf("refresh dial value: %w", err)
	}

	// Publish event to all dial members.
	if err := publishDialEvent(ctx, tx, membership.DialID, wtf.Event{
		Type: wtf.EventTypeDialMembershipValueChanged,
		Payload: &wtf.DialMembershipValueChangedPayload{
			ID:    id,
			Value: membership.Value,
		},
	}); err != nil {
		return membership, fmt.Errorf("publish dial event: %w", err)
	}

	return membership, nil
}

// deleteDialMembership permanently deletes a membership and updates the dial value.
func deleteDialMembership(ctx context.Context, tx *Tx, id int) error {
	// Fetch user ID of currently logged in user.
	userID := wtf.UserIDFromContext(ctx)

	// Verify object exists and fetch associations.
	membership, err := findDialMembershipByID(ctx, tx, id)
	if err != nil {
		return err
	} else if err := attachDialMembershipAssociations(ctx, tx, membership); err != nil {
		return err
	}

	// Verify user owns membership or parent dial.
	if membership.UserID != userID && membership.Dial.UserID != userID {
		return wtf.Errorf(wtf.EUNAUTHORIZED, "You do not have permission to delete the dial membership.")
	}

	// Do not allow dial owner to delete their own membership.
	if membership.UserID == membership.Dial.UserID {
		return wtf.Errorf(wtf.ECONFLICT, "Dial owner may not delete their own membership.")
	}

	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM dial_memberships WHERE id = ?`, id); err != nil {
		return FormatError(err)
	}

	// Ensure computed dial value is up to date.
	if err := refreshDialValue(ctx, tx, membership.DialID); err != nil {
		return fmt.Errorf("refresh dial value: %w", err)
	}
	return nil
}

func attachDialMembershipAssociations(ctx context.Context, tx *Tx, membership *wtf.DialMembership) (err error) {
	if membership.Dial, err = findDialByID(ctx, tx, membership.DialID); err != nil {
		return fmt.Errorf("attach membership dial: %w", err)
	} else if membership.User, err = findUserByID(ctx, tx, membership.UserID); err != nil {
		return fmt.Errorf("attach membership user: %w", err)
	}
	return nil
}
