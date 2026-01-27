package commands_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"gtask/internal/commands"
	"gtask/internal/config"
	"gtask/internal/exitcode"
	"gtask/internal/testutil"
)

// runCommand is a helper to run a command with FakeService.
func runCommand(t *testing.T, cmd commands.Command, svc *testutil.FakeService, args []string, quiet bool) (stdout, stderr string, code int) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer

	cfg := &config.Config{
		Dir:   t.TempDir(),
		Quiet: quiet,
	}

	ctx := context.Background()
	code = cmd.Run(ctx, cfg, svc, args, &outBuf, &errBuf)
	return outBuf.String(), errBuf.String(), code
}

// Tests for version command
func TestVersionCommand(t *testing.T) {
	cmd := &commands.VersionCmd{}

	stdout, stderr, code := runCommand(t, cmd, nil, nil, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "gtask 0.1.0\n" {
		t.Errorf("expected version output, got %q", stdout)
	}
}

// Tests for help command
func TestHelpCommand(t *testing.T) {
	cmd := &commands.HelpCmd{}

	stdout, stderr, code := runCommand(t, cmd, nil, nil, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout == "" {
		t.Error("expected help output, got empty")
	}
	// Check for key elements
	if !bytes.Contains([]byte(stdout), []byte("Usage:")) {
		t.Error("help output should contain 'Usage:'")
	}
}

// Tests for lists command
func TestListsCommand(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddList("work", "Work")

	cmd := &commands.ListsCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}

	expected := "My Tasks [default]\nShopping\nWork\n"
	if stdout != expected {
		t.Errorf("expected %q, got %q", expected, stdout)
	}
}

// Tests for list command
func TestListCommand_DefaultListWithTasks(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddTask("@default", "task1", "Buy milk")
	svc.AddTask("@default", "task2", "Buy eggs")

	cmd := &commands.ListCmd{}
	cmd.SetPage(1)
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}

	expected := "   1  Buy milk\n   2  Buy eggs\n"
	if stdout != expected {
		t.Errorf("expected %q, got %q", expected, stdout)
	}
}

func TestListCommand_EmptyDefaultList(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.ListCmd{}
	cmd.SetPage(1)
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}

	expected := "no tasks found\n"
	if stdout != expected {
		t.Errorf("expected %q, got %q", expected, stdout)
	}
}

func TestListCommand_EmptyDefaultListQuiet(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.ListCmd{}
	cmd.SetPage(1)
	stdout, stderr, code := runCommand(t, cmd, svc, nil, true)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}

	// Quiet mode should suppress "no tasks found"
	if stdout != "" {
		t.Errorf("expected empty stdout in quiet mode, got %q", stdout)
	}
}

func TestListCommand_SpecificList(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Bread")
	svc.AddTask("shopping", "item2", "Butter")

	cmd := &commands.ListCmd{}
	cmd.SetPage(1)
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"Shopping"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}

	expected := "------------\nShopping\n------------\n       1  Bread\n       2  Butter\n"
	if stdout != expected {
		t.Errorf("expected %q, got %q", expected, stdout)
	}
}

func TestListCommand_ListNotFound(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.ListCmd{}
	cmd.SetPage(1)
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"NonExistent"}, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	expected := "error: list not found: NonExistent\n"
	if stderr != expected {
		t.Errorf("expected %q, got %q", expected, stderr)
	}
}

func TestListCommand_MultipleListsWithTasks(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddTask("@default", "task1", "Default task")
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Buy bread")

	cmd := &commands.ListCmd{}
	cmd.SetPage(1)
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}

	// Named lists show letter+number (e.g., "a1") in gtask (all-lists) view
	expected := "   1  Default task\n------------\nShopping\n------------\n      a1  Buy bread\n"
	if stdout != expected {
		t.Errorf("expected %q, got %q", expected, stdout)
	}
}

// Tests for add command
func TestAddCommand_Success(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.AddCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"Buy", "groceries"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}

	// Verify task was created
	tasks, _ := svc.ListOpenTasks(context.Background(), "@default", 1)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "Buy groceries" {
		t.Errorf("expected title 'Buy groceries', got %q", tasks[0].Title)
	}
}

func TestAddCommand_Quiet(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.AddCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"Buy", "milk"}, true)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "" {
		t.Errorf("expected empty stdout in quiet mode, got %q", stdout)
	}
}

func TestAddCommand_NoTitle(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.AddCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: title required\n" {
		t.Errorf("expected title required error, got %q", stderr)
	}
}

func TestAddCommand_ToSpecificList(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")

	cmd := &commands.AddCmd{}
	// Register and set the flag
	cmd.SetListName("Shopping")
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"Bread"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}

	// Verify task was created in Shopping list
	tasks, _ := svc.ListOpenTasks(context.Background(), "shopping", 1)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task in Shopping list, got %d", len(tasks))
	}
}

