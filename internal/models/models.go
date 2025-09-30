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
