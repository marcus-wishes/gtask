# gtask

A minimal, fast command-line interface for Google Tasks.

## Overview

gtask is a tiny CLI tool that provides a simple interface to Google Tasks. It's designed to be fast, scriptable, and focused on the essential operations you need to manage your tasks from the terminal.

**Key features:**
- List open tasks across all your lists
- Create, complete, and delete tasks
- Create and delete task lists
- Clean, script-friendly output
- No interactive prompts (except OAuth login)

## Installation

### From Source

Requires Go 1.21 or later:

```bash
git clone https://github.com/yourusername/gtask.git
cd gtask
go build -o gtask ./cmd/gtask
sudo mv gtask /usr/local/bin/  # or add to your PATH
```

## Setup

Before using gtask, you need to set up Google OAuth credentials.

### 1. Create OAuth Credentials

1. Go to [Google Cloud Console - Credentials](https://console.cloud.google.com/apis/credentials)
2. Create a new project (or select an existing one)
3. Enable the Google Tasks API:
   - Visit [Tasks API](https://console.cloud.google.com/apis/library/tasks.googleapis.com)
   - Click "Enable"
4. Configure the OAuth consent screen:
   - Go to "OAuth consent screen" in the sidebar
   - Choose "External" user type (or "Internal" for Workspace)
   - Fill in the required fields (app name, support email)
   - Add the scope: `https://www.googleapis.com/auth/tasks`
   - Add yourself as a test user (required for "External" apps in testing mode)
5. Create OAuth credentials:
   - Go to "Credentials" in the sidebar
   - Click "Create Credentials" > "OAuth client ID"
   - Choose "Desktop app" as the application type
   - Download the JSON file

### 2. Install Credentials

Save the downloaded JSON file to your config directory:

```bash
mkdir -p ~/.config/gtask
mv ~/Downloads/client_secret_*.json ~/.config/gtask/oauth_client.json
```

### 3. Login

Authenticate with your Google account:

```bash
gtask login
```

This will:
1. Print a URL to open in your browser
2. Start a local server to receive the OAuth callback
3. After you authorize in the browser, save your credentials

You only need to do this once. Your token is stored in `~/.config/gtask/token.json`.

## Usage

### List Tasks

```bash
# List all open tasks (default list + named lists with tasks)
gtask

# List tasks in a specific list
gtask list "Shopping"
gtask list Work

# Paginate through large lists (100 tasks per page)
gtask list "My Tasks" --page 2
```

**Output format:**
```
   1  Buy groceries
   2  Call mom
   3  Finish report
------------
Shopping
------------
       1  Milk
       2  Eggs
       3  Bread
```

Tasks from the default list appear without a header. Named lists are shown with a separator and title.

### Create Tasks

```bash
# Add to default list
gtask add Buy milk
gtask add "Call dentist tomorrow"

# Add to a specific list
gtask add --list Shopping Eggs
gtask add -l Shopping Eggs              # shorthand
gtask add --list "Work Projects" Review quarterly report
```

The `create` command is an alias for `add`:
```bash
gtask create Send email to team
```

### Complete Tasks

Mark a task as done using its number from the listing:

```bash
# Complete task #1 from default list
gtask done 1

# Complete task #2 from a specific list
gtask done --list Shopping 2
gtask done -l Shopping 2                # shorthand
```

### Delete Tasks

Remove a task entirely:

```bash
# Delete task #3 from default list
gtask rm 3

# Delete task #1 from a specific list
gtask rm --list Work 1
gtask rm -l Work 1                      # shorthand
```

### Manage Lists

```bash
# Show all lists
gtask lists

# Create a new list
gtask createlist "New Project"
gtask addlist Groceries  # alias for createlist

# Delete an empty list
gtask rmlist "Old Project"

# Delete a list with tasks (requires --force)
gtask rmlist --force "Old Project"
```

### Authentication

```bash
# Login (authenticate with Google)
gtask login

# Logout (remove stored credentials)
gtask logout
```

### Other Commands

```bash
# Show help
gtask help

# Show version
gtask version
```

## Common Flags

These flags are available on all commands (must appear after the command name):

| Flag | Description |
|------|-------------|
| `--quiet` | Suppress informational output (ok, no tasks found, etc.) |
| `--debug` | Print debug logs to stderr |
| `--config <dir>` | Override config directory |

### Command-Specific Flags

| Command | Flag | Shorthand | Description |
|---------|------|-----------|-------------|
| `add`, `create` | `--list <name>` | `-l <name>` | Add task to specified list |
| `done`, `rm` | `--list <name>` | `-l <name>` | Operate on task in specified list |
| `list` | `--page <n>` | | Page number (default: 1, 100 tasks/page) |
| `rmlist` | `--force` | | Delete list even if it has tasks |

Examples:
```bash
gtask add --quiet Buy milk      # No "ok" output
gtask lists --config /tmp/test  # Use alternate config dir
```

## Task References

Tasks are referenced by their number in the current listing (1, 2, 3, ...). These numbers correspond to the order returned by the Google Tasks API.

**Important:** Task numbers may change between runs if tasks are reordered, completed, or deleted. Always run `gtask` or `gtask list` to see the current numbering before operating on tasks.

## Output Format

gtask is designed for scripting:

- **Success output** goes to stdout
- **Error messages** go to stderr
- **Exit codes:**
  - `0` - Success
  - `1` - User error (bad arguments, not found, ambiguous)
  - `2` - Authentication/config error
  - `3` - Backend/API/network error

### Examples

**Task listing:**
```
   1  First task
   2  Second task
```
Format: 4-character right-aligned number, two spaces, title

**List sections:**
```
------------
List Name
------------
       1  Task in list
```

**Mutating commands:** Print `ok` on success (unless `--quiet`)

**Empty results:** Print `no tasks found` (unless `--quiet`)

## Configuration

gtask stores its configuration in `$XDG_CONFIG_HOME/gtask` (defaults to `~/.config/gtask`).

| File | Purpose |
|------|---------|
| `oauth_client.json` | Your Google OAuth credentials (you provide this) |
| `token.json` | Stored OAuth token (created by `gtask login`) |

The `token.json` file is created with mode 0600 for security.

## Limitations

Current limitations (v1):

- **No completed tasks** - Only open tasks are shown
- **No due dates** - Date fields are not supported
- **No notes** - Task notes/descriptions are not displayed
- **No subtasks** - Subtask hierarchy is flattened
- **No offline mode** - Requires network connectivity
- **Titles starting with `-`** - Not supported (parsed as flags)

## Troubleshooting

### "oauth_client.json not found"

You need to create OAuth credentials. See the [Setup](#setup) section.

### "not logged in (run: gtask login)"

Run `gtask login` to authenticate with your Google account.

### "token expired or revoked"

Your authentication has expired. Run `gtask login` again.

### OAuth callback issues

If the OAuth flow doesn't complete:

1. Make sure you've configured the OAuth consent screen in Google Cloud Console
2. Add yourself as a test user if the app is in "Testing" mode
3. Ensure ports 8085-8089 are available for the callback server
4. Check that your `oauth_client.json` is for a "Desktop app" type

### "list not found"

List names are case-insensitive but must match exactly otherwise. Check your list names with `gtask lists`.

## Development

### Building

```bash
go build ./cmd/gtask
```

### Testing

All tests run without network access using a fake service:

```bash
go test ./...
```

### Project Structure

```
gtask/
  cmd/gtask/           # Entry point
  internal/
    cli/               # Command dispatcher
    commands/          # Command implementations
    backend/googletasks/  # Google Tasks API client
    service/           # Service interface
    config/            # Configuration handling
    output/            # Output formatting
    testutil/          # Test utilities
```

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please read the AGENTS.md file for architecture guidelines.
