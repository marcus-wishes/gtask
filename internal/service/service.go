// Package service defines the backend-agnostic interface for task operations.
package service

import "context"

// Service defines the interface for task backend operations.
// All Google Tasks API calls go through this interface.
// Commands never import Google SDK directly.
type Service interface {
	// DefaultList returns the user's default task list.
	DefaultList(ctx context.Context) (TaskList, error)

	// ListLists returns all task lists in API order.
	ListLists(ctx context.Context) ([]TaskList, error)

	// ResolveList finds a list by name (case-insensitive, trimmed).
	// Returns error if not found or ambiguous.
	ResolveList(ctx context.Context, name string) (TaskList, error)

	// CreateList creates a new task list.
	CreateList(ctx context.Context, name string) error

	// DeleteList deletes a task list by ID.
	DeleteList(ctx context.Context, listID string) error

	// ListOpenTasks returns open tasks for a list.
	// page is 1-based; page size is 100.
	// Returns empty slice if page is out of range.
	// Results are in API order (no client-side sorting).
	ListOpenTasks(ctx context.Context, listID string, page int) ([]Task, error)

	// HasOpenTasks checks if a list has any open tasks.
	HasOpenTasks(ctx context.Context, listID string) (bool, error)

	// CreateTask creates a new task in the specified list.
	CreateTask(ctx context.Context, listID, title string) error

	// CompleteTask marks a task as completed.
	CompleteTask(ctx context.Context, listID, taskID string) error

	// DeleteTask deletes a task.
	DeleteTask(ctx context.Context, listID, taskID string) error
}
