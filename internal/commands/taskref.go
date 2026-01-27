package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"unicode"

	"gtask/internal/service"
)

// TaskRef represents a parsed task reference.
type TaskRef struct {
	Letter    rune // 0 if no letter, 'a'-'z' otherwise
	TaskNum   int  // 1-based task number
	HasLetter bool // true if a list letter was provided
}

// ErrTaskRefRequired indicates no task reference was provided.
var ErrTaskRefRequired = errors.New("task reference required")

// ParseTaskRef parses task reference from args.
// Returns the parsed reference and any error.
//
// Parsing rules (from spec §3.5):
// 1. If first arg is all digits → default list reference
// 2. If first arg is <letter><digits> (e.g., a1, b12) → combined reference
// 3. If first arg is single letter and second arg is all digits → separated reference (a 1)
// 4. If first arg is single letter with no second arg → error: task reference required
// 5. Otherwise → error: invalid task reference: <ref>
func ParseTaskRef(args []string) (TaskRef, error) {
	if len(args) == 0 {
		return TaskRef{}, ErrTaskRefRequired
	}

	firstArg := args[0]

	// Case 1: All digits → default list, numeric reference
	if isAllDigits(firstArg) {
		num, err := strconv.Atoi(firstArg)
		if err != nil {
			return TaskRef{}, fmt.Errorf("invalid task reference: %s", firstArg)
		}
		return TaskRef{TaskNum: num, HasLetter: false}, nil
	}

	// Check if first character is a lowercase letter
	if len(firstArg) > 0 && isLetter(rune(firstArg[0])) {
		letter := rune(firstArg[0])

		// Case 2: <letter><digits> (e.g., a1, b12)
		if len(firstArg) > 1 && isAllDigits(firstArg[1:]) {
			num, err := strconv.Atoi(firstArg[1:])
			if err != nil {
				return TaskRef{}, fmt.Errorf("invalid task reference: %s", firstArg)
			}
			return TaskRef{Letter: letter, TaskNum: num, HasLetter: true}, nil
		}

		// Case 3: Single letter, check for second arg with digits
		if len(firstArg) == 1 {
			if len(args) < 2 {
				// Case 4: Single letter with no second arg
				return TaskRef{}, ErrTaskRefRequired
			}
			secondArg := args[1]
			if isAllDigits(secondArg) {
				num, err := strconv.Atoi(secondArg)
				if err != nil {
					return TaskRef{}, fmt.Errorf("invalid task reference: %s", secondArg)
				}
				return TaskRef{Letter: letter, TaskNum: num, HasLetter: true}, nil
			}
			// Second arg is not all digits
			return TaskRef{}, fmt.Errorf("invalid task reference: %s", firstArg)
		}
	}

	// Case 5: Invalid reference
	return TaskRef{}, fmt.Errorf("invalid task reference: %s", firstArg)
}

// isAllDigits returns true if s consists only of ASCII digits and is non-empty.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// isLetter returns true if r is a lowercase letter a-z.
func isLetter(r rune) bool {
	return r >= 'a' && r <= 'z'
}

// ResolveListByLetter resolves a list letter to a TaskList.
// Fetches all lists, assigns letters to named lists with open tasks, returns matching list.
// Returns error if letter is not found.
func ResolveListByLetter(ctx context.Context, svc service.Service, letter rune) (service.TaskList, error) {
	lists, err := svc.ListLists(ctx)
	if err != nil {
		return service.TaskList{}, err
	}

	currentLetter := 'a'
	for _, list := range lists {
		if list.IsDefault {
			continue
		}

		// Check if list has open tasks
		hasOpen, err := svc.HasOpenTasks(ctx, list.ID)
		if err != nil {
			return service.TaskList{}, err
		}

		if !hasOpen {
			continue // Skip empty lists
		}

		if currentLetter == letter {
			return list, nil
		}

		currentLetter++
		if currentLetter > 'z' {
			break // Exceeded max letters
		}
	}

	return service.TaskList{}, fmt.Errorf("list letter not found: %c", letter)
}
