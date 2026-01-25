// Package googletasks implements the service.Service interface using Google Tasks API.
package googletasks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	tasks "google.golang.org/api/tasks/v1"

	"gtask/internal/config"
	"gtask/internal/service"
)

const (
	// DefaultListID is the special ID for the default list.
	DefaultListID = "@default"

	// PageSize is the number of tasks per page.
	PageSize = 100

	// APITimeout is the timeout for API calls.
	APITimeout = 5 * time.Second

	// OAuth scope for Google Tasks
	tasksScope = "https://www.googleapis.com/auth/tasks"
)

// Client implements service.Service using Google Tasks API.
type Client struct {
	svc       *tasks.Service
	cfg       *config.Config
	tokenPath string
}

// New creates a new Google Tasks client.
// Requires oauth_client.json and token.json to exist.
func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	// Load OAuth client config
	clientJSON, err := os.ReadFile(cfg.OAuthClientPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read oauth_client.json: %w", err)
	}

	oauthConfig, err := google.ConfigFromJSON(clientJSON, tasksScope)
	if err != nil {
		return nil, fmt.Errorf("invalid oauth_client.json: %w", err)
	}

	// Load token
	tokenData, err := os.ReadFile(cfg.TokenPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read token.json: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(tokenData, &token); err != nil {
		return nil, fmt.Errorf("invalid token.json: %w", err)
	}

	// Create token source that auto-refreshes
	tokenSource := oauthConfig.TokenSource(ctx, &token)

	// Create HTTP client with token source
	httpClient := oauth2.NewClient(ctx, tokenSource)

	// Create Tasks service
	svc, err := tasks.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create tasks service: %w", err)
	}

	return &Client{
		svc:       svc,
		cfg:       cfg,
		tokenPath: cfg.TokenPath(),
	}, nil
}

// NewWithHTTPClient creates a client with a custom HTTP client (for testing).
func NewWithHTTPClient(ctx context.Context, httpClient *http.Client) (*Client, error) {
	svc, err := tasks.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}
	return &Client{svc: svc}, nil
}

// DefaultList returns the user's default task list.
func (c *Client) DefaultList(ctx context.Context) (service.TaskList, error) {
	ctx, cancel := context.WithTimeout(ctx, APITimeout)
	defer cancel()

	list, err := c.svc.Tasklists.Get(DefaultListID).Context(ctx).Do()
	if err != nil {
		return service.TaskList{}, wrapError(err)
	}

	return service.TaskList{
		ID:        DefaultListID,
		Title:     list.Title,
		IsDefault: true,
	}, nil
}

// ListLists returns all task lists in API order.
func (c *Client) ListLists(ctx context.Context) ([]service.TaskList, error) {
	ctx, cancel := context.WithTimeout(ctx, APITimeout)
	defer cancel()

	// First, get the default list to know its real ID
	defaultList, err := c.svc.Tasklists.Get(DefaultListID).Context(ctx).Do()
	if err != nil {
		return nil, wrapError(err)
	}
	defaultRealID := defaultList.Id

	// List all task lists
	var result []service.TaskList
	err = c.svc.Tasklists.List().MaxResults(100).Pages(ctx, func(resp *tasks.TaskLists) error {
		for _, list := range resp.Items {
			isDefault := list.Id == defaultRealID
			id := list.Id
			if isDefault {
				id = DefaultListID // Normalize to @default
			}
			result = append(result, service.TaskList{
				ID:        id,
				Title:     list.Title,
				IsDefault: isDefault,
			})
		}
		return nil
	})
	if err != nil {
		return nil, wrapError(err)
	}

	return result, nil
}

// ResolveList finds a list by name (case-insensitive, trimmed).
func (c *Client) ResolveList(ctx context.Context, name string) (service.TaskList, error) {
	name = strings.TrimSpace(name)
	nameLower := strings.ToLower(name)

	lists, err := c.ListLists(ctx)
	if err != nil {
		return service.TaskList{}, err
	}

	var matches []service.TaskList
	for _, list := range lists {
		if strings.ToLower(strings.TrimSpace(list.Title)) == nameLower {
			matches = append(matches, list)
		}
	}

	switch len(matches) {
	case 0:
		return service.TaskList{}, fmt.Errorf("list not found: %s", name)
	case 1:
		return matches[0], nil
	default:
		return service.TaskList{}, fmt.Errorf("ambiguous list name: %s", name)
	}
}

