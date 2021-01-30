package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/benbjohnson/wtf"
)

// Ensure service implements interface.
var _ wtf.UserService = (*UserService)(nil)

// UserService represents a service for managing users.
type UserService struct {
	db *DB
}

// NewUserService returns a new instance of UserService.
func NewUserService(db *DB) *UserService {
	return &UserService{db: db}
}

// FindUserByID retrieves a user by ID along with their associated auth objects.
// Returns ENOTFOUND if user does not exist.
func (s *UserService) FindUserByID(ctx context.Context, id int) (*wtf.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Fetch user and their associated OAuth objects.
	user, err := findUserByID(ctx, tx, id)
	if err != nil {
		return nil, err
	} else if err := attachUserAuths(ctx, tx, user); err != nil {
		return user, err
	}
	return user, nil
}

// FindUsers retrieves a list of users by filter. Also returns total count of
// matching users which may differ from returned results if filter.Limit is specified.
func (s *UserService) FindUsers(ctx context.Context, filter wtf.UserFilter) ([]*wtf.User, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()
	return findUsers(ctx, tx, filter)
}

// CreateUser creates a new user. This is only used for testing since users are
// typically created during the OAuth creation process in AuthService.CreateAuth().
func (s *UserService) CreateUser(ctx context.Context, user *wtf.User) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create a new user object and attach associated OAuth objects.
	if err := createUser(ctx, tx, user); err != nil {
		return err
	} else if err := attachUserAuths(ctx, tx, user); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateUser updates a user object. Returns EUNAUTHORIZED if current user is
// not the user that is being updated. Returns ENOTFOUND if user does not exist.
func (s *UserService) UpdateUser(ctx context.Context, id int, upd wtf.UserUpdate) (*wtf.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Update user & attach associated OAuth objects.
	user, err := updateUser(ctx, tx, id, upd)
	if err != nil {
		return user, err
	} else if err := attachUserAuths(ctx, tx, user); err != nil {
		return user, err
	} else if err := tx.Commit(); err != nil {
		return user, err
	}
	return user, nil
}

