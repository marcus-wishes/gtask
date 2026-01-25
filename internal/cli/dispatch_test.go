package cli_test

import (
	"bytes"
	"context"
	"testing"

	"gtask/internal/cli"
	"gtask/internal/commands"
	"gtask/internal/config"
	"gtask/internal/exitcode"
	"gtask/internal/service"
	"gtask/internal/testutil"
)

// testFactory creates a service factory that returns the given FakeService.
func testFactory(svc *testutil.FakeService) cli.ServiceFactory {
	return func(ctx context.Context, cfg *config.Config) (service.Service, error) {
		return svc, nil
	}
}

func TestDispatcher_UnknownCommand(t *testing.T) {
	svc := testutil.NewFakeService()
	dispatcher := cli.NewDispatcher(commands.DefaultRegistry, testFactory(svc))

	var stdout, stderr bytes.Buffer
	code := dispatcher.Run(context.Background(), []string{"unknowncmd"}, &stdout, &stderr)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	expected := "error: unknown command: unknowncmd\n"
	if stderr.String() != expected {
		t.Errorf("expected %q, got %q", expected, stderr.String())
	}
}

func TestDispatcher_FlagBeforeCommand(t *testing.T) {
	svc := testutil.NewFakeService()
	dispatcher := cli.NewDispatcher(commands.DefaultRegistry, testFactory(svc))

	var stdout, stderr bytes.Buffer
	code := dispatcher.Run(context.Background(), []string{"--quiet"}, &stdout, &stderr)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	expected := "error: unknown command: --quiet\n"
	if stderr.String() != expected {
		t.Errorf("expected %q, got %q", expected, stderr.String())
	}
}

func TestDispatcher_HelpCommand(t *testing.T) {
	svc := testutil.NewFakeService()
	dispatcher := cli.NewDispatcher(commands.DefaultRegistry, testFactory(svc))

	var stdout, stderr bytes.Buffer
	code := dispatcher.Run(context.Background(), []string{"help"}, &stdout, &stderr)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr.String() != "" {
		t.Errorf("expected no stderr, got %q", stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage:")) {
		t.Error("expected help output to contain 'Usage:'")
	}
}

func TestDispatcher_VersionCommand(t *testing.T) {
	svc := testutil.NewFakeService()
	dispatcher := cli.NewDispatcher(commands.DefaultRegistry, testFactory(svc))

	var stdout, stderr bytes.Buffer
	code := dispatcher.Run(context.Background(), []string{"version"}, &stdout, &stderr)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr.String() != "" {
		t.Errorf("expected no stderr, got %q", stderr.String())
	}
	if stdout.String() != "gtask 0.1.0\n" {
		t.Errorf("expected 'gtask 0.1.0\\n', got %q", stdout.String())
	}
}

func TestDispatcher_UnknownFlag(t *testing.T) {
	svc := testutil.NewFakeService()
	dispatcher := cli.NewDispatcher(commands.DefaultRegistry, testFactory(svc))

	var stdout, stderr bytes.Buffer
	code := dispatcher.Run(context.Background(), []string{"help", "--unknown"}, &stdout, &stderr)

	if code != exitcode.UserError {
		t.Errorf("expected exit code %d, got %d", exitcode.UserError, code)
	}
	expected := "error: unknown flag: -unknown\n"
	if stderr.String() != expected {
		t.Errorf("expected %q, got %q", expected, stderr.String())
	}
}

func TestDispatcher_AddWithShortListFlag(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	dispatcher := cli.NewDispatcher(commands.DefaultRegistry, testFactory(svc))

	var stdout, stderr bytes.Buffer
	code := dispatcher.Run(context.Background(), []string{"add", "-l", "Shopping", "Buy", "bread"}, &stdout, &stderr)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr.String() != "" {
		t.Errorf("expected no stderr, got %q", stderr.String())
	}
	if stdout.String() != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", stdout.String())
	}

	// Verify task was created in Shopping list
	tasks, _ := svc.ListOpenTasks(context.Background(), "shopping", 1)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task in Shopping list, got %d", len(tasks))
	}
	if tasks[0].Title != "Buy bread" {
		t.Errorf("expected title 'Buy bread', got %q", tasks[0].Title)
	}
}

func TestDispatcher_DoneWithShortListFlag(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Bread")

	dispatcher := cli.NewDispatcher(commands.DefaultRegistry, testFactory(svc))

	var stdout, stderr bytes.Buffer
	code := dispatcher.Run(context.Background(), []string{"done", "-l", "Shopping", "1"}, &stdout, &stderr)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr.String() != "" {
		t.Errorf("expected no stderr, got %q", stderr.String())
	}

	// Verify task was completed
	tasks, _ := svc.ListOpenTasks(context.Background(), "shopping", 1)
	if len(tasks) != 0 {
		t.Errorf("expected 0 open tasks, got %d", len(tasks))
	}
}

func TestDispatcher_RmWithShortListFlag(t *testing.T) {
	svc := testutil.NewFakeService()
	svc.AddList("shopping", "Shopping")
	svc.AddTask("shopping", "item1", "Bread")

	dispatcher := cli.NewDispatcher(commands.DefaultRegistry, testFactory(svc))

	var stdout, stderr bytes.Buffer
	code := dispatcher.Run(context.Background(), []string{"rm", "-l", "Shopping", "1"}, &stdout, &stderr)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if stderr.String() != "" {
		t.Errorf("expected no stderr, got %q", stderr.String())
	}

	// Verify task was deleted
	tasks, _ := svc.ListOpenTasks(context.Background(), "shopping", 1)
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}
