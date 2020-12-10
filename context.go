package wtf

import "context"

// contextKey represents an internal key for adding context fields.
// This is considered best practice as it prevents other packages from
// interfering with our context keys.
type contextKey int

// List of context keys.
// These are used to store request-scoped information.
const (
	// Stores the current logged in user in the context.
	userContextKey = contextKey(iota + 1)

	// Stores the "flash" in the context. This is a term used in web development
	// for a message that is passed from one request to the next for informational
	// purposes. This could be moved into the "http" package as it is only HTTP
	// related but both the "http" and "http/html" packages use it so it is
	// easier to move it to the root.
	flashContextKey
)

// NewContextWithUser returns a new context with the given user.
func NewContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext returns the current logged in user.
func UserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userContextKey).(*User)
	return user
}

// UserIDFromContext is a helper function that returns the ID of the current
// logged in user. Returns zero if no user is logged in.
func UserIDFromContext(ctx context.Context) int {
	if user := UserFromContext(ctx); user != nil {
		return user.ID
	}
	return 0
}

// NewContextWithFlash returns a new context with the given flash value.
func NewContextWithFlash(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, flashContextKey, v)
}

// FlashFromContext returns the flash value for the current request.
func FlashFromContext(ctx context.Context) string {
	v, _ := ctx.Value(flashContextKey).(string)
	return v
}