// DeleteUser permanently deletes a user and all owned dials.
// Returns EUNAUTHORIZED if current user is not the user being deleted.
// Returns ENOTFOUND if user does not exist.
func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := deleteUser(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// findUserByID is a helper function to fetch a user by ID.
// Returns ENOTFOUND if user does not exist.
func findUserByID(ctx context.Context, tx *Tx, id int) (*wtf.User, error) {
	a, _, err := findUsers(ctx, tx, wtf.UserFilter{ID: &id})
	if err != nil {
		return nil, err
	} else if len(a) == 0 {
		return nil, &wtf.Error{Code: wtf.ENOTFOUND, Message: "User not found."}
	}
	return a[0], nil
}

// findUserByEmail is a helper function to fetch a user by email.
// Returns ENOTFOUND if user does not exist.
func findUserByEmail(ctx context.Context, tx *Tx, email string) (*wtf.User, error) {
	a, _, err := findUsers(ctx, tx, wtf.UserFilter{Email: &email})
	if err != nil {
		return nil, err
	} else if len(a) == 0 {
		return nil, &wtf.Error{Code: wtf.ENOTFOUND, Message: "User not found."}
	}
	return a[0], nil
}

// findUsers returns a list of users matching a filter. Also returns a count of
// total matching users which may differ if filter.Limit is set.
func findUsers(ctx context.Context, tx *Tx, filter wtf.UserFilter) (_ []*wtf.User, n int, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := filter.Email; v != nil {
		where, args = append(where, "email = ?"), append(args, *v)
	}
	if v := filter.APIKey; v != nil {
		where, args = append(where, "api_key = ?"), append(args, *v)
	}

	// Execute query to fetch user rows.
	rows, err := tx.QueryContext(ctx, `
		SELECT 
		    id,
		    name,
		    email,
		    api_key,
		    created_at,
		    updated_at,
		    COUNT(*) OVER()
		FROM users
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY id ASC
		`+FormatLimitOffset(filter.Limit, filter.Offset),
		args...,
	)
	if err != nil {
		return nil, n, err
	}
	defer rows.Close()

	// Deserialize rows into User objects.
	users := make([]*wtf.User, 0)
	for rows.Next() {
		var email sql.NullString
		var user wtf.User
		if rows.Scan(
			&user.ID,
			&user.Name,
			&email,
			&user.APIKey,
			(*NullTime)(&user.CreatedAt),
			(*NullTime)(&user.UpdatedAt),
			&n,
		); err != nil {
			return nil, 0, err
		}

		if email.Valid {
			user.Email = email.String
		}

		users = append(users, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, n, nil
}

// createUser creates a new user. Sets the new database ID to user.ID and sets
// the timestamps to the current time.
func createUser(ctx context.Context, tx *Tx, user *wtf.User) error {
	// Set timestamps to the current time.
	user.CreatedAt = tx.now
	user.UpdatedAt = user.CreatedAt

	// Perform basic field validation.
	if err := user.Validate(); err != nil {
		return err
	}

	// Email is nullable and has a UNIQUE constraint so ensure we store blank
	// fields as NULLs.
	var email *string
	if user.Email != "" {
		email = &user.Email
	}

	// Generate random API key.
	apiKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, apiKey); err != nil {
		return err
	}
	user.APIKey = hex.EncodeToString(apiKey)

	// Execute insertion query.
	result, err := tx.ExecContext(ctx, `
		INSERT INTO users (
			name,
			email,
			api_key,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		user.Name,
		email,
		user.APIKey,
		(*NullTime)(&user.CreatedAt),
		(*NullTime)(&user.UpdatedAt),
	)
	if err != nil {
		return FormatError(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = int(id)

	return nil
}

// updateUser updates fields on a user object. Returns EUNAUTHORIZED if current
// user is not the user being updated.
func updateUser(ctx context.Context, tx *Tx, id int, upd wtf.UserUpdate) (*wtf.User, error) {
	// Fetch current object state.
	user, err := findUserByID(ctx, tx, id)
	if err != nil {
		return user, err
	} else if user.ID != wtf.UserIDFromContext(ctx) {
		return nil, wtf.Errorf(wtf.EUNAUTHORIZED, "You are not allowed to update this user.")
	}

	// Update fields.
	if v := upd.Name; v != nil {
		user.Name = *v
	}
	if v := upd.Email; v != nil {
		user.Email = *v
	}

	// Set last updated date to current time.
	user.UpdatedAt = tx.now

	// Perform basic field validation.
	if err := user.Validate(); err != nil {
		return user, err
	}

	// Email is nullable and has a UNIQUE constraint so ensure we store blank
	// fields as NULLs.
	var email *string
	if user.Email != "" {
		email = &user.Email
	}

	// Execute update query.
	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET name = ?,
		    email = ?,
		    updated_at = ?
		WHERE id = ?
	`,
		user.Name,
		email,
		(*NullTime)(&user.UpdatedAt),
		id,
	); err != nil {
		return user, FormatError(err)
	}

	return user, nil
}

// deleteUser permanently removes a user by ID. Returns EUNAUTHORIZED if current
// user is not the one being deleted.
func deleteUser(ctx context.Context, tx *Tx, id int) error {
	// Verify object exists.
	if user, err := findUserByID(ctx, tx, id); err != nil {
		return err
	} else if user.ID != wtf.UserIDFromContext(ctx) {
		return wtf.Errorf(wtf.EUNAUTHORIZED, "You are not allowed to delete this user.")
	}

	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id); err != nil {
		return FormatError(err)
	}
	return nil
}

// attachUserAuths attaches OAuth objects associated with the user.
func attachUserAuths(ctx context.Context, tx *Tx, user *wtf.User) (err error) {
	if user.Auths, _, err = findAuths(ctx, tx, wtf.AuthFilter{UserID: &user.ID}); err != nil {
		return fmt.Errorf("attach user auths: %w", err)
	}
	return nil
}
