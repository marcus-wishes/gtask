// Package exitcode defines exit codes for the CLI.
package exitcode

// Exit codes as defined in the spec.
const (
	// Success indicates successful completion.
	Success = 0

	// UserError indicates a user error (bad args, not found, ambiguous).
	UserError = 1

	// AuthError indicates an auth/config error.
	AuthError = 2

	// BackendError indicates a backend/API/network error.
	BackendError = 3
)
