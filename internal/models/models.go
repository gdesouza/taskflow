package models

// Shared data models for tasks, calendar events, etc.

// Task represents a single task item.
type Task struct {
	ID          string
	Title       string
	Description string
	DueDate     string // ISO8601 format
	Completed   bool
	Priority    int
}

// CalendarEvent represents a calendar event.
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
	Tasks []SampleTask `yaml:"tasks"`
}

// SampleTask represents a task from the sample file.
type SampleTask struct {
	Title    string   `yaml:"title"`
	Status   string   `yaml:"status"`
	Priority string   `yaml:"priority"`
	Source   string   `yaml:"source"`
	Link     string   `yaml:"link"`
	Tags     []string `yaml:"tags"`
	DueDate  string   `yaml:"due"`
}
