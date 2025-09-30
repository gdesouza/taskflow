package cmd

import (
	"fmt"
	"taskflow/internal/config"
	"taskflow/internal/storage"
	"time"

	"github.com/spf13/cobra"
)

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Display notifications for upcoming tasks and calendar events",
	Run: func(cmd *cobra.Command, args []string) {
		// Check for upcoming tasks
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

		now := time.Now()
		in24Hours := now.Add(24 * time.Hour)

		fmt.Println("--- Upcoming Tasks ---")
		foundUpcomingTask := false
		for _, task := range tasks {
			if !task.Completed && task.DueDate != "" {
				dueDate, err := time.Parse(time.RFC3339, task.DueDate)
				if err == nil && dueDate.After(now) && dueDate.Before(in24Hours) {
					fmt.Printf("Task: %s (Due: %s)\n", task.Title, dueDate.Format("2006-01-02 15:04"))
					foundUpcomingTask = true
				}
			}
		}
		if !foundUpcomingTask {
			fmt.Println("No upcoming tasks in the next 24 hours.")
		}

		// Check for upcoming calendar events
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

		fmt.Println("\n--- Upcoming Calendar Events ---")
		foundUpcomingEvent := false
		for _, event := range events {
			startTime, err := time.Parse(time.RFC3339, event.StartTime)
			if err == nil && startTime.After(now) && startTime.Before(in24Hours) {
				fmt.Printf("Event: %s (Starts: %s)\n", event.Title, startTime.Format("2006-01-02 15:04"))
				foundUpcomingEvent = true
			}
		}
		if !foundUpcomingEvent {
			fmt.Println("No upcoming calendar events in the next 24 hours.")
		}
	},
}

func init() {
	RootCmd.AddCommand(notifyCmd)
}