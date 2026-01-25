# gtask — Architectural & Implementation Specification (Implementation-Ready)

> **Goal:** A tiny, fast CLI named `gtask` that is a minimal interface to Google Tasks written in Go:
> - list **open** tasks
> - create tasks
> - mark tasks done
> - delete tasks
> - create/delete lists
>
> **Key architecture requirements:**
> 1) The Google Tasks API usage must be encapsulated behind a replaceable backend interface.  
> 2) Input commands must be implemented via a “plugin-like” command API (compile-time registry) to make extensions easy.

---

## 0. Status of this document

This is the **current** consolidated spec based on the decisions established so far.

### Confirmed decisions included here
- Config dir uses `XDG_CONFIG_HOME` with fallback to `$HOME/.config`
- Default Google list (e.g. “My Tasks”, whatever Google calls it) is the **implicit** list for `gtask`:
  - tasks created without specifying a list go into this default list
  - when listing, the default list prints **without a list header**
- Completed tasks are **not shown**
- No due dates (yet)
- Titles whose **first token** starts with `-` are not supported (treated as flags → unknown flag error)
- List names whose **first token** starts with `-` are not supported (treated as flags → unknown flag error)
- Risk accepted: representational numbering can change between runs; acceptable for now
- Everything must be testable with Google API mocked (no real network access in tests)
- Command extensibility is via interfaces + registry (Go runtime plugins are not required)

### Note about command grammar
You previously stated: “The golden input and command grammar should come from the *gtask grammar ambiguity* chat (not from this one).”

This document therefore:
- **does not** invent complicated separators like `--` as an argument terminator
- uses only GNU-style long options like `--flag` / `--list`
- includes a **simple implementable grammar** that is consistent with all constraints we established here
  - flags always come **after** the command name

If you already have a finalized grammar from the other chat, you can replace **Section 3** while keeping the rest of this spec unchanged (output, architecture, tests).

---

## 1. Scope, Non-Goals, and Terminology

### 1.1 Scope (v1)
`gtask` must support:
- Login / logout (OAuth authentication)
- List open tasks (default list + non-empty named lists)
- Create tasks (default list; optionally in a named list)
- Mark tasks done (default list; optionally in a named list)
- Delete tasks (default list; optionally in a named list)
- Create lists
- Delete lists

### 1.2 Non-goals (v1)
- No viewing completed tasks
- No due dates
- No notes
- No interactive UI / TUI
- No background sync, no daemon mode
- No runtime plugin loading (`plugin` package). Commands are “plugin-like” via registry.

### 1.3 Terminology
- **Default list:** Google’s default task list (e.g. “My Tasks”), represented by tasklist id `@default` in Google APIs.  
  - Acts as the implicit list.
- **Named list:** Any other task list besides default.
- **Open task:** A task whose status is `"needsAction"` (and not deleted/done/hidden).
- **Task reference:** A user-provided selector for a task, typically a **number** (`1`, `2`, `3`…) in the current listing order.  
  - Optionally, future support: an ID prefix (see §3.6).

---

## 2. File System Layout and Configuration

### 2.1 Config directory
Use:
- `CONFIG_DIR = $XDG_CONFIG_HOME/gtask`  
- If `XDG_CONFIG_HOME` is empty/unset:
  - `CONFIG_DIR = $HOME/.config/gtask`

### 2.2 Files in config dir
Required:
- `oauth_client.json`  
  - OAuth client credentials for a “Desktop” (installed) application.
- `token.json`  
  - Stored OAuth token (refresh token included).

Optional (future):
- `config.json` (defaults, formatting options, etc.)

### 2.3 Permissions
- `token.json` must be created with mode `0600`
- Directory should be created with mode `0700` if it does not exist.

---

## 3. CLI and Parsing Specification

### 3.1 Global principles
- `gtask` runs one operation and exits.
- No prompts except when doing OAuth authentication.
- Output is stable and script-friendly.
- Unknown flags are errors.
- Flags must appear **after the command name** and before positional args.
- When no command is provided (`gtask`), no flags are accepted.
- If the first token after `gtask` starts with `-`, treat it as an **unknown command** (flags require a command).
- Common flags and command-specific flags may appear in any order before positional args.
- Within a command, flag parsing stops at the first non-flag token (Go stdlib `flag` parsing).
- If the **first positional token** starts with `-`, it is an error (unknown flag).
- After a valid first positional token, remaining tokens are treated as positional arguments even if they start with `-`.
- Therefore:
  - **task titles cannot start with `-`** (first token after flags)
  - **list names cannot start with `-`** (first token after flags)

