package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/benbjohnson/wtf"
)

// AuthService represents a service for managing OAuth authentication.
type AuthService struct {
	db *DB
}

// NewAuthService returns a new instance of AuthService attached to DB.
func NewAuthService(db *DB) *AuthService {
	return &AuthService{db: db}
}

// FindAuthByID retrieves an authentication object by ID along with the associated user.
// Returns ENOTFOUND if ID does not exist.
func (s *AuthService) FindAuthByID(ctx context.Context, id int) (*wtf.Auth, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Look up auth by ID and read associated user object.
	auth, err := findAuthByID(ctx, tx, id)
	if err != nil {
		return nil, err
	} else if err := attachAuthAssociations(ctx, tx, auth); err != nil {
		return nil, err
	}

	return auth, nil
}

// FindAuths retrieves authentication objects based on a filter.
//
// Also returns the total number of objects that match the filter. This may
// differ from the returned object count if the Limit field is set.
func (s *AuthService) FindAuths(ctx context.Context, filter wtf.AuthFilter) ([]*wtf.Auth, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	// Fetch the individual authentication objects from the database.
	auths, n, err := findAuths(ctx, tx, filter)
	if err != nil {
		return auths, n, err
	}

	// Iterate over returned objects and attach user objects.
	// This works well for SQLite because it is in-process but remote database
	// servers will incur a high per-query latency so queries should be batched.
	for _, auth := range auths {
		if err := attachAuthAssociations(ctx, tx, auth); err != nil {
			return auths, n, err
		}
	}
	return auths, n, nil
}

// CreateAuth Creates a new authentication object If a User is attached to auth,
// then the auth object is linked to an existing user. Otherwise a new user
// object is created.
//
// On success, the auth.ID is set to the new authentication ID.
func (s *AuthService) CreateAuth(ctx context.Context, auth *wtf.Auth) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check to see if the auth already exists for the given source.
	if other, err := findAuthBySourceID(ctx, tx, auth.Source, auth.SourceID); err == nil {
		// If an auth already exists for the source user, update with the new tokens.
		if other, err = updateAuth(ctx, tx, other.ID, auth.AccessToken, auth.RefreshToken, auth.Expiry); err != nil {
			return fmt.Errorf("cannot update auth: id=%d err=%w", other.ID, err)
		} else if err := attachAuthAssociations(ctx, tx, other); err != nil {
			return err
		}

		// Copy found auth back to the caller's arg & return.
		*auth = *other
		return tx.Commit()
	} else if wtf.ErrorCode(err) != wtf.ENOTFOUND {
		return fmt.Errorf("canot find auth by source user: %w", err)
	}

	// Check if auth has a new user object passed in. It is considered "new" if
	// the caller doesn't know the database ID for the user.
	if auth.UserID == 0 && auth.User != nil {
		// Look up the user by email address. If no user can be found then
		// create a new user with the auth.User object passed in.
		if user, err := findUserByEmail(ctx, tx, auth.User.Email); err == nil { // user exists
			auth.User = user
		} else if wtf.ErrorCode(err) == wtf.ENOTFOUND { // user does not exist
			if err := createUser(ctx, tx, auth.User); err != nil {
				return fmt.Errorf("cannot create user: %w", err)
			}
		} else {
			return fmt.Errorf("cannot find user by email: %w", err)
		}

		// Assign the created/found user ID back to the auth object.
		auth.UserID = auth.User.ID
	}

	// Create new auth object & attach associated user.
	if err := createAuth(ctx, tx, auth); err != nil {
		return err
	} else if err := attachAuthAssociations(ctx, tx, auth); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteAuth permanently deletes an authentication object from the system by ID.
