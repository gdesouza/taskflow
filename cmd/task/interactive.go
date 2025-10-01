package task

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"

	"github.com/eiannone/keyboard"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func init() {
	TaskCmd.AddCommand(InteractiveCmd)
}

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

		fmt.Println("🚀 Welcome to TaskFlow Interactive Mode")
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
			if action != "list" {
				fmt.Println("\nPress Enter to continue...")
				fmt.Scanln()
			}
			clearScreen()
			fmt.Println("🚀 Welcome to TaskFlow Interactive Mode")
		}
	},
}

func showMainMenu() (string, error) {
	_, height := getTerminalSize()
	size := height - 4
	if size < 5 {
		size = 5
	}

	prompt := promptui.Select{
		Label: "What would you like to do? (Ctrl+C to quit)",
		Items: []string{
			"📋 List tasks",
			"➕ Add task",
			"✏️ Edit task",
			"🔍 Search tasks",
			"📊 View statistics",
			"🚪 Quit",
		},
		Templates: &promptui.SelectTemplates{
			Help: `{{ "Use arrow keys to navigate, Enter to select, or Ctrl+C to quit" | faint }}`,
		},
		Size: size,
	}
	index, _, err := prompt.Run()
	if err != nil {
		return "quit", nil
	}
	actions := []string{"list", "add", "edit", "search", "stats", "quit"}
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

	sort.Slice(tasks, func(i, j int) bool {
		iID, _ := strconv.Atoi(tasks[i].ID)
		jID, _ := strconv.Atoi(tasks[j].ID)
		return iID < jID
	})

	selectedIndex := 0
	startIndex := 0

	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer keyboard.Close()

	for {
		clearScreen()
		width, height := getTerminalSize()
		fmt.Println("Tasks (use arrow keys to navigate, Enter to view details, 'x' to mark done, 'q' to quit)")

		// Adjust startIndex if selectedIndex is out of view
		if selectedIndex < startIndex {
			startIndex = selectedIndex
		}
		if selectedIndex >= startIndex+height-2 {
			startIndex = selectedIndex - height + 3
		}

		endIndex := startIndex + height - 3
		if endIndex > len(tasks) {
			endIndex = len(tasks)
		}

		for i := startIndex; i < endIndex; i++ {
			task := tasks[i]
			status := " "
			if task.Status == "done" {
				status = "x"
			}
			line := fmt.Sprintf("[%s] (%s) %s", status, task.Priority, task.Title)
			line = truncateText(line, width)
			if i == selectedIndex {
				fmt.Println("\033[7m" + line + "\033[0m")
			} else {
				fmt.Println(line)
			}
		}

		char, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}

		switch key {
		case keyboard.KeyArrowUp:
			if selectedIndex > 0 {
				selectedIndex--
			}
		case keyboard.KeyArrowDown:
			if selectedIndex < len(tasks)-1 {
				selectedIndex++
			}
		case keyboard.KeyEnter:
			showTaskDetails(s, &tasks[selectedIndex])
			if err := keyboard.Open(); err != nil {
				panic(err)
			}
		case keyboard.KeyEsc:
			return
		}

		if char == 'q' {
			return
		}

		if char == 'x' {
			task := &tasks[selectedIndex]
			if task.Status == "done" {
				task.Status = "todo"
			} else {
				task.Status = "done"
			}
			s.UpdateTask(*task)
		}
	}
}

func showTaskDetails(s *storage.Storage, task *models.Task) {
	fields := []string{"Title", "Status", "Priority", "Link", "Tags", "Notes"}
	selectedIndex := 0

	for {
		clearScreen()
		fmt.Printf("Task Details (use arrow keys to navigate, Enter to edit, 'q' or Esc to return)\n\n")
		id, _ := strconv.Atoi(task.ID)

		for i, field := range fields {
			var value string
			switch field {
			case "Title":
				value = task.Title
			case "Status":
				value = task.Status
			case "Priority":
				value = task.Priority
			case "Link":
				value = task.Link
			case "Tags":
				value = strings.Join(task.Tags, ", ")
			case "Notes":
				value = task.Notes
			}

			line := fmt.Sprintf("%s: %s", field, value)
			if i == selectedIndex {
				fmt.Println("[7m" + line + "[0m")
			} else {
				fmt.Println(line)
			}
		}
		fmt.Printf("\nID: %d\n", id)

		char, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}

		switch key {
		case keyboard.KeyArrowUp:
			if selectedIndex > 0 {
				selectedIndex--
			}
		case keyboard.KeyArrowDown:
			if selectedIndex < len(fields)-1 {
				selectedIndex++
			}
		case keyboard.KeyEnter:
			keyboard.Close() // Close keyboard before showing prompt
			newValue := promptForValue(fields[selectedIndex], getFieldValue(task, fields[selectedIndex]))
			setFieldValue(task, fields[selectedIndex], newValue)
			s.UpdateTask(*task)
			if err := keyboard.Open(); err != nil {
				panic(err)
			}
		case keyboard.KeyEsc:
			return
		}

		if char == 'q' {
			return
		}
	}
}

func getFieldValue(task *models.Task, field string) string {
	switch field {
	case "Title":
		return task.Title
	case "Status":
		return task.Status
	case "Priority":
		return task.Priority
	case "Link":
		return task.Link
	case "Tags":
		return strings.Join(task.Tags, ", ")
	case "Notes":
		return task.Notes
	}
	return ""
}

func setFieldValue(task *models.Task, field, value string) {
	switch field {
	case "Title":
		task.Title = value
	case "Status":
		task.Status = value
	case "Priority":
		task.Priority = value
	case "Link":
		task.Link = value
	case "Tags":
		task.Tags = strings.Split(value, ",")
	case "Notes":
		task.Notes = value
	}
}

func promptForValue(field, defaultValue string) string {
	prompt := promptui.Prompt{
		Label:   fmt.Sprintf("Enter new %s", field),
		Default: defaultValue,
	}

	result, err := prompt.Run()
	if err != nil {
		return defaultValue
	}
	return result
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
