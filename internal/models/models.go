package models

// Shared data models for tasks, calendar events, etc.

type CalendarEvent struct {
	ID          string
	Title       string
	StartTime   string // ISO8601 format
	EndTime     string // ISO8601 format
	Location    string
	Description string
}

// TaskList represents the top-level structure of the sample tasks file.
type TaskList struct {
	Tasks []Task `yaml:"tasks"`
}

// Task represents a task from the sample file.
type Task struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description,omitempty"`
	DueDate     string   `yaml:"due,omitempty"`
	Completed   bool     `yaml:"-"`
	Status      string   `yaml:"status"`
	Priority    string   `yaml:"priority"`
	PriorityInt int      `yaml:"-"`
	Source      string   `yaml:source"`
	Link        string   `yaml:"link"`
	Tags        []string `yaml:"tags,omitempty"`
	Notes       string   `yaml:"notes,omitempty"`
}
