package task

import (
	"fmt"
	"os"
	"golang.org/x/term"
	"github.com/spf13/cobra"
	"github.com/manifoldco/promptui"
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
				fmt.Println("[List tasks placeholder]")
			case "done":
				fmt.Println("[Mark tasks as done placeholder]")
			case "filter":
				fmt.Println("[Filter tasks placeholder]")
			case "stats":
				fmt.Println("[View statistics placeholder]")
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
			"✅ Mark tasks as done",
			"🔍 Filter tasks",
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
	actions := []string{"list", "done", "filter", "stats", "quit"}
	return actions[index], nil
}
