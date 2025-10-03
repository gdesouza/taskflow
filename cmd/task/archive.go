package task

import (
	"fmt"
	"os"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	TaskCmd.AddCommand(ArchiveCmd)
	ArchiveCmd.Flags().Bool("dry-run", false, "Show what would be archived without modifying files")
}

// ArchiveCmd moves tasks with status=done into a separate archive file and removes them from the main tasks file.
var ArchiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive completed (done) tasks",
	Run: func(cmd *cobra.Command, args []string) {
		storagePath := config.GetStoragePath()
		s, err := storage.NewStorage(storagePath)
		if err != nil {
			fmt.Printf("Error creating storage: %v\n", err)
			return
		}

		all, err := s.ReadTasks()
		if err != nil {
			fmt.Printf("Error reading tasks: %v\n", err)
			return
		}

		var active []models.Task
		var completed []models.Task
		for _, t := range all {
			if t.Status == "done" {
				completed = append(completed, t)
			} else {
				active = append(active, t)
			}
		}

		if len(completed) == 0 {
			fmt.Println("No completed tasks to archive.")
			return
		}

		dry, _ := cmd.Flags().GetBool("dry-run")
		archivePath := config.GetArchiveFilePath()

		if dry {
			fmt.Printf("Would archive %d tasks to %s\n", len(completed), archivePath)
			return
		}

		// Backup main file before modifying so undo works.
		if err := s.Backup(); err != nil {
			fmt.Printf("Warning: failed to create backup: %v\n", err)
		}

		if err := appendToArchive(archivePath, completed); err != nil {
			fmt.Printf("Error writing archive: %v\n", err)
			return
		}

		if err := s.WriteTasks(active); err != nil {
			fmt.Printf("Error writing updated tasks: %v\n", err)
			return
		}

		fmt.Printf("Archived %d tasks → %s (remaining active: %d)\n", len(completed), archivePath, len(active))
	},
}

// appendToArchive reads existing archive (if any), appends new tasks, and writes back.
func appendToArchive(path string, tasks []models.Task) error {
	var existing models.TaskList
	if data, err := os.ReadFile(path); err == nil {
		if len(data) > 0 {
			_ = yaml.Unmarshal(data, &existing) // best-effort
		}
	}
	existing.Tasks = append(existing.Tasks, tasks...)
	out, err := yaml.Marshal(existing)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}
