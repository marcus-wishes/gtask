package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"unicode"

	"gtask/internal/service"
)

// ErrTooManyLists indicates there are more than 26 named lists with open tasks
// and list letters can therefore not be assigned.
var ErrTooManyLists = errors.New("too many lists (max 26)")

// TaskRef represents a parsed task reference.
type TaskRef struct {
	Letter    rune // 0 if no letter, 'a'-'z' otherwise
	TaskNum   int  // 1-based task number
	HasLetter bool // true if a list letter was provided
}

// ErrTaskRefRequired indicates no task reference was provided.
var ErrTaskRefRequired = errors.New("task reference required")

// ParseTaskRefs parses one or more task references from args.
//
// References are parsed left-to-right, consuming either:
//   - one token for <number>
//   - one token for <letter><number>
func ParseTaskRefs(args []string) ([]TaskRef, error) {
	if len(args) == 0 {
		return nil, ErrTaskRefRequired
	}

	var refs []TaskRef
	for i := 0; i < len(args); {
		token := args[i]

		// <number>
		if isAllDigits(token) {
			num, err := strconv.Atoi(token)
			if err != nil {
				return nil, fmt.Errorf("invalid task reference: %s", token)
			}
			refs = append(refs, TaskRef{TaskNum: num, HasLetter: false})
			i++
			continue
		}

		// <letter><number>
		if len(token) > 0 && isLetter(rune(token[0])) {
			letter := rune(token[0])

			// <letter><number>
			if len(token) > 1 {
				if !isAllDigits(token[1:]) {
					return nil, fmt.Errorf("invalid task reference: %s", token)
				}
				num, err := strconv.Atoi(token[1:])
				if err != nil {
					return nil, fmt.Errorf("invalid task reference: %s", token)
				}
				refs = append(refs, TaskRef{Letter: letter, TaskNum: num, HasLetter: true})
				i++
				continue
			}
		}

		return nil, fmt.Errorf("invalid task reference: %s", token)
	}

	return refs, nil
}

// ParseTaskRef parses task reference from args.
// Returns the parsed reference and any error.
//
// Parsing rules (from spec §3.5):
// 1. If first arg is all digits → default list reference
// 2. If first arg is <letter><digits> (e.g., a1, b12) → combined reference
// 3. Otherwise → error: invalid task reference: <ref>
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
	}

	// Case 3: Invalid reference
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

// BuildListLetterMap assigns letters (a-z) to named lists with open tasks in API order.
// The default list never receives a letter.
func BuildListLetterMap(ctx context.Context, svc service.Service) (map[rune]service.TaskList, error) {
	lists, err := svc.ListLists(ctx)
	if err != nil {
		return nil, err
	}

	letter := 'a'
	byLetter := make(map[rune]service.TaskList)

	for _, list := range lists {
		if list.IsDefault {
			continue
		}

		hasOpen, err := svc.HasOpenTasks(ctx, list.ID)
		if err != nil {
			return nil, err
		}
		if !hasOpen {
			continue
		}

		if letter > 'z' {
			return nil, ErrTooManyLists
		}

		byLetter[letter] = list
		letter++
	}

	return byLetter, nil
}

// ResolveListByLetter resolves a list letter to a TaskList.
// Fetches all lists, assigns letters to named lists with open tasks, returns matching list.
// Returns error if letter is not found.
func ResolveListByLetter(ctx context.Context, svc service.Service, letter rune) (service.TaskList, error) {
	byLetter, err := BuildListLetterMap(ctx, svc)
	if err != nil {
		return service.TaskList{}, err
	}
	list, ok := byLetter[letter]
	if !ok {
		return service.TaskList{}, fmt.Errorf("list letter not found: %c", letter)
	}
	return list, nil
}