// Tests for done command
func TestDoneCommand_Success(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddTask("@default", "task1", "Buy milk")
	svc.AddTask("@default", "task2", "Buy eggs")

	cmd := &commands.DoneCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"1"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}

	// Verify task was completed (only 1 open task remains)
	tasks, _ := svc.ListOpenTasks(context.Background(), "@default", 1)
	if len(tasks) != 1 {
		t.Errorf("expected 1 open task remaining, got %d", len(tasks))
	}
	if tasks[0].Title != "Buy eggs" {
		t.Errorf("expected remaining task 'Buy eggs', got %q", tasks[0].Title)
	}
}

func TestDoneCommand_NoRef(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.DoneCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: task reference required\n" {
		t.Errorf("expected task reference required error, got %q", stderr)
	}
}

func TestDoneCommand_InvalidRef(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.DoneCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"abc"}, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: invalid task reference: abc\n" {
		t.Errorf("expected invalid task reference error, got %q", stderr)
	}
}

func TestDoneCommand_OutOfRange(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddTask("@default", "task1", "Only task")

	cmd := &commands.DoneCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"5"}, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: task number out of range: 5\n" {
		t.Errorf("expected out of range error, got %q", stderr)
	}
}

// Tests for rm command
func TestRmCommand_Success(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddTask("@default", "task1", "Buy milk")
	svc.AddTask("@default", "task2", "Buy eggs")

	cmd := &commands.RmCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"1"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}

	// Verify task was deleted
	tasks, _ := svc.ListOpenTasks(context.Background(), "@default", 1)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task remaining, got %d", len(tasks))
	}
}

func TestRmCommand_NoRef(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.RmCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: task reference required\n" {
		t.Errorf("expected task reference required error, got %q", stderr)
	}
}

// Tests for createlist command
func TestCreateListCommand_Success(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.CreateListCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"Shopping"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}

	// Verify list was created
	lists, _ := svc.ListLists(context.Background())
	if len(lists) != 2 { // default + new list
		t.Errorf("expected 2 lists, got %d", len(lists))
	}
}

func TestCreateListCommand_NoName(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.CreateListCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: list name required\n" {
		t.Errorf("expected list name required error, got %q", stderr)
	}
}

// Tests for rmlist command
func TestRmListCommand_EmptyListSuccess(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")

	cmd := &commands.RmListCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"Shopping"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}
}

func TestRmListCommand_NonEmptyListNoForce(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Bread")

	cmd := &commands.RmListCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"Shopping"}, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: list not empty (use --force)\n" {
		t.Errorf("expected list has tasks error, got %q", stderr)
	}
}

func TestRmListCommand_NonEmptyListWithForce(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Bread")

	cmd := &commands.RmListCmd{}
	cmd.SetForce(true)
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"Shopping"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}
}

func TestRmListCommand_ListNotFound(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.RmListCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"NonExistent"}, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	expected := "error: list not found: NonExistent\n"
	if stderr != expected {
		t.Errorf("expected %q, got %q", expected, stderr)
	}
}

func TestRmListCommand_NoName(t *testing.T) {
	svc := testutil.NewFakeService()

	cmd := &commands.RmListCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: list name required\n" {
		t.Errorf("expected list name required error, got %q", stderr)
	}
}

// Tests for list letters feature

func TestListCommand_MultipleNamedListsWithLetters(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddList("work", "Work")
	svc.AddTask("shopping", "item1", "Buy bread")
	svc.AddTask("work", "task1", "Finish report")

	cmd := &commands.ListCmd{}
	cmd.SetPage(1)
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}

	// First named list gets 'a', second gets 'b'
	expected := "------------\nShopping\n------------\n      a1  Buy bread\n------------\nWork\n------------\n      b1  Finish report\n"
	if stdout != expected {
		t.Errorf("expected %q, got %q", expected, stdout)
	}
}

func TestListCommand_DefaultListOnlyNoLetters(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddTask("@default", "task1", "Buy milk")
	svc.AddTask("@default", "task2", "Buy eggs")

	cmd := &commands.ListCmd{}
	cmd.SetPage(1)
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}

	// Default list has no letters, just numbers
	expected := "   1  Buy milk\n   2  Buy eggs\n"
	if stdout != expected {
		t.Errorf("expected %q, got %q", expected, stdout)
	}
}

func TestListCommand_SkipEmptyListsNoLetterWasted(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("empty", "Empty List")  // No tasks
	svc.AddList("shopping", "Shopping") // Has tasks
	svc.AddTask("shopping", "item1", "Buy bread")

	cmd := &commands.ListCmd{}
	cmd.SetPage(1)
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}

	// Empty list is skipped, Shopping gets 'a' (not 'b')
	expected := "------------\nShopping\n------------\n      a1  Buy bread\n"
	if stdout != expected {
		t.Errorf("expected %q, got %q", expected, stdout)
	}
}

