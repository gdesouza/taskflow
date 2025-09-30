package calendar

import (
	"fmt"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var SyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync calendar events to tasks",
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

		tasks, err := taskStorage.ReadTasks()
		if err != nil {
			fmt.Printf("Error reading tasks: %v\n", err)
			return
		}

		var newTasks []models.Task
		for _, event := range events {
			task := models.Task{
				ID:      uuid.New().String(),
				Title:   event.Title,
				DueDate: event.StartTime,
			}
			newTasks = append(newTasks, task)
		}

		tasks = append(tasks, newTasks...)

		if err := taskStorage.WriteTasks(tasks); err != nil {
			fmt.Printf("Error writing tasks: %v\n", err)
			return
		}

		fmt.Printf("Synced %d calendar events to tasks.\n", len(newTasks))
	},
}

func init() {
	CalendarCmd.AddCommand(SyncCmd)
}