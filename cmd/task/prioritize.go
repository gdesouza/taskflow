package task

import (
	"fmt"
	"taskflow/internal/config"
	"taskflow/internal/storage"
	"time"

	"github.com/spf13/cobra"
)

var PrioritizeCmd = &cobra.Command{
	Use:   "prioritize",
	Short: "Prioritize tasks based on due date and calendar events",
	Run: func(cmd *cobra.Command, args []string) {
		taskStoragePath := config.GetStoragePath()
		taskStorage, err := storage.NewStorage(taskStoragePath)
		if err != nil {
			fmt.Printf("Error creating task storage: %v\n", err)
			return
		}

		tasks, err := taskStorage.ReadTasks()
		if err != nil {
			fmt.Printf("Error reading tasks: %v\n", err)
			return
		}

		calendarStoragePath := config.GetCalendarStoragePath()
		calendarStorage, err := storage.NewStorage(calendarStoragePath)
		if err != nil {
			fmt.Printf("Error creating calendar storage: %v\n", err)
			return
		}

		events, err := calendarStorage.ReadCalendarEvents()
		if err != nil {
			fmt.Printf("Error reading calendar events: %v\n", err)
			return
		}

		now := time.Now()
		in24Hours := now.Add(24 * time.Hour)

		for i, task := range tasks {
			// Prioritize based on due date
			if task.DueDate != "" {
				dueDate, err := time.Parse(time.RFC3339, task.DueDate)
				if err == nil {
					if dueDate.After(now) && dueDate.Before(in24Hours) {
						tasks[i].Priority = 5 // Highest priority
					}
				}
			}

			// Prioritize based on calendar events
			for _, event := range events {
				if task.Title == event.Title {
					eventTime, err := time.Parse(time.RFC3339, event.StartTime)
					if err == nil {
						if eventTime.After(now) && eventTime.Before(in24Hours) {
							tasks[i].Priority = 5 // Highest priority
						}
					}
				}
			}
		}

		if err := taskStorage.WriteTasks(tasks); err != nil {
			fmt.Printf("Error writing tasks: %v\n", err)
			return
		}

		fmt.Println("Tasks prioritized successfully.")
	},
}
