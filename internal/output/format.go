// Package output provides formatters for CLI output.
package output

import (
	"fmt"
	"io"
	"strings"

	"gtask/internal/service"
)

const (
	// ListSeparator is the separator line for list sections.
	ListSeparator = "------------"
)

// FormatTask formats a task line for the default list.
// Format: "{N:>4}  {TITLE}\n" (4-wide right-aligned number, two spaces, title)
func FormatTask(w io.Writer, num int, task service.Task) {
	title := normalizeTitle(task.Title)
	fmt.Fprintf(w, "%4d  %s\n", num, title)
}

// FormatTaskIndented formats a task line for a named list section.
// Format: "    {N:>4}  {TITLE}\n" (4 spaces indent + 4-wide number + 2 spaces + title)
func FormatTaskIndented(w io.Writer, num int, task service.Task) {
	title := normalizeTitle(task.Title)
	fmt.Fprintf(w, "    %4d  %s\n", num, title)
}

// FormatListHeader formats a list section header.
func FormatListHeader(w io.Writer, title string, isDefault bool) {
	displayTitle := normalizeListTitle(title)
	if isDefault {
		displayTitle += " [default]"
	}
	fmt.Fprintln(w, ListSeparator)
	fmt.Fprintln(w, displayTitle)
	fmt.Fprintln(w, ListSeparator)
}

// FormatListName formats a list name for the lists command.
func FormatListName(w io.Writer, list service.TaskList) {
	title := normalizeListTitle(list.Title)
	if list.IsDefault {
		title += " [default]"
	}
	fmt.Fprintln(w, title)
}

// normalizeTitle normalizes a task title for display.
// - Empty or whitespace-only titles become "(untitled)"
// - Newlines are replaced with spaces
func normalizeTitle(title string) string {
	// Replace newlines with spaces
	title = strings.ReplaceAll(title, "\r", " ")
	title = strings.ReplaceAll(title, "\n", " ")

	// Trim and check for empty
	if strings.TrimSpace(title) == "" {
		return "(untitled)"
	}
	return title
}

// normalizeListTitle normalizes a list title for display.
// Empty or whitespace-only titles become "(untitled)".
func normalizeListTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return "(untitled)"
	}
	return title
}
