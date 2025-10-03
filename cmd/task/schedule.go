package task

import (
	"fmt"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var ScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Create tasks based on calendar events",
	Run: func(cmd *cobra.Command, args []string) {
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

		taskStoragePath := config.GetStoragePath()
		taskStorage, err := storage.NewStorage(taskStoragePath)
		if err != nil {
			fmt.Printf("Error creating task storage: %v\n", err)
			return
		}

		var newTasks []models.Task
		for _, event := range events {
			task := models.Task{
				ID:        uuid.New().String(),
				Title:     event.Title,
				DueDate:   event.StartTime,
				UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			}
			newTasks = append(newTasks, task)
		}

		if len(newTasks) == 0 {
			fmt.Println("No calendar events to schedule.")
			return
		}

		existingTasks, err := taskStorage.ReadTasks()
		if err != nil {
			fmt.Printf("Error reading tasks: %v\n", err)
			return
		}

		existingTasks = append(existingTasks, newTasks...)

		if err := taskStorage.WriteTasks(existingTasks); err != nil {
			fmt.Printf("Error writing tasks: %v\n", err)
			return
		}

		fmt.Printf("Scheduled %d tasks from calendar events.\n", len(newTasks))
	},
}