### 3.2 Commands overview
Supported commands:

- `gtask`  
  List open tasks from default list and all lists (max 100 per list).

- `gtask list [--page <n>] <list-name>`  
  List open tasks for one named list (or the default list by name).

- `gtask add [--list <list-name>] <title...>`  
  Create a task.

- `gtask create [--list <list-name>] <title...>`  
  Alias for `gtask add`.

- `gtask done [--list <list-name>] <ref>`  
  Mark a task completed.

- `gtask rm [--list <list-name>] <ref>`  
  Delete a task.

- `gtask lists`  
  Print all lists (including default).

- `gtask createlist <list-name>`  
  Create a new list.

- `gtask addlist <list-name>`  
  Alias for `gtask createlist`.

- `gtask rmlist [--force] <list-name>`  
  Delete list. Without `--force`, error if list is not empty.

- `gtask login`  
  Authenticate with Google (prints OAuth URL to open in a browser). Creates `token.json`.

- `gtask logout`  
  Remove stored credentials (deletes `token.json`).

- `gtask help`  
  Print usage.

- `gtask version`  
  Print version.


### 3.3 Flags (v1)
Long options only (`--flag`).

**Common flags (available on all commands, must appear after the command name):**
- `--quiet` (suppress non-error informational output: `ok`, `already logged in`, `not logged in`, `no tasks found`; does not suppress errors)
- `--debug` (prints debug logs to stderr; must never print tokens)
- `--config <dir>` (override config dir; useful for tests)

**Important:** Common flags are only valid when a command is provided. `gtask --quiet` and `gtask --config ...` are invalid.
Common flags may appear before or after command-specific flags, as long as they are in the flag prefix (before positional args).

**Command flags:**
- `add`:
  - `--list <list-name>`
- `done`, `rm`:
  - `--list <list-name>`
- `rmlist`:
  - `--force`
- `list`:
  - `--page <n>` (1-based page number; default: 1; page size: 100 tasks per list)

### 3.4 Title and list-name token constraints
#### Titles
- Title is formed by joining remaining args with single spaces **between args**.
  - Internal spaces inside a single arg are preserved (use quotes to include multiple spaces).
- Title must not be empty after trimming.
- Title’s **first token** must not start with `-` (because it would be parsed as a flag).
- If a title contains newline characters, replace `\r` and `\n` with spaces for output consistency.

#### List names
- List name is formed by joining remaining positional arguments with single spaces **between args** (same rule as titles).
  - Internal spaces inside a single arg are preserved (use quotes to include multiple spaces).
- List names provided via `--list` flag should be quoted if they contain spaces: `--list "My List"`.
- List names must not start with `-`.
- This rule applies to `--list` values too; if the value starts with `-`, treat it as `error: unknown flag: <token>`.
- Matching is case-insensitive, trim surrounding whitespace.
- After trimming, list name must not be empty.
- If multiple lists match (possible if lists differ only by case/spacing), error:
  - `error: ambiguous list name: <name>`

### 3.5 Task references (v1 + optional extension)
**v1 MUST support:**
- Numeric references: `1`, `2`, `3`, … within the relevant list’s current ordering (API order).

**Optional (recommended for later):**
- ID prefix references (non-numeric token)
- Resolution rules:
  - if token is all digits → treat as index
  - else → treat as ID prefix:
    - list tasks for scope
    - match where `task.ID` starts with token
    - if exactly 1 match → use it
    - if 0 matches → `error: task not found: <token>`
    - if >1 matches → `error: ambiguous task id prefix: <token>`

This allows scripting via JSON output later without forcing users to type full IDs.

### 3.6 Accepted risk about ordering
The representational number `N` refers to the current ordering at execution time.
- It may change between runs if tasks are reordered externally.
- This risk is accepted for v1.

---

## 4. Golden Output Specification (MUST)

