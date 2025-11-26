package task

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"taskflow/internal/config"
	"taskflow/internal/storage"
	"taskflow/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// InteractiveCmd launches the new modular Bubble Tea UI (skeleton).
func init() { TaskCmd.AddCommand(InteractiveCmd) }

var InteractiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"i"},
	Short:   "Start interactive task management mode",
	Run: func(cmd *cobra.Command, args []string) {
		storagePath := config.GetStoragePath()
		s, err := storage.NewStorage(storagePath)
		if err != nil {
			fmt.Printf("Error creating storage: %v\n", err)
			return
		}
		initialHash, _ := computeLocalHash()
		m := ui.New(s, initialHash, storagePath)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running program: %v\n", err)
		}
	},
}

// computeLocalHash returns a hash of tasks + archive for sync detection.
func computeLocalHash() (string, error) {
	mainPath := config.GetTasksFilePath()
	archPath := config.GetArchiveFilePath()
	m, err := os.ReadFile(mainPath)
	if err != nil {
		return "", err
	}
	a, err := os.ReadFile(archPath)
	if err != nil {
		if os.IsNotExist(err) {
			a = []byte("tasks: []\n")
		} else {
			return "", err
		}
	}
	h := sha256.Sum256(append(append(m, []byte("\n--\n")...), a...))
	return hex.EncodeToString(h[:]), nil
}

// promptSyncIfUnsynced would later prompt for syncing; currently informational.
func promptSyncIfUnsynced(initialHash string) error {
	lastHash := config.GetGistLastLocalHash()
	current, err := computeLocalHash()
	if err != nil {
		return nil
	}
	changedSinceLast := lastHash != "" && current != lastHash
	changedSinceStart := initialHash != "" && current != initialHash
	if !changedSinceLast && !changedSinceStart {
		return nil
	}
	fmt.Println("Unsynced changes detected. Run 'taskflow remote gist-sync' to sync.")
	return nil
}
