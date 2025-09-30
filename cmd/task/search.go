package task

import (
	"fmt"
	"strings"
	"taskflow/internal/config"
	"taskflow/internal/storage"

	"github.com/spf13/cobra"
)

var SearchCmd = &cobra.Command{
	Use:     "search [query]",
	Short:   "Search for tasks",
	Aliases: []string{"find", "grep"},
	Args:    cobra.MinimumNArgs(1),
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

		query := strings.ToLower(strings.Join(args, " "))
		var foundTasks bool

		for _, task := range tasks {
			if strings.Contains(strings.ToLower(task.Title), query) {
				status := " "
				if task.Completed {
					status = "x"
				}
				fmt.Printf("[%s] %s\n", status, task.Title)
				foundTasks = true
			}
		}

		if !foundTasks {
			fmt.Println("No tasks found matching your query.")
		}
	},
}