func TestDoneCommand_CombinedRef(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Buy bread")
	svc.AddTask("shopping", "item2", "Buy milk")

	cmd := &commands.DoneCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"a1"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}

	// Verify task was completed
	tasks, _ := svc.ListOpenTasks(context.Background(), "shopping", 1)
	if len(tasks) != 1 {
		t.Errorf("expected 1 open task remaining, got %d", len(tasks))
	}
	if tasks[0].Title != "Buy milk" {
		t.Errorf("expected remaining task 'Buy milk', got %q", tasks[0].Title)
	}
}

func TestDoneCommand_SeparatedRef(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Buy bread")

	cmd := &commands.DoneCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"a", "1"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}

	// Verify task was completed
	tasks, _ := svc.ListOpenTasks(context.Background(), "shopping", 1)
	if len(tasks) != 0 {
		t.Errorf("expected 0 open tasks remaining, got %d", len(tasks))
	}
}

func TestDoneCommand_ListAndLetterConflict(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Buy bread")

	cmd := &commands.DoneCmd{}
	cmd.SetListName("Shopping")
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"a1"}, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: cannot use both --list and list letter\n" {
		t.Errorf("expected conflict error, got %q", stderr)
	}
}

func TestDoneCommand_LetterNotFound(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Buy bread")

	cmd := &commands.DoneCmd{}
	// 'z' doesn't exist, only 'a' does
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"z1"}, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: list letter not found: z\n" {
		t.Errorf("expected letter not found error, got %q", stderr)
	}
}

func TestDoneCommand_LetterOnly(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Buy bread")

	cmd := &commands.DoneCmd{}
	// Letter 'a' without number
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"a"}, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "error: task reference required\n" {
		t.Errorf("expected task reference required error, got %q", stderr)
	}
}

func TestRmCommand_CombinedRef(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Buy bread")
	svc.AddTask("shopping", "item2", "Buy milk")

	cmd := &commands.RmCmd{}
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"a2"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}

	// Verify task was deleted
	tasks, _ := svc.ListOpenTasks(context.Background(), "shopping", 1)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task remaining, got %d", len(tasks))
	}
	if tasks[0].Title != "Buy bread" {
		t.Errorf("expected remaining task 'Buy bread', got %q", tasks[0].Title)
	}
}

func TestRmCommand_SeparatedRef(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("work", "Work")
	svc.AddList("shopping", "Shopping")
	svc.AddTask("work", "task1", "Finish report")
	svc.AddTask("shopping", "item1", "Buy bread")

	cmd := &commands.RmCmd{}
	// 'b' is the second list (Shopping)
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"b", "1"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}

	// Verify task was deleted from Shopping list
	tasks, _ := svc.ListOpenTasks(context.Background(), "shopping", 1)
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks in Shopping, got %d", len(tasks))
	}
}

func TestDoneCommand_SecondList(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("work", "Work")
	svc.AddList("shopping", "Shopping")
	svc.AddTask("work", "task1", "Finish report")
	svc.AddTask("shopping", "item1", "Buy bread")

	cmd := &commands.DoneCmd{}
	// 'b' is the second list (Shopping)
	stdout, stderr, code := runCommand(t, cmd, svc, []string{"b1"}, false)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
	if stdout != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout)
	}

	// Verify task was completed in Shopping list (b), not Work (a)
	shoppingTasks, _ := svc.ListOpenTasks(context.Background(), "shopping", 1)
	if len(shoppingTasks) != 0 {
		t.Errorf("expected 0 tasks in Shopping, got %d", len(shoppingTasks))
	}
	workTasks, _ := svc.ListOpenTasks(context.Background(), "work", 1)
	if len(workTasks) != 1 {
		t.Errorf("expected 1 task in Work, got %d", len(workTasks))
	}
}

func TestListCommand_TooManyLists(t *testing.T) {
	svc := testutil.NewFakeService()

	// Create 27 named lists (more than 26), each with a task
	for i := 0; i < 27; i++ {
		listID := fmt.Sprintf("list%d", i)
		listTitle := fmt.Sprintf("List %d", i)
		svc.AddList(listID, listTitle)
		svc.AddTask(listID, fmt.Sprintf("task%d", i), fmt.Sprintf("Task %d", i))
	}

	cmd := &commands.ListCmd{}
	cmd.SetPage(1)
	stdout, stderr, code := runCommand(t, cmd, svc, nil, false)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	// Some output may have been printed before the error
	_ = stdout
	if stderr != "error: too many lists (max 26)\n" {
		t.Errorf("expected too many lists error, got %q", stderr)
	}
}
