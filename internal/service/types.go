// Package service defines the backend-agnostic interface for task operations.
package service

// Task represents a single task item.
type Task struct {
	ID       string
	Title    string
	Position string
	Status   string // "needsAction" or "completed"
}

// TaskList represents a task list.
type TaskList struct {
	ID        string
	Title     string
	IsDefault bool
}