// CreateList creates a new task list.
func (c *Client) CreateList(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, APITimeout)
	defer cancel()

	_, err := c.svc.Tasklists.Insert(&tasks.TaskList{Title: name}).Context(ctx).Do()
	if err != nil {
		return wrapError(err)
	}
	return nil
}

// DeleteList deletes a task list by ID.
func (c *Client) DeleteList(ctx context.Context, listID string) error {
	ctx, cancel := context.WithTimeout(ctx, APITimeout)
	defer cancel()

	err := c.svc.Tasklists.Delete(listID).Context(ctx).Do()
	if err != nil {
		return wrapError(err)
	}
	return nil
}

// ListOpenTasks returns open tasks for a list.
func (c *Client) ListOpenTasks(ctx context.Context, listID string, page int) ([]service.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, APITimeout)
	defer cancel()

	// Build request
	call := c.svc.Tasks.List(listID).
		MaxResults(PageSize).
		ShowCompleted(false).
		ShowDeleted(false).
		ShowHidden(false).
		Context(ctx)

	// Handle pagination by fetching pages until we reach the requested one
	// Google Tasks API uses page tokens, not page numbers
	currentPage := 1
	var pageToken string

	for currentPage < page {
		resp, err := call.PageToken(pageToken).Do()
		if err != nil {
			return nil, wrapError(err)
		}
		if resp.NextPageToken == "" {
			// No more pages, requested page is out of range
			return nil, nil
		}
		pageToken = resp.NextPageToken
		currentPage++
	}

	// Fetch the requested page
	resp, err := call.PageToken(pageToken).Do()
	if err != nil {
		return nil, wrapError(err)
	}

	var result []service.Task
	for _, task := range resp.Items {
		result = append(result, service.Task{
			ID:       task.Id,
			Title:    task.Title,
			Position: task.Position,
			Status:   task.Status,
		})
	}

	return result, nil
}

// HasOpenTasks checks if a list has any open tasks.
func (c *Client) HasOpenTasks(ctx context.Context, listID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, APITimeout)
	defer cancel()

	resp, err := c.svc.Tasks.List(listID).
		MaxResults(1).
		ShowCompleted(false).
		ShowDeleted(false).
		ShowHidden(false).
		Context(ctx).
		Do()
	if err != nil {
		return false, wrapError(err)
	}

	return len(resp.Items) > 0, nil
}

// CreateTask creates a new task in the specified list.
func (c *Client) CreateTask(ctx context.Context, listID, title string) error {
	ctx, cancel := context.WithTimeout(ctx, APITimeout)
	defer cancel()

	_, err := c.svc.Tasks.Insert(listID, &tasks.Task{Title: title}).Context(ctx).Do()
	if err != nil {
		return wrapError(err)
	}
	return nil
}

// CompleteTask marks a task as completed.
func (c *Client) CompleteTask(ctx context.Context, listID, taskID string) error {
	ctx, cancel := context.WithTimeout(ctx, APITimeout)
	defer cancel()

	_, err := c.svc.Tasks.Patch(listID, taskID, &tasks.Task{
		Status: "completed",
	}).Context(ctx).Do()
	if err != nil {
		return wrapError(err)
	}
	return nil
}

// DeleteTask deletes a task.
func (c *Client) DeleteTask(ctx context.Context, listID, taskID string) error {
	ctx, cancel := context.WithTimeout(ctx, APITimeout)
	defer cancel()

	err := c.svc.Tasks.Delete(listID, taskID).Context(ctx).Do()
	if err != nil {
		return wrapError(err)
	}
	return nil
}

// wrapError wraps API errors with user-friendly messages.
func wrapError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for timeout
	if strings.Contains(errStr, "context deadline exceeded") {
		return fmt.Errorf("request timed out")
	}

	// Check for auth errors
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
		return fmt.Errorf("token expired or revoked (run: gtask login)")
	}

	// Check for not found
	if strings.Contains(errStr, "404") {
		return fmt.Errorf("not found")
	}

	return err
}
