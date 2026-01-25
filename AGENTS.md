# AGENTS.md — gtask

> Minimal CLI for Google Tasks in Go. Read `Spec.md` for full details.

## Quick Context

**gtask** is a tiny, fast CLI that wraps Google Tasks API with these operations:
- List open tasks (completed tasks are never shown)
- Create/complete/delete tasks
- Create/delete task lists
- OAuth login/logout

## Architecture (Non-Negotiable)

### 1. Backend Abstraction
All Google Tasks API calls go through `service.Service` interface. Commands never import Google SDK directly.

```go
type Service interface {
    DefaultList(ctx context.Context) (TaskList, error)
    ListLists(ctx context.Context) ([]TaskList, error)
    ResolveList(ctx context.Context, name string) (TaskList, error)
    CreateList(ctx context.Context, name string) error
    DeleteList(ctx context.Context, listID string) error
    ListOpenTasks(ctx context.Context, listID string, page int) ([]Task, error)
    HasOpenTasks(ctx context.Context, listID string) (bool, error)
    CreateTask(ctx context.Context, listID, title string) error
    CompleteTask(ctx context.Context, listID, taskID string) error
    DeleteTask(ctx context.Context, listID, taskID string) error
}
```

### 2. Command Registry
Commands implement `Command` interface and register themselves. No monolithic switch statements.

```go
type Command interface {
    Name() string
    Aliases() []string
    NeedsAuth() bool
    RegisterFlags(fs *flag.FlagSet)
    Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int
}
```

## Project Structure

```
gtask/
  cmd/gtask/main.go
  internal/
    cli/           # dispatch, flag parsing
    commands/      # command implementations + registry
    output/        # formatters (golden output)
    service/       # Service interface + types
    backend/
      googletasks/ # Google API implementation
    config/        # XDG paths, token persistence
    testutil/      # FakeService, golden helpers
```

## Key Conventions

### Output Format
- Success output → stdout, errors → stderr
- Exit codes: 0=success, 1=user error, 2=auth error, 3=backend error
- Error format: `error: <message>\n`
- Mutating commands print `ok\n` on success (unless `--quiet`)
- Task line: `"{N:>4}  {TITLE}\n"` (4-wide right-aligned number, 2 spaces, title)

### CLI Parsing
- Flags come AFTER command name: `gtask add --list Shopping eggs` ✓
- `gtask --quiet` is invalid (flags require a command)
- Titles/list names cannot start with `-` (parsed as unknown flag)
- Long flags only (`--flag`, no `-f`)

### Testing
- **All tests run without real Google credentials or network**
- Use `FakeService` (in-memory) for command tests
- Use `httptest.Server` for backend tests
- Golden tests compare exact byte output

## Common Flags

```
--config <dir>   Override config directory (useful for tests)
--quiet          Suppress informational output (ok, already logged in, etc.)
--debug          Print debug logs to stderr (never prints tokens)
```

## Files

| File | Purpose |
|------|---------|
| `$XDG_CONFIG_HOME/gtask/oauth_client.json` | OAuth client credentials (user provides) |
| `$XDG_CONFIG_HOME/gtask/token.json` | Stored OAuth token (mode 0600) |

## Do's and Don'ts

✅ **Do:**
- Keep `Service` interface minimal and backend-agnostic
- Write golden tests for any output changes
- Use context with 5s timeout for API calls
- Preserve API ordering (no client-side sorting)

❌ **Don't:**
- Add Google SDK imports outside `backend/googletasks/`
- Show completed tasks
- Auto-open browser during OAuth (print URL only)
- Print tokens in debug output
- Add interactive prompts (except OAuth flow)

## Task References

Tasks are referenced by number (1-based, current listing order). Numbers can change between runs — this is accepted risk for v1.

## Default List

Google's default list uses ID `@default`. When displaying, fetch actual title. `ResolveList("My Tasks")` should match and return `@default` as ID.
