# TaskFlow
[![Build Status](https://gdesouza.semaphoreci.com/badges/DevFlow/branches/master.svg?style=shields&key=dbff7292-7a82-4922-b626-72eefeef5b82)](https://gdesouza.semaphoreci.com/projects/DevFlow)
**Unified Task and Calendar Management Suite**

TaskFlow is an integrated CLI for tasks, calendars, and visualization. It helps you manage your tasks and calendar events from the command line, providing a unified workflow to boost your productivity.

## Features

- **Task Management:**
  - Add, list, edit, and complete tasks.
  - Search for tasks with a query.
  - Prioritize tasks based on due dates and calendar events.
  - View task statistics.
  - Interactive mode for a more user-friendly experience.
- **Calendar Integration:**
  - Import events from Google Calendar (`gcalcli` tsv format) and ICS files.
  - List calendar events.
  - Sync calendar events to your task list.
- **Web Interface:**
  - A simple web interface to view your tasks.
- **Notifications:**
  - Get notifications for upcoming tasks and events.
- **Customizable Configuration:**
  - Configure the storage path for your tasks and calendar events.

## Installation

### From Source

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/gdesouza/taskflow.git
    cd taskflow
    ```
2.  **Build the application:**
    ```bash
    make build
    ```
    This will create the `taskflow` binary in the `bin/` directory.

3.  **Install the binary (optional):**
    You can move the `taskflow` binary to a directory in your `PATH`, like `/usr/local/bin/`, to make it accessible from anywhere.
    ```bash
    sudo mv bin/taskflow /usr/local/bin/
    ```

### Prerequisites

- Go 1.16 or higher.

## Usage

### Filtering Examples

```bash
# Tasks containing both "deploy" and "staging" in title (default field)
taskflow task list --contains "deploy staging"

# Search words across title and notes
taskflow task list --contains "refactor cache" --contains-fields title,notes

# Filter by status AND word search in description
taskflow task list --status in-progress --contains performance --contains-fields description

# Combine tags, priority and multi-field contains
taskflow task list --tags backend,infra --priority high \
  --contains "latency alert" --contains-fields title,description,notes
```


### Task Management

- `taskflow task add [title] --due-date [RFC3339 format]`: Add a new task.
- `taskflow task list`: List all tasks. Filters: `--status`, `--priority`, `--tags tag1,tag2`, `--contains "word1 word2"`, `--contains-fields title,description,notes,link,tags` (AND match across chosen fields).
- `taskflow task done`: Mark a task as done.
- `taskflow task edit`: Edit a task's title.
- `taskflow task search [query]`: Search for tasks.
- `taskflow task stats`: Show task statistics.
- `taskflow task prioritize`: Prioritize tasks based on due dates and calendar events.
- `taskflow task schedule`: Create tasks from calendar events.
- `taskflow task interactive`: Start interactive mode (arrow keys navigate, Enter details, 'a' add, 'x' toggle done, 'f' filter, 's' sort, 'h' help, 'q'/Esc quit, auto-reloads on external file changes).

### Interactive Mode

Interactive mode provides a terminal UI for rapid task review and editing.

Key bindings:
- Arrow Up/Down: Navigate tasks
- Enter: View/edit selected task fields
- a: Add a new task (focus returns to list afterward)
- x: Toggle done/todo status
- f: Filter tasks (Status, Priority, Tags, Title Contains multi-word AND search, Clear Filters)
- s: Sort tasks (Priority, Status, Default [stable by ID])
- h: Toggle contextual help panel
- q or Esc: Exit list view (and from main menu choose another action or quit)

Other behaviors:
- Auto-Reload: The list checks the tasks file every 1s; external edits (CLI commands, editor) are reflected automatically while preserving selection when possible.
- Tasks File Path: Shown in the main menu so you can open it quickly in an editor.
- Truncation: Long lines are truncated to terminal width with an ellipsis.

Planned enhancements (not yet implemented):
- Event-based reload via fsnotify (lower latency)
- Flash indicator when a reload occurs
- Configurable reload interval / disable flag

- `taskflow task completion [bash|zsh|fish|powershell]`: Generate completion script.
- `taskflow task config`: Manage configuration.
- `taskflow task undo`: Undo the last operation.
- `taskflow task archive`: Archive all tasks with status=done into a separate archive file (supports `--dry-run`).

### Calendar Management

- `taskflow calendar import gcal`: Import from Google Calendar.
- `taskflow calendar import ics [file] [days_ahead]`: Import from an ICS file.
- `taskflow calendar list`: List calendar events.
- `taskflow calendar sync`: Sync calendar events to tasks.

### Other Commands

- `taskflow serve`: Start a web interface for task management.
- `taskflow notify`: Display notifications for upcoming tasks and calendar events.
- `taskflow version`: Print the version number.
- `taskflow display table`: Display tasks in a table.

## Remote Sync (GitHub Gist)

TaskFlow provides an MVP remote synchronization feature using a private (or public) GitHub Gist to store two files:
- `tasks.yaml`
- `tasks.archive.yaml`

This is plaintext (no encryption) and uses a simple remote-wins (pull) / blind-overwrite (push) strategy. Future improvements may add merge/conflict detection and encryption.

Prerequisites:
1. Create a GitHub Personal Access Token (classic) or fine-grained token with the `gist` scope.
2. Export it in your environment:
```bash
export TASKFLOW_GIST_TOKEN=ghp_yourtokenhere
```

Commands:
- `taskflow remote gist-init` : Creates a new gist (default private; add `--public` to make it public) and stores its ID in config under `remote.gist.id`.
- `taskflow remote gist-status` : Shows the configured gist ID and fetches its latest version hash.
- `taskflow remote gist-pull` : Downloads `tasks.yaml` and `tasks.archive.yaml` from the gist and overwrites local files (remote wins).
- `taskflow remote gist-push` : Uploads local `tasks.yaml` and archive file to the gist (blind overwrite).
- `taskflow remote gist-sync` : Stateful sync using stored metadata:
  * First run: pulls remote.
  * If remote unchanged but local changed: pushes local.
  * If remote advanced and local unchanged since last sync: pulls (fast-forward).
  * If both changed (divergence): aborts unless `--force --mode=push|pull` supplied.
  * Flags: `--force`, `--mode=push|pull`.

Behavior & Notes:
- Config location: `~/.config/taskflow/config.yaml` gains keys: `remote.gist.id`, `remote.gist.last_version`, `remote.gist.last_local_hash` after syncing.
- If you delete the gist manually, `gist-status` / `gist-pull` / `gist-push` / `gist-sync` will error until you re-run `gist-init`.
- Archive file naming follows: `tasks.yaml` -> `tasks.archive.yaml` (or `<name>.archive.<ext>` generically).
- `gist-pull` validates the main file contains a `tasks:` key before overwriting.
- Divergence handling (both changed) currently requires an explicit force with direction; no automatic merge yet.

Planned Enhancements:
- Three-way merge (structural task merging) during `gist-sync` for divergence without force.
- Conflict report listing differing task IDs and fields.
- Optional encryption (age / sops style) before uploading.

## Configuration

TaskFlow uses a configuration file located at `~/.config/taskflow/config.yaml`.

The following options are available:

- `storage.path`: The path to the YAML file where tasks are stored. Defaults to `~/.config/taskflow/tasks.yaml`.
- `calendar.storage.path`: The path to the YAML file where calendar events are stored. Defaults to `~/.config/taskflow/calendar.yaml`.

The application will create the configuration file with default values if it doesn't exist.

## Development

### Building

To build the application, run:
```bash
make build
```

### Testing

Run unit and integration tests (limited to packages with tests to avoid Go 1.25 coverage toolchain issue):
```bash
make test
```
This generates `coverage.out` and prints a one-line summary. View detailed HTML:
```bash
go tool cover -html=coverage.out
```
Generate full HTML report directly:
```bash
make coverage
```
Coverage currently focuses on internal logic (storage, filtering, config, task commands).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
