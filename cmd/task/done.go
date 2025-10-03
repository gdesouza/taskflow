package task

import (
	"fmt"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var DoneCmd = &cobra.Command{
	Use:     "done",
	Short:   "Mark tasks as done",
	Aliases: []string{"complete", "finish"},
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

		var activeTasks []models.Task
		for _, task := range tasks {
			if task.Status != "done" {
				activeTasks = append(activeTasks, task)
			}
		}

		if len(activeTasks) == 0 {
			fmt.Println("No active tasks to mark as done.")
			return
		}

		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "-> {{ .Title | cyan }}",
			Inactive: "   {{ .Title | white }}",
			Selected: "=> {{ .Title | green }}",
		}

		prompt := promptui.Select{
			Label:     "Select a task to mark as done",
			Items:     activeTasks,
			Templates: templates,
		}

		i, _, err := prompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		doneTask := activeTasks[i]

		for i, task := range tasks {
			if task.ID == doneTask.ID {
				tasks[i].Status = "done"
				tasks[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
				break
			}
		}

		if err := s.WriteTasks(tasks); err != nil {
			fmt.Printf("Error writing tasks: %v\n", err)
			return
		}

		fmt.Printf("Marked task as done: %s\n", doneTask.Title)
	},
}