### 4.1 General rules
- Output is UTF-8 text.
- ANSI colors are optional
- Use spaces only (no tabs).
- End output with newline.
- Success output → stdout
- Error output → stderr
- Exit codes:
  - `0` success
  - `1` user error (bad args, not found, ambiguous)
  - `2` auth/config error
  - `3` backend/API/network error

### 4.2 Task line format
Default list tasks:

- Format:
  - `"{N:>4}  {TITLE}\n"`
  - 4-wide right-aligned number, two spaces, title.
- If title is empty or whitespace-only after trimming, display as `(untitled)`

Example:
```
    1  buy eggs
    2  buy jam
   12  call mom
```


### 4.3 Named list section format
Exactly:

- Separator line:
  - `"------------\n"` (12 hyphens)
- List name:
  - `"{LIST_NAME}\n"` where empty/whitespace-only names are displayed as `(untitled)`
- Separator line again:
  - `"------------\n"`
- Tasks within list section:
  - `"    {N:>4}  {TITLE}\n"`
  - 4 spaces indent + 4-wide number + 2 spaces + title

Example section:

```
------------
Shopping List
------------
    1  buy eggs
```


(That line is: 4 spaces + “   1” + two spaces + title.)

### 4.4 Output for `gtask` (no args)
Print:
1) Open tasks in default list (no header)
2) Then for each named list that has ≥1 open task (API order), print a list section

No blank lines anywhere.

Example:
```
   1  buy eggs
   2  buy butter
------------
Shopping List
------------
    1  buy eggs
```


### 4.5 Output for `gtask list <list-name>`
Always prints exactly a list section (even if empty tasks).

If the resolved list is the default list, append ` [default]` to the header (same as in `gtask lists`).

Examples:

Named list (empty):
```
------------
Shopping List
------------
```

Default list explicitly requested (e.g., `gtask list "My Tasks"`):
```
------------
My Tasks [default]
------------
    1  buy eggs
```


### 4.6 Output for `gtask lists`
Print each list title on its own line (including default list title from Google), in the order returned by the API.
If a list title is empty or whitespace-only, display as `(untitled)`.
If `IsDefault` is true, append ` [default]` to the output name; otherwise do not append.

```
My Tasks [default]
Shopping List
Work
```


### 4.7 Output for successful mutating commands
On success (unless `--quiet`):

ok

### 4.8 Output for empty `gtask` listing
If there are no open tasks across all lists when running `gtask`, print:

no tasks found

### 4.9 Output for `gtask version`
Print to stdout:

gtask <version>

### 4.10 Output for `gtask help`
Print to stdout:

Usage:
  gtask                                              List all open tasks
  gtask list [common flags] [--page <n>] <list-name> List tasks in a specific list
  gtask add [common flags] [--list <list-name>] <title...>
  gtask create [common flags] [--list <list-name>] <title...>
  gtask done [common flags] [--list <list-name>] <ref>
  gtask rm [common flags] [--list <list-name>] <ref>
  gtask lists [common flags]
  gtask createlist [common flags] <list-name>
  gtask addlist [common flags] <list-name>
  gtask rmlist [common flags] [--force] <list-name>
  gtask login [common flags]
  gtask logout [common flags]
  gtask help
  gtask version

Common flags:
  --config <dir>   Override config directory
  --quiet          Suppress informational output
  --debug          Print debug logs to stderr


### 4.11 Error output format
Single line:

error: <message>


Examples:
- `error: unknown flag: -foo`
- `error: list not found: Shopping List`
- `error: task number out of range: 7`
- `error: ambiguous list name: Shopping`

---

## 5. Ordering and Numbering Rules

### 5.1 List ordering
When printing multiple list sections (in `gtask`):
- Preserve the order returned by the API.
- Do not print named lists that have zero open tasks.

### 5.2 Task ordering within a list
Preserve the order returned by the API.
Do not perform any client-side sorting or reordering.

### 5.3 Numbering
- Default list numbering starts at 1 and increments in printed order.
- Each named list section restarts numbering at 1.
- For `gtask` (no args), each list shows up to 100 tasks (page 1 only), numbered 1-100.
- When `gtask list --page <n>` is used, numbering is absolute within the list:
  - page 1 starts at 1, page 2 starts at 101, page 3 starts at 201, etc.

