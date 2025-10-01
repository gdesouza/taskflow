package storage

import (
	"fmt"
	"os"
	"sort"
	"taskflow/internal/models"

	"gopkg.in/yaml.v3"
)

// Storage handles reading from and writing to the YAML file.
type Storage struct {
	filePath string
}

// NewStorage creates a new Storage instance.
func NewStorage(filePath string) (*Storage, error) {
	return &Storage{filePath: filePath}, nil
}

// ReadTasks reads all tasks from the YAML file.
func (s *Storage) ReadTasks() ([]models.Task, error) {
	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return []models.Task{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	var taskList models.TaskList
	if err := yaml.Unmarshal(data, &taskList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tasks: %w", err)
	}

	// Populate the internal fields of Task
	for i := range taskList.Tasks {
		taskList.Tasks[i].Completed = taskList.Tasks[i].Status == "done"
		taskList.Tasks[i].PriorityInt = convertPriority(taskList.Tasks[i].Priority)
		if taskList.Tasks[i].Description == "" {
			taskList.Tasks[i].Description = taskList.Tasks[i].Link
		}
	}

	return taskList.Tasks, nil
}

func convertPriority(priority string) int {
	switch priority {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// WriteTasks writes all tasks to the YAML file.
func (s *Storage) WriteTasks(tasks []models.Task) error {
	// Sort tasks by ID for consistent ordering
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	taskList := models.TaskList{
		Tasks: tasks,
	}

	data, err := yaml.Marshal(taskList)
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}
	return nil
}

// UpdateTask updates a single task in the YAML file.
func (s *Storage) UpdateTask(tasks []models.Task, updatedTask models.Task) error {
	for i, task := range tasks {
		if task.ID == updatedTask.ID {
			tasks[i] = updatedTask
			break
		}
	}

	return s.WriteTasks(tasks)
}

// Backup creates a backup of the current tasks file.
func (s *Storage) Backup() error {
	backupPath := s.filePath + ".bak"
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file to backup
		}
		return fmt.Errorf("failed to read tasks file for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	return nil
}

// ReadCalendarEvents reads all calendar events from the YAML file.
func (s *Storage) ReadCalendarEvents() ([]models.CalendarEvent, error) {
	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return []models.CalendarEvent{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read calendar events file: %w", err)
	}

	var events []models.CalendarEvent
	if err := yaml.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal calendar events: %w", err)
	}
	return events, nil
}

// WriteCalendarEvents writes all calendar events to the YAML file.
func (s *Storage) WriteCalendarEvents(events []models.CalendarEvent) error {
	// Sort events by StartTime for consistent ordering
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].StartTime < events[j].StartTime
	})

	data, err := yaml.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal calendar events: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write calendar events file: %w", err)
	}
	return nil
}
