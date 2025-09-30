package task

import (
	"fmt"
	"os"
	"taskflow/internal/config"

	"taskflow/internal/storage"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// clearScreen clears the terminal screen (cross-platform)
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// truncateText truncates text to fit terminal width, Unicode-aware
func truncateText(text string, width int) string {
	if len([]rune(text)) > width {
		return string([]rune(text)[:width-1]) + "…"
	}
	return text
}

// getTerminalSize returns the current terminal width and height
func getTerminalSize() (width, height int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24 // default size
	}
	return w, h
}

// getOptimalListSize returns the optimal number of items to display
func getOptimalListSize() int {
	_, h := getTerminalSize()
	if h < 10 {
		return 5
	} else if h > 25 {
		return 20
	}
	return h - 5
}

// displayContent formats and displays content in the terminal
func displayContent(items []string) {
	clearScreen()
	w, _ := getTerminalSize()
	for _, item := range items {
		fmt.Println(truncateText(item, w))
	}
}

// InteractiveCmd is the cobra command for interactive mode
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

		clearScreen()
		fmt.Println("🚀 Welcome to TaskFlow Interactive Mode\n")
		for {
			action, err := showMainMenu()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			clearScreen()
			switch action {
			case "list":
				listTasks(s)
			case "add":
				addTask(s)
			case "done":
				doneTask(s)
			case "edit":
				editTask(s)
			case "search":
				searchTask(s)
			case "stats":
				showStats(s)
			case "quit":
				clearScreen()
				fmt.Println("👋 Goodbye!")
				return
			}
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
			clearScreen()
			fmt.Println("🚀 Welcome to TaskFlow Interactive Mode\n")
		}
	},
}

func showMainMenu() (string, error) {
	prompt := promptui.Select{
		Label: "What would you like to do? (Ctrl+C to quit)",
		Items: []string{
			"📋 List tasks",
			"➕ Add task",
			"✅ Mark tasks as done",
			"✏️ Edit task",
			"🔍 Search tasks",
			"📊 View statistics",
			"🚪 Quit",
		},
		Templates: &promptui.SelectTemplates{
			Help: `{{ "Use arrow keys to navigate, Enter to select, or Ctrl+C to quit" | faint }}`,
		},
	}
	index, _, err := prompt.Run()
	if err != nil {
		return "quit", nil
	}
	actions := []string{"list", "add", "done", "edit", "search", "stats", "quit"}
	return actions[index], nil
}

func listTasks(s *storage.Storage) {
	tasks, err := s.ReadTasks()
	if err != nil {
		fmt.Printf("Error reading tasks: %v\n", err)
		return
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}

	for _, task := range tasks {
		status := " "
		if task.Completed {
			status = "x"
		}
		fmt.Printf("[%s] %s\n", status, task.Title)
	}
}

func addTask(s *storage.Storage) {
	prompt := promptui.Prompt{
		Label: "Task title",
	}

	title, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	AddCmd.Run(AddCmd, []string{title})
}

func doneTask(s *storage.Storage) {
	DoneCmd.Run(DoneCmd, []string{})
}

func editTask(s *storage.Storage) {
	EditCmd.Run(EditCmd, []string{})
}

func searchTask(s *storage.Storage) {
	prompt := promptui.Prompt{
		Label: "Search query",
	}

	query, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	SearchCmd.Run(SearchCmd, []string{query})
}

func showStats(s *storage.Storage) {
	StatsCmd.Run(StatsCmd, []string{})
}