---

## 6. Backend Encapsulation (Replaceable Service)

### 6.1 Architectural rule
All Google Tasks API interaction must be encapsulated behind a narrow interface so it can be replaced later with a different task service (local file, CalDAV, etc.) without changing command logic.

### 6.2 Service interface (MUST)
```go
// package service
type Service interface {
    // Lists
    DefaultList(ctx context.Context) (TaskList, error)
    ListLists(ctx context.Context) ([]TaskList, error) // API order (no sorting)
    ResolveList(ctx context.Context, name string) (TaskList, error)
    CreateList(ctx context.Context, name string) error
    DeleteList(ctx context.Context, listID string) error

    // Tasks
    // page is 1-based; pageSize is 100. Return empty slice if page is out of range.
    // Results are in API order (no client-side sorting).
    ListOpenTasks(ctx context.Context, listID string, page int) ([]Task, error)
    HasOpenTasks(ctx context.Context, listID string) (bool, error) // for rmlist emptiness check
    CreateTask(ctx context.Context, listID, title string) error
    CompleteTask(ctx context.Context, listID, taskID string) error
    DeleteTask(ctx context.Context, listID, taskID string) error
}

type TaskList struct {
    ID    string
    Title string
    IsDefault bool
}

type Task struct {
    ID       string
    Title    string
    Position string
    Status   string // "needsAction" / "completed" (backend-specific but normalized)
}
```

### 6.3 List resolution semantics

ResolveList(name) must:
- trim whitespace
- match case-insensitively against list titles
- error if not found: list not found
- error if multiple matches: ambiguous list name

Default list:
- DefaultList() returns the backend's default list object and uses list ID `@default`.
- ResolveList() can also match the default list by its title (e.g., "My Tasks").
  - If ResolveList matches the default list, it must return `ID = @default`.
  - Therefore `gtask add eggs` and `gtask add --list "My Tasks" eggs` are equivalent.


## 7. Google Tasks Backend (Implementation details)

### 7.1 API usage (high-level)

Use Google Tasks API v1 via official Go client library.

Required operations:
- List tasklists
- Insert tasklist
- Delete tasklist
- List tasks within list
- Insert task
- Patch task status to completed
- Delete task


### 7.2 Filtering open tasks

When listing open tasks:
- exclude completed tasks
- exclude deleted tasks
- exclude hidden tasks

Use API query params equivalent to:
- `showCompleted=false`
- `showDeleted=false`
- `showHidden=false`


### 7.3 Listing all tasks (for rmlist without --force)

To determine whether a list is empty:
- Query tasks with `showCompleted=false`, `showHidden=false`, `showDeleted=false`
- If any open tasks returned → list is not empty

**Definition of "empty":** A list is considered empty if it has no **open** tasks. A list with only completed tasks is considered empty for `rmlist` purposes, since completed tasks are not shown to the user anyway.

If any open tasks exist:
`error: list not empty (use --force)`


### 7.4 Completing tasks

Marking a task completed should be done with patch semantics:
- set status to `completed`
- set completed timestamp if required by API model (backend may set automatically)


### 7.5 Default list

Google supports addressing the default list with id `@default`.
Backend must:
- use `@default` as list ID for API calls
- fetch the default list's metadata (title) for display in `gtask lists`

**Populating `IsDefault` field:**
Google's API does not return an `isDefault` field on tasklists. To populate `TaskList.IsDefault`:
1. Call `tasklists.get(@default)` to retrieve the default list's real ID
2. When returning lists from `ListLists()`, mark the list whose ID matches as `IsDefault = true`
3. Normalize that list's ID to `@default` in service responses

This means `ListLists()` internally requires two API calls: one for `@default` metadata, one for the full list.

**Locale handling:**
The `@default` API alias works regardless of user locale. The default list's title (e.g., "My Tasks", "Meine Aufgaben", etc.) is returned by the API in the user's locale. `ResolveList()` matches against actual API-returned titles — no hardcoded translations are needed or performed.


### 7.6 OAuth scope

Use full tasks scope:

    https://www.googleapis.com/auth/tasks

