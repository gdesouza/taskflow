package task

import (
	"fmt"
	"sort"
	"strings"
	"taskflow/internal/config"

	"taskflow/internal/storage"
	"taskflow/internal/tasks"

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

		allTasks, err := s.ReadTasks()
		if err != nil {
			fmt.Printf("Error reading tasks: %v\n", err)
			return
		}

		// Build filter options
		status, _ := cmd.Flags().GetString("status")
		priority, _ := cmd.Flags().GetString("priority")
		tagsFlag, _ := cmd.Flags().GetString("tags")
		contains, _ := cmd.Flags().GetString("contains")
		containsFields, _ := cmd.Flags().GetString("contains-fields")

		var tagList []string
		if tagsFlag != "" {
			for _, t := range strings.Split(tagsFlag, ",") {
				trim := strings.TrimSpace(t)
				if trim != "" {
					tagList = append(tagList, trim)
				}
			}
		}

		fieldSet := map[string]bool{}
		if containsFields != "" {
			for _, f := range strings.Split(containsFields, ",") {
				trim := strings.TrimSpace(strings.ToLower(f))
				if trim != "" {
					fieldSet[trim] = true
				}
			}
		}

		var words []string
		if contains != "" {
			for _, w := range strings.Fields(strings.ToLower(contains)) {
				words = append(words, w)
			}
		}

		opts := tasks.FilterOptions{
			Status:         status,
			Priority:       priority,
			Tags:           tagList,
			ContainsWords:  words,
			ContainsFields: fieldSet,
		}

		filtered := tasks.ApplyFilters(allTasks, opts)

		// Sorting
		sortBy, _ := cmd.Flags().GetString("sort-by")
		if sortBy != "" {
			switch sortBy {
			case "priority":
				prioRank := map[string]int{"highest": 4, "high": 3, "medium": 2, "low": 1}
				sort.SliceStable(filtered, func(i, j int) bool {
					return prioRank[filtered[i].Priority] > prioRank[filtered[j].Priority]
				})
			case "status":
				statusRank := map[string]int{"todo": 1, "in-progress": 2, "done": 3}
				sort.SliceStable(filtered, func(i, j int) bool {
					return statusRank[filtered[i].Status] < statusRank[filtered[j].Status]
				})
			}
		}

		if len(filtered) == 0 {
			fmt.Println("No tasks found.")
			return
		}

		for _, task := range filtered {
			status := " "
			if task.Status == "done" {
				status = "x"
			}
			fmt.Printf("[%s] (%s) %s\n", status, task.Priority, task.Title)
		}
	},
}
