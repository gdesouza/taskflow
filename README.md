# TaskFlow

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

### Task Management

- `taskflow task add [title] --due-date [RFC3339 format]`: Add a new task.
- `taskflow task list`: List all tasks. Supports filters: `--status`, `--priority`, `--tags tag1,tag2`, `--contains "word1 word2"` (all words must appear in the title).
- `taskflow task done`: Mark a task as done.
- `taskflow task edit`: Edit a task's title.
- `taskflow task search [query]`: Search for tasks.
- `taskflow task stats`: Show task statistics.
- `taskflow task prioritize`: Prioritize tasks based on due dates and calendar events.
- `taskflow task schedule`: Create tasks from calendar events.
- `taskflow task interactive`: Start interactive mode (list view supports: arrow keys navigate, Enter details, 'a' add task, 'x' toggle done, 'f' filter (now includes Title Contains multi-word filter), 's' sort, 'q' quit).
- `taskflow task completion [bash|zsh|fish|powershell]`: Generate completion script.
- `taskflow task config`: Manage configuration.
- `taskflow task undo`: Undo the last operation.

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

There are no tests yet.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