// The parent user object is not removed.
func (s *AuthService) DeleteAuth(ctx context.Context, id int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := deleteAuth(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// findAuthByID is a helper function to return an auth object by ID.
// Returns ENOTFOUND if auth doesn't exist.
func findAuthByID(ctx context.Context, tx *Tx, id int) (*wtf.Auth, error) {
	auths, _, err := findAuths(ctx, tx, wtf.AuthFilter{ID: &id})
	if err != nil {
		return nil, err
	} else if len(auths) == 0 {
		return nil, &wtf.Error{Code: wtf.ENOTFOUND, Message: "Auth not found."}
	}
	return auths[0], nil
}

// findAuthBySourceID is a helper function to return an auth object by source ID.
// Returns ENOTFOUND if auth doesn't exist.
func findAuthBySourceID(ctx context.Context, tx *Tx, source, sourceID string) (*wtf.Auth, error) {
	auths, _, err := findAuths(ctx, tx, wtf.AuthFilter{Source: &source, SourceID: &sourceID})
	if err != nil {
		return nil, err
	} else if len(auths) == 0 {
		return nil, &wtf.Error{Code: wtf.ENOTFOUND, Message: "Auth not found."}
	}
	return auths[0], nil
}

// findAuths returns a list of auth objects that match a filter. Also returns
// a total count of matches which may differ from results if filter.Limit is set.
func findAuths(ctx context.Context, tx *Tx, filter wtf.AuthFilter) (_ []*wtf.Auth, n int, err error) {
	// Build WHERE clause. Each part of the clause is AND-ed together to further
	// restrict the results. Placeholders are added to "args" and are used
	// to avoid SQL injection.
	//
	// Each filter field is optional.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := filter.UserID; v != nil {
		where, args = append(where, "user_id = ?"), append(args, *v)
	}
	if v := filter.Source; v != nil {
		where, args = append(where, "source = ?"), append(args, *v)
	}
	if v := filter.SourceID; v != nil {
		where, args = append(where, "source_id = ?"), append(args, *v)
	}

	// Execute the query with WHERE clause and LIMIT/OFFSET injected.
	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
		    user_id,
		    source,
		    source_id,
		    access_token,
		    refresh_token,
		    expiry,
		    created_at,
		    updated_at,
		    COUNT(*) OVER()
		FROM auths
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY id ASC
		`+FormatLimitOffset(filter.Limit, filter.Offset)+`
	`,
		args...,
	)
	if err != nil {
		return nil, n, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into Auth objects.
	auths := make([]*wtf.Auth, 0)
	for rows.Next() {
		var auth wtf.Auth
		var expiry sql.NullString
		if err := rows.Scan(
			&auth.ID,
			&auth.UserID,
			&auth.Source,
			&auth.SourceID,
			&auth.AccessToken,
			&auth.RefreshToken,
			&expiry,
			(*NullTime)(&auth.CreatedAt),
			(*NullTime)(&auth.UpdatedAt),
			&n,
		); err != nil {
			return nil, 0, err
		}

		if expiry.Valid {
			if v, _ := time.Parse(time.RFC3339, expiry.String); !v.IsZero() {
				auth.Expiry = &v
			}
		}

		auths = append(auths, &auth)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, FormatError(err)
	}

	return auths, n, nil
}

// createAuth creates a new auth object in the database. On success, the
// ID is set to the new database ID & timestamp fields are set to the current time.
func createAuth(ctx context.Context, tx *Tx, auth *wtf.Auth) error {
	// Set timestamp fields to current time.
	auth.CreatedAt = tx.now
	auth.UpdatedAt = auth.CreatedAt

	// Ensure auth object passes basic validation.
	if err := auth.Validate(); err != nil {
		return err
	}

	// Convert expiry date to RFC 3339 for SQLite.
	var expiry *string
	if auth.Expiry != nil {
		tmp := auth.Expiry.Format(time.RFC3339)
		expiry = &tmp
	}

	// Execute insertion query.
	result, err := tx.ExecContext(ctx, `
		INSERT INTO auths (
			user_id,
			source,
			source_id,
			access_token,
			refresh_token,
			expiry,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		auth.UserID,
		auth.Source,
		auth.SourceID,
		auth.AccessToken,
		auth.RefreshToken,
		expiry,
		(*NullTime)(&auth.CreatedAt),
		(*NullTime)(&auth.UpdatedAt),
	)
	if err != nil {
		return FormatError(err)
	}

	// Update caller object to set ID.
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	auth.ID = int(id)

	return nil
}

// updateAuth updates tokens & expiry on exist auth object.
// Returns new state of the auth object.
func updateAuth(ctx context.Context, tx *Tx, id int, accessToken, refreshToken string, expiry *time.Time) (*wtf.Auth, error) {
	// Fetch current object state.
	auth, err := findAuthByID(ctx, tx, id)
	if err != nil {
		return auth, err
	}

	// Update fields & last updated date.
	auth.AccessToken = accessToken
	auth.RefreshToken = refreshToken
	auth.Expiry = expiry
	auth.UpdatedAt = tx.now

	// Perform basic field validation.
	if err := auth.Validate(); err != nil {
		return auth, err
	}

	// Format timestamp to RFC 3339 for SQLite.
	var expiryStr *string
	if auth.Expiry != nil {
		v := auth.Expiry.Format(time.RFC3339)
		expiryStr = &v
	}

	// Execute SQL update query.
	if _, err := tx.ExecContext(ctx, `
		UPDATE auths
		SET access_token = ?,
		    refresh_token = ?,
		    expiry = ?,
		    updated_at = ?
		WHERE id = ?
	`,
		auth.AccessToken,
		auth.RefreshToken,
		expiryStr,
		(*NullTime)(&auth.UpdatedAt),
		id,
	); err != nil {
		return auth, FormatError(err)
	}

	return auth, nil
}

// deleteAuth permanently removes an auth object by ID.
func deleteAuth(ctx context.Context, tx *Tx, id int) error {
	// Verify object exists & that the user is the owner of the auth.
	if auth, err := findAuthByID(ctx, tx, id); err != nil {
		return err
	} else if auth.UserID != wtf.UserIDFromContext(ctx) {
		return wtf.Errorf(wtf.EUNAUTHORIZED, "You are not allowed to delete this auth.")
	}

	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM auths WHERE id = ?`, id); err != nil {
		return FormatError(err)
	}
	return nil
}

// attachAuthAssociations is a helper function to fetch & attach the associated user
// to the auth object.
func attachAuthAssociations(ctx context.Context, tx *Tx, auth *wtf.Auth) (err error) {
	if auth.User, err = findUserByID(ctx, tx, auth.UserID); err != nil {
		return fmt.Errorf("attach auth user: %w", err)
	}
	return nil
}
