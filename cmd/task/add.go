

package task

import (
	"fmt"
	"strings"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var dueDate string

var AddCmd = &cobra.Command{
	Use:     "add [title]",
	Short:   "Add a new task",
	Aliases: []string{"create", "new"},
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

		task := models.Task{
			ID:    uuid.New().String(),
			Title: strings.Join(args, " "),
			DueDate: dueDate,
		}

		tasks = append(tasks, task)

		if err := s.WriteTasks(tasks); err != nil {
			fmt.Printf("Error writing tasks: %v\n", err)
			return		}

		fmt.Printf("Added task: %s\n", task.Title)
	},
}

func init() {
	AddCmd.Flags().StringVar(&dueDate, "due-date", "", "Due date of the task (RFC3339 format)")
}
