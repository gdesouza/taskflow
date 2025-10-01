package task

import (
	"fmt"
	"taskflow/internal/config"
	"taskflow/internal/storage"

	"github.com/spf13/cobra"
)

var StatsCmd = &cobra.Command{
	Use:     "stats",
	Short:   "Show task statistics",
	Aliases: []string{"status", "overview"},
	Run: func(cmd *cobra.Command, args []string) {
		storagePath := config.GetStoragePath()
		s, err := storage.NewStorage(storagePath)
		if err != nil {
			fmt.Printf("Error creating storage: %v\n", err)
			return
		}

		tasks, err := s.ReadTasks()
		if err != nil {
			fmt.Printf("Error reading tasks: %v\n", err)
			return
		}

		totalTasks := len(tasks)
		var completedTasks int
		for _, task := range tasks {
			if task.Status == "done" {
				completedTasks++
			}
		}

		fmt.Printf("Total tasks: %d\n", totalTasks)
		fmt.Printf("Completed tasks: %d\n", completedTasks)
		fmt.Printf("Pending tasks: %d\n", totalTasks-completedTasks)
	},
}