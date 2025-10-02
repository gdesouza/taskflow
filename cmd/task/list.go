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
	ListCmd.Flags().String("contains", "", "Filter by words contained in fields (space-separated)")
	ListCmd.Flags().String("contains-fields", "title", "Comma-separated list of fields to search: title,description,notes,link,tags (tags matched by tag value)")
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

		// Word filtering across selected fields (case-insensitive, AND logic)
		contains, _ := cmd.Flags().GetString("contains")
		containsFields, _ := cmd.Flags().GetString("contains-fields")
		if contains != "" {
			needleWords := strings.Fields(strings.ToLower(contains))
			fieldSet := map[string]bool{}
			for _, f := range strings.Split(containsFields, ",") {
				trim := strings.TrimSpace(strings.ToLower(f))
				if trim != "" {
					fieldSet[trim] = true
				}
			}
			if len(fieldSet) == 0 { // default
				fieldSet["title"] = true
			}
			var filteredTasks []models.Task
		TaskLoop:
			for _, task := range tasks {
				var haystackParts []string
				if fieldSet["title"] {
					haystackParts = append(haystackParts, task.Title)
				}
				if fieldSet["description"] {
					haystackParts = append(haystackParts, task.Description)
				}
				if fieldSet["notes"] {
					haystackParts = append(haystackParts, task.Notes)
				}
				if fieldSet["link"] {
					haystackParts = append(haystackParts, task.Link)
				}
				if fieldSet["tags"] {
					haystackParts = append(haystackParts, strings.Join(task.Tags, " "))
				}
				joined := strings.ToLower(strings.Join(haystackParts, " \n "))
				for _, w := range needleWords {
					if !strings.Contains(joined, w) {
						continue TaskLoop
					}
				}
				filteredTasks = append(filteredTasks, task)
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
