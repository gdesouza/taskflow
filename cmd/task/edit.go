package task

import (
	"fmt"
	"taskflow/internal/config"
	"taskflow/internal/storage"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var EditCmd = &cobra.Command{
	Use:     "edit",
	Short:   "Edit task properties",
	Aliases: []string{"modify", "update"},
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

		if len(tasks) == 0 {
			fmt.Println("No tasks to edit.")
			return
		}

		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "-> {{ .Title | cyan }}",
			Inactive: "   {{ .Title | white }}",
			Selected: "=> {{ .Title | green }}",
		}

		prompt := promptui.Select{
			Label:     "Select a task to edit",
			Items:     tasks,
			Templates: templates,
		}

		i, _, err := prompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		editTask := tasks[i]

		prompt2 := promptui.Prompt{
			Label:   "New title",
			Default: editTask.Title,
		}

		newTitle, err := prompt2.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		for i, task := range tasks {
			if task.ID == editTask.ID {
				tasks[i].Title = newTitle
				tasks[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
				break
			}
		}

		if err := s.WriteTasks(tasks); err != nil {
			fmt.Printf("Error writing tasks: %v\n", err)
			return
		}

		fmt.Printf("Edited task: %s\n", newTitle)
	},
}
