package task

import (
	"fmt"
	"os"
	"taskflow/internal/config"

	"github.com/spf13/cobra"
)

var UndoCmd = &cobra.Command{
	Use:     "undo",
	Short:   "Undo the last operation",
	Aliases: []string{"restore"},
	Run: func(cmd *cobra.Command, args []string) {
		storagePath := config.GetStoragePath()
		backupPath := storagePath + ".bak"

		data, err := os.ReadFile(backupPath)
		if err != nil {
			fmt.Printf("Error reading backup file: %v\n", err)
			return
		}

		if err := os.WriteFile(storagePath, data, 0644); err != nil {
			fmt.Printf("Error restoring backup: %v\n", err)
			return
		}

		fmt.Println("Last operation undone.")
	},
}