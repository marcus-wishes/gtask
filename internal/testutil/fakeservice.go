// Package testutil provides testing utilities.
package testutil

import (
	"context"
	"errors"
	"strings"
	"sync"

	"gtask/internal/service"
)

// DefaultListID is the ID used for the default list.
const DefaultListID = "@default"

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("not found")

// ErrAmbiguous is returned when multiple matches are found.
var ErrAmbiguous = errors.New("ambiguous")

// FakeService is an in-memory implementation of service.Service for testing.
type FakeService struct {
	mu    sync.RWMutex
	lists []service.TaskList
	tasks map[string][]service.Task // listID -> tasks

	// Error injection for testing
	DefaultListErr   error
	ListListsErr     error
	ResolveListErr   error
	CreateListErr    error
	DeleteListErr    error
	ListOpenTasksErr map[string]error // listID -> error
	HasOpenTasksErr  error
	CreateTaskErr    error
	CompleteTaskErr  error
	DeleteTaskErr    error
}

// NewFakeService creates a new FakeService with a default list.
func NewFakeService() *FakeService {
	fs := &FakeService{
		tasks:            make(map[string][]service.Task),
		ListOpenTasksErr: make(map[string]error),
	}
	// Add default list
	fs.lists = []service.TaskList{
		{ID: DefaultListID, Title: "My Tasks", IsDefault: true},
	}
	fs.tasks[DefaultListID] = nil
	return fs
}

// AddList adds a list to the fake service.
func (f *FakeService) AddList(id, title string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lists = append(f.lists, service.TaskList{ID: id, Title: title, IsDefault: false})
	if f.tasks[id] == nil {
		f.tasks[id] = nil
	}
}

// AddTask adds a task to a list.
func (f *FakeService) AddTask(listID, taskID, title string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tasks[listID] = append(f.tasks[listID], service.Task{
		ID:     taskID,
		Title:  title,
		Status: "needsAction",
	})
}

// DefaultList implements service.Service.
func (f *FakeService) DefaultList(ctx context.Context) (service.TaskList, error) {
	if f.DefaultListErr != nil {
		return service.TaskList{}, f.DefaultListErr
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, l := range f.lists {
		if l.IsDefault {
			return l, nil
		}
	}
	return service.TaskList{}, errors.New("no default list")
}

// ListLists implements service.Service.
func (f *FakeService) ListLists(ctx context.Context) ([]service.TaskList, error) {
	if f.ListListsErr != nil {
		return nil, f.ListListsErr
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make([]service.TaskList, len(f.lists))
	copy(result, f.lists)
	return result, nil
}

// ResolveList implements service.Service.
func (f *FakeService) ResolveList(ctx context.Context, name string) (service.TaskList, error) {
	if f.ResolveListErr != nil {
		return service.TaskList{}, f.ResolveListErr
	}
	f.mu.RLock()
	defer f.mu.RUnlock()

	name = strings.TrimSpace(name)
	nameLower := strings.ToLower(name)

	var matches []service.TaskList
	for _, l := range f.lists {
		if strings.ToLower(strings.TrimSpace(l.Title)) == nameLower {
			matches = append(matches, l)
		}
	}

	switch len(matches) {
	case 0:
		return service.TaskList{}, ErrNotFound
	case 1:
		return matches[0], nil
	default:
		return service.TaskList{}, ErrAmbiguous
	}
}

// CreateList implements service.Service.
func (f *FakeService) CreateList(ctx context.Context, name string) error {
	if f.CreateListErr != nil {
		return f.CreateListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	// Generate a simple ID
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	f.lists = append(f.lists, service.TaskList{ID: id, Title: name, IsDefault: false})
	f.tasks[id] = nil
	return nil
}

// DeleteList implements service.Service.
func (f *FakeService) DeleteList(ctx context.Context, listID string) error {
	if f.DeleteListErr != nil {
		return f.DeleteListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	for i, l := range f.lists {
		if l.ID == listID {
			f.lists = append(f.lists[:i], f.lists[i+1:]...)
			delete(f.tasks, listID)
			return nil
		}
	}
	return ErrNotFound
}

// ListOpenTasks implements service.Service.
func (f *FakeService) ListOpenTasks(ctx context.Context, listID string, page int) ([]service.Task, error) {
	if err, ok := f.ListOpenTasksErr[listID]; ok && err != nil {
		return nil, err
	}
	f.mu.RLock()
	defer f.mu.RUnlock()

	tasks, ok := f.tasks[listID]
	if !ok {
		return nil, ErrNotFound
	}

	// Filter open tasks
	var open []service.Task
	for _, t := range tasks {
		if t.Status == "needsAction" {
			open = append(open, t)
		}
	}

	// Paginate (100 per page)
	const pageSize = 100
	start := (page - 1) * pageSize
	if start >= len(open) {
		return nil, nil
	}
	end := start + pageSize
	if end > len(open) {
		end = len(open)
	}
	return open[start:end], nil
}

// HasOpenTasks implements service.Service.
func (f *FakeService) HasOpenTasks(ctx context.Context, listID string) (bool, error) {
	if f.HasOpenTasksErr != nil {
		return false, f.HasOpenTasksErr
	}
	f.mu.RLock()
	defer f.mu.RUnlock()

	tasks, ok := f.tasks[listID]
	if !ok {
		return false, ErrNotFound
	}

	for _, t := range tasks {
		if t.Status == "needsAction" {
			return true, nil
		}
	}
	return false, nil
}

// CreateTask implements service.Service.
func (f *FakeService) CreateTask(ctx context.Context, listID, title string) error {
	if f.CreateTaskErr != nil {
		return f.CreateTaskErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.tasks[listID]; !ok {
		return ErrNotFound
	}

	// Generate a simple ID
	id := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
	f.tasks[listID] = append(f.tasks[listID], service.Task{
		ID:     id,
		Title:  title,
		Status: "needsAction",
	})
	return nil
}

// CompleteTask implements service.Service.
func (f *FakeService) CompleteTask(ctx context.Context, listID, taskID string) error {
	if f.CompleteTaskErr != nil {
		return f.CompleteTaskErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	tasks, ok := f.tasks[listID]
	if !ok {
		return ErrNotFound
	}

	for i, t := range tasks {
		if t.ID == taskID {
			f.tasks[listID][i].Status = "completed"
			return nil
		}
	}
	return ErrNotFound
}

// DeleteTask implements service.Service.
func (f *FakeService) DeleteTask(ctx context.Context, listID, taskID string) error {
	if f.DeleteTaskErr != nil {
		return f.DeleteTaskErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	tasks, ok := f.tasks[listID]
	if !ok {
		return ErrNotFound
	}

	for i, t := range tasks {
		if t.ID == taskID {
			f.tasks[listID] = append(tasks[:i], tasks[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