### 7.7 Token persistence
- Store tokens in `token.json` in config dir.
- Ensure file mode 0600.
- **After each authenticated command**, write back the token (access tokens expire ~1 hour; 
  the OAuth library refreshes automatically, but we must persist the updated token).

### 7.8 Token refresh errors
If the OAuth library fails to refresh the token (e.g., token revoked, refresh token expired):
- Return `error: auth error: token expired or revoked (run: gtask login)` (exit 2)
- Do not delete `token.json` automatically; let the user run `gtask login` to re-authenticate.

### 7.9 API call timeouts
All backend API calls must use a context with a 5-second timeout by default. On timeout:
- Return `error: backend error: request timed out` (exit 3)

### 7.10 Pagination
Google Tasks API returns max 100 items per page.
- Page size is fixed at 100 tasks per list
- The `--page <n>` flag (1-based) selects which page to display for `gtask list`
- Backend must support fetching a specific page of results
- If `--page` exceeds available pages:
  - `gtask list`: print empty section as usual (see §4.5)
- Task numbering is absolute within the list: page 2 starts at task 101


## 8. Command "Plugin" Architecture (Extensible Commands)

## 8.1 Goal

Adding a new command should be possible by:
- creating a new command implementation
- registering it without modifying a monolithic switch

## 8.2 Command interface

```go
// package commands
type Command interface {
    Name() string
    Aliases() []string
    Synopsis() string
    Usage() string
    NeedsAuth() bool // false for login, logout, help, version
    RegisterFlags(fs *flag.FlagSet)
    Run(
        ctx context.Context,
        cfg *config.Config,  // always provided (config dir, paths)
        svc service.Service, // nil if NeedsAuth() returns false
        args []string,
        out io.Writer,
        errOut io.Writer,
    ) int
}
```

### Dispatch and auth check

1. All commands receive `cfg` (config paths, etc.)
2. If `command.NeedsAuth()` returns `true`:
   - Dispatcher checks for valid `token.json`
   - If missing/invalid → `error: not logged in (run: gtask login)` (exit 2)
   - Otherwise, creates Service and passes it
3. If `NeedsAuth()` returns `false`:
   - `svc` is `nil`
   - Command handles its own logic (e.g., `login` does OAuth, `logout` deletes token)

### Args contract

The `args []string` parameter passed to `Run()`:
- Contains arguments **after** the command name has been stripped
- Does **not** contain any flags (common or command-specific); all flags are parsed by the dispatcher
- Contains only positional arguments in their original order

Command-specific flag values are stored on the command instance via `RegisterFlags`, and `Run()` should read them from there.

Example: `gtask add --list Shopping buy milk`
- Dispatcher sees: `["add", "--list", "Shopping", "buy", "milk"]`
- `add` command receives: `["buy", "milk"]`

### 8.3 Registry

```go
type Registry struct {
    cmds map[string]Command // name and aliases map to command
}

func (r *Registry) Register(c Command) error
func (r *Registry) Find(name string) (Command, bool)
func (r *Registry) All() []Command // sorted for help
```

## 8.4 Dispatch rules
1. If no args → dispatch to the `list` command (§9.1) with empty args. No flags are accepted in this mode.
2. Otherwise, read first token:
   - if token matches a registered command name/alias → select that command
   - if token begins with `-` → `error: unknown command: <token>` (flags require a command)
   - else → `error: unknown command: <token>`
3. Parse flags from the remaining args using a single `flag.FlagSet`:
   - Register common flags (`--quiet`, `--debug`, `--config`).
   - Call `command.RegisterFlags(fs)` to register command-specific flags.
   - Parse with Go stdlib `flag` semantics (flags may be interleaved; parsing stops at first non-flag token).
   - The remaining args after parsing are positional and passed to `Run()`.
4. If the command `NeedsAuth()`:
   - ensure `oauth_client.json` exists; if missing → `error: oauth_client.json not found in <config_dir>` (exit 2)
   - ensure `token.json` is valid; if missing/invalid → `error: not logged in (run: gtask login)` (exit 2)
   - create Service and dispatch
5. If the command does not require auth, dispatch with `svc = nil`.

**Note:** The `list` command serves dual purpose: `gtask` (no args) and `gtask list <name>`. When invoked without args, it lists tasks across all lists. When invoked with a name, it lists that specific list.



