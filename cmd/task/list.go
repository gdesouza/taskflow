package task

import (
	"fmt"
	"sort"
	"strings"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"

	"github.com/spf13/cobra"
)

func init() {
	ListCmd.Flags().String("status", "", "Filter by status")
	ListCmd.Flags().String("priority", "", "Filter by priority")
	ListCmd.Flags().String("tags", "", "Filter by tags (comma-separated)")
	ListCmd.Flags().String("sort-by", "", "Sort by priority or status")
	TaskCmd.AddCommand(ListCmd)
}

var ListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List tasks",
	Aliases: []string{"ls", "show"},
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

		// Filtering
		status, _ := cmd.Flags().GetString("status")
		if status != "" {
			var filteredTasks []models.Task
			for _, task := range tasks {
				if task.Status == status {
					filteredTasks = append(filteredTasks, task)
				}
			}
			tasks = filteredTasks
		}

		priority, _ := cmd.Flags().GetString("priority")
		if priority != "" {
			var filteredTasks []models.Task
			for _, task := range tasks {
				if task.Priority == priority {
					filteredTasks = append(filteredTasks, task)
				}
			}
			tasks = filteredTasks
		}

		tags, _ := cmd.Flags().GetString("tags")
		if tags != "" {
			tagList := strings.Split(tags, ",")
			var filteredTasks []models.Task
			for _, task := range tasks {
				for _, t1 := range tagList {
					for _, t2 := range task.Tags {
						if t1 == t2 {
							filteredTasks = append(filteredTasks, task)
							break
						}
					}
				}
			}
			tasks = filteredTasks
		}

		// Sorting
		sortBy, _ := cmd.Flags().GetString("sort-by")
		if sortBy != "" {
			switch sortBy {
			case "priority":
				sort.Slice(tasks, func(i, j int) bool {
					return tasks[i].Priority > tasks[j].Priority
				})
			case "status":
				sort.Slice(tasks, func(i, j int) bool {
					return tasks[i].Status < tasks[j].Status
				})
			}
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return
		}

		for _, task := range tasks {
			status := " "
			if task.Status == "done" {
				status = "x"
			}
			fmt.Printf("[%s] (%s) %s\n", status, task.Priority, task.Title)
		}
	},
}
