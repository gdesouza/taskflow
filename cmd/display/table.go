package display

import (
	"fmt"
	"taskflow/internal/config"
	"taskflow/internal/storage"
	"taskflow/internal/table"

	"github.com/spf13/cobra"
)

var compact bool

var TableCmd = &cobra.Command{
	Use:   "table",
	Short: "Display tasks in a table",
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

		table.RenderTasks(tasks, compact)
	},
}

func init() {
	TableCmd.Flags().BoolVar(&compact, "compact", false, "Show compact table (status, title)")
	DisplayCmd.AddCommand(TableCmd)
}