## 9. Per-Command Semantics (Implementation Details)

## 9.1 `gtask` (list command, no args)
1. Fetch default list open tasks (page 1, max 100 tasks, API order)
2. Print them (no header)
3. Fetch all lists (API order)
4. For each named list:
   1. fetch open tasks (page 1, max 100 tasks, API order)
   2. if non-empty, print section
5. If there are no open tasks across all lists, print `no tasks found`

**Partial failure behavior:**
If fetching tasks for a specific list fails (network error, etc.):
- Print output for all lists that succeeded (up to and including the point of failure)
- Then print error to stderr: `error: failed to fetch list: <list-name>: <reason>`
- Exit with code 3

This allows users to see partial results while knowing which list failed.


## 9.2 `gtask add [--list <name>] <title...>`

1. Resolve list:
2. if --list present → ResolveList, else → DefaultList
3. Validate title: 
   1. not empty, else → `error: title required`, exit
   2. first token must not start with `-` (parsed as flag → `error: unknown flag: <token>`)
4. CreateTask
5. Print `ok`

## 9.2a `gtask create [--list <name>] <title...>`

Alias for `gtask add` with identical behavior.


## 9.3 `gtask done [--list <name>] <ref>`
1. Resolve list (same as add)
2. Validate `<ref>` provided, else → `error: task reference required`
3. Resolve `<ref>` using absolute numbering across the entire list (not per-page):
   1. if numeric → index in the full list in API order (1-based)
   2. if out of range → `error: task number out of range: <ref>`
   3. (v1) if non-numeric → `error: invalid task reference: <ref>`
   4. (future) non-numeric → id prefix match
4. Fetch open tasks page-by-page (page size 100), in API order,
   and advance a running index until the referenced task is found.
   - If a page returns no tasks before the reference is reached → `error: task number out of range: <ref>`
5. CompleteTask(listID, taskID)
6. Print `ok`

## 9.4 `gtask rm [--list <name>] <ref>`

Same validation and resolution as `done` (§9.3), but call DeleteTask instead of CompleteTask.


## 9.5 `gtask list [--page <n>] <list-name>`

1. Validate `<list-name>` provided, else → `error: list name required`
2. ResolveList(list-name)
   - This works for the default list too (e.g., `gtask list "My Tasks"`)
3. Fetch open tasks (for specified page, default 1)
4. Print section (even if empty; no "no tasks found" message for `gtask list`)


## 9.6 `gtask lists`

1. Fetch Lists (via ListLists)
2. Print each title on its own line (API order)
   - If `IsDefault` is true, append ` [default]` to the output name; otherwise do not append


## 9.7 `gtask createlist <list-name>`

1. Validate name:
   - if missing → `error: list name required`
   - if empty or whitespace-only after trimming → `error: list name required`
   - if starts with `-` → `error: unknown flag: <name>`
   - if list already exists → `error: list already exists: <name>` (§10.5)
2. CreateList
3. Print `ok`

**Implementation note:** To check for duplicates, call `ResolveList(name)`. If it returns successfully, the list exists → error. If it returns "list not found", proceed with creation.

## 9.7a `gtask addlist <list-name>`

Alias for `gtask createlist` with identical behavior.


## 9.8 `gtask rmlist [--force] <list-name>`

1. Validate `<list-name>` provided, else → `error: list name required`
2. ResolveList(list-name)
3. If resolved list is default list → `error: cannot delete default list`
4. Without `--force`:
   - HasOpenTasks (§7.3)
   - if any open tasks exist → `error: list not empty (use --force)`
5. DeleteList
6. Print `ok`


## 9.9 `gtask login`

1. Check if `oauth_client.json` exists in config dir
   - if not → `error: oauth_client.json not found in <config_dir>` (exit 2)
2. If `token.json` already exists and is valid (parseable and contains a non-empty refresh token; no network check):
   - Print `already logged in` (unless `--quiet`)
   - exit 0
3. Start OAuth flow:
   - Start local HTTP server on localhost:
     - Try ports starting at 8085, increment on failure
     - Retry up to 5 times if port is in use
     - If all fail → `error: could not bind to local port for OAuth callback` (exit 2)
   - Print URL to stderr: `Open this URL in your browser:\n<url>`
   - Do **not** attempt to auto-open a browser.
   - Wait for OAuth callback with authorization code (timeout: 5 minutes)
     - On timeout → `error: oauth callback timed out` (exit 2)
   - On callback received, respond with HTML: `<html><body><h1>Authentication successful</h1><p>You may close this window.</p></body></html>`
   - Exchange code for tokens (uses 30-second timeout for this HTTP call)
4. Save tokens to `token.json` (mode 0600)
5. Print `ok`


## 9.10 `gtask logout`

1. Check if `token.json` exists in config dir
   - if not → Print `not logged in` (unless `--quiet`), exit 0
2. Delete `token.json`
3. Print `ok`


## 10. Error Handling and Exit Codes

## 10.0 Unknown command

- `error: unknown command: <token>` (exit 1)
  - Example: `gtask --quiet` → `error: unknown command: --quiet`

## 10.1 Unknown flags

Within the **flag prefix** (before the first positional token), any token beginning with `-`
that is not a known flag for that command must produce:
1. stderr: `error: unknown flag: <token>`
2. exit code: 1

This includes cases where the user tries to use:
- task titles whose **first token** starts with `-`
- list names whose **first token** starts with `-`

Flags must precede positional args. After the first positional token, remaining tokens are positional
even if they start with `-`.


## 10.2 Not found / out of range

- Missing list: `error: list not found: <name>` (exit 1)
- Task number out of range: `error: task number out of range: <n>` (exit 1)
- Invalid task reference (non-numeric in v1): `error: invalid task reference: <ref>` (exit 1)
- Deleting default list: `error: cannot delete default list` (exit 1)

## 10.3 Missing required arguments

- `error: title required` (exit 1)
- `error: task reference required` (exit 1)
- `error: list name required` (exit 1)
- Missing flag value:
  - `error: flag needs an argument: --list` (exit 1)
  - `error: flag needs an argument: --config` (exit 1)
  - `error: flag needs an argument: --page` (exit 1)
- Invalid flag value:
  - `error: invalid page number: <value>` (exit 1) — must be positive integer


## 10.4 Backend failures

### Network/API failures:

`error: backend error: <summary>` (exit 3)

### Auth/config failures:

`error: auth error: <summary>` (exit 2)


## 10.5 List name already exists

`error: list already exists: <name>` (exit 1)


## 10.6 Not logged in

If a command requiring authentication is run without `oauth_client.json`:

`error: oauth_client.json not found in <config_dir>` (exit 2)

If a command requiring authentication is run without a valid `token.json`:

`error: not logged in (run: gtask login)` (exit 2)

Valid `token.json` means: file exists, is parseable, and contains a non-empty refresh token (no network check).

Commands requiring authentication: all except `login`, `logout`, `help`, `version`.


## 10.7 Login errors

- `error: oauth_client.json not found in <config_dir>` (exit 2)
- `error: could not bind to local port for OAuth callback` (exit 2)
- `error: oauth callback timed out` (exit 2)


## 11. Testing Strategy (MUST)

## 11.1 Hard requirement

All tests must run without real Google credentials and without real network calls.


## 11.2 Test layers

1. Command tests (primary)
    1. Create an in-memory fake implementation of service.Service
    2. Execute command handlers with controlled data
    	Assert:
        - stdout matches golden output exactly
        - stderr matches golden errors exactly
        - exit codes correct

These tests guarantee the CLI contract stays stable.

2. Backend tests (Google backend) with mocked HTTP
	1. Use `httptest.Server` (or custom RoundTripper) to simulate:
        - tasklists.list / insert / delete
        - tasks.list / insert / patch / delete
    2. Verify request shapes and that response parsing works


No OAuth browser flow required in tests:
- inject an HTTP client and token source into backend constructor

3. Login command tests
   - Login (`NeedsAuth() = false`) handles its own OAuth flow
   - Test approach:
     1. Create a test that starts the login flow
     2. Simulate the OAuth callback by making an HTTP request to the local callback server
     3. Mock the token exchange endpoint (Google's token URL) via custom HTTP transport
     4. Verify `token.json` is written correctly
   - Browser opening cannot be tested (OS-dependent); test the callback handler and token persistence

### 11.3 Golden tests

For output formatting:
1. store expected text output fixtures as .golden files
2. compare exact bytes

## 12. Implementation Plan (Steps to Do)

This plan incorporates your original steps and adds missing glue steps so development stays smooth.

1. **Understand Google Tasks API (read-only exploration)**
   1. Identify the exact methods needed:
      - list tasklists
      - list tasks
      - create task
      - patch task (complete)
      - delete task
      - create list
      - delete list
   2. Confirm required fields for output and task refs:
      - id, title, status (position is carried through but not used for sorting)

2. **Define the backend interface (service layer)**
   1. Create service.Service and data types (Task, TaskList)
   2. Ensure it's backend-agnostic and small

3. **Build a fake Service (in-memory)**
   1. Implement FakeService for tests:
      - store lists and tasks in-memory
      - allow deterministic test data setup

4. **Write command tests first**
   1. Implement commands against service.Service
   2. Add golden tests for:
      - `gtask` (list all open tasks)
      - `gtask` with no results (`no tasks found`)
      - `gtask` with partial failure (some lists succeed, one fails)
      - `gtask add` (default list)
      - `gtask add --list`
      - `gtask create` (alias of add)
      - `gtask done` (by number)
      - `gtask rm` (by number)
      - `gtask lists`
      - `gtask list <name>`
      - `gtask list <name> --page`
      - `gtask list` with default list name (locale-dependent title)
      - `gtask createlist`
      - `gtask addlist` (alias of createlist)
      - `gtask rmlist` and `gtask rmlist --force`
      - `gtask login` (with mocked OAuth callback and token exchange)
      - `gtask logout`
      - not logged in error
      - unknown command (e.g., `gtask --quiet`)
      - unknown flags (titles/lists whose first token starts with `-`)
      - empty/untitled tasks and lists

5. **Implement CLI parsing and dispatch**
   1. Command-first parsing, then parse flags with a single FlagSet (common + command-specific)
   2. Command registry + dispatch logic
   3. Ensure unknown flags behavior matches spec

6. **Implement Google backend (real code, but testable)**
   1. Backend constructor accepts injected HTTP client / token source
   2. Implement pagination loops for list operations
   3. Implement open tasks filtering
   4. Implement complete task via patch

7. **Backend tests with mocked HTTP**
   1. Use httptest to ensure backend correctly:
      - calls endpoints
      - respects query params
      - parses responses
      - handles errors and pagination

8. **Wire main()**
   1. Load config dir
   2. Load oauth_client.json
   3. Load/store token.json
   4. Create Google backend service
   5. Dispatch command
   6. Exit with returned code


## 13. Suggested Project Structure (Recommended)

```
gtask/
  cmd/gtask/main.go

  internal/
    cli/                # argument parsing, flags, dispatch
    commands/           # command implementations + registry
    output/             # formatters for golden output
    service/            # backend interface + types
    backend/
      googletasks/      # Google Tasks implementation (OAuth + API calls)
    config/             # XDG config directory, file IO, token persistence
                        # exports: Config struct with Dir, OAuthClientPath, TokenPath
    testutil/           # FakeService, golden helpers
```

## 14. Future Enhancements (Explicitly out of v1)

- `--json` output for scripting and stable ids
- `--ids` to show short-id in listings
- due dates / notes
- moving tasks between lists
- clearing completed tasks

## 15. Acceptance Checklist (Definition of Done)

- ✅ gtask builds into a single binary named gtask
- ✅ Config stored in $XDG_CONFIG_HOME/gtask (fallback $HOME/.config/gtask)
- ✅ `gtask login` performs OAuth flow and stores token
- ✅ `gtask logout` removes stored credentials
- ✅ Commands fail gracefully if not logged in
- ✅ OAuth token persisted; subsequent runs do not re-auth
- ✅ Completed tasks never shown
- ✅ Golden output formatting matches spec exactly
- ✅ Titles and list names whose first token starts with `-` fail as unknown flags
- ✅ Google backend is fully encapsulated behind service.Service
- ✅ Command system is registry-based (plugin-like)
- ✅ All tests pass with mocked Google API; no live calls
