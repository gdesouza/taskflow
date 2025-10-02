package task

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"

	"github.com/eiannone/keyboard"
	"github.com/google/uuid"
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
			action, err := showMainMenuCustom()
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

func showMainMenuCustom() (string, error) {
	menuItems := []string{
		"📋 List tasks",
		"➕ Add task",
		"✏️ Edit task",
		"🔍 Search tasks",
		"📊 View statistics",
		"🚪 Quit",
	}
	actions := []string{"list", "add", "edit", "search", "stats", "quit"}
	selectedIndex := 0

	if err := keyboard.Open(); err != nil {
		return "", err
	}
	defer keyboard.Close()

	storagePath := config.GetStoragePath()
	for {
		clearScreen()
		fmt.Println("What would you like to do? (use arrow keys to navigate, Enter to select, 'q' to quit)")
		fmt.Printf("Tasks file: %s\n\n", storagePath)

		for i, item := range menuItems {
			if i == selectedIndex {
				fmt.Println("\033[7m" + item + "\033[0m")
			} else {
				fmt.Println(item)
			}
		}

		char, key, err := keyboard.GetKey()
		if err != nil {
			return "", err
		}

		switch key {
		case keyboard.KeyArrowUp:
			if selectedIndex > 0 {
				selectedIndex--
			}
		case keyboard.KeyArrowDown:
			if selectedIndex < len(menuItems)-1 {
				selectedIndex++
			}
		case keyboard.KeyEnter:
			return actions[selectedIndex], nil
		case keyboard.KeyEsc:
			return "quit", nil
		}

		if char == 'q' {
			return "quit", nil
		}
	}
}

func listTasks(s *storage.Storage) {
	tasks, err := s.ReadTasks()
	if err != nil {
		fmt.Printf("Error reading tasks: %v\n", err)
		return
	}

	originalTasks := make([]models.Task, len(tasks))
	copy(originalTasks, tasks)

	selectedIndex := 0
	startIndex := 0

	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer keyboard.Close()

	for {
		clearScreen()
		width, height := getTerminalSize()
		fmt.Println("Tasks (use arrow keys to navigate, Enter to view details, 'a' to add, 'x' to toggle done, 'f' to filter, 's' to sort, 'q' to quit)")

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

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
		} else {
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
			if len(tasks) > 0 {
				showTaskDetails(s, originalTasks, &tasks[selectedIndex])
				if err := keyboard.Open(); err != nil {
					panic(err)
				}
			}
		case keyboard.KeyEsc:
			return
		}

		if char == 'q' {
			return
		}

		if char == 'x' {
			if len(tasks) > 0 {
				task := &tasks[selectedIndex]
				if task.Status == "done" {
					task.Status = "todo"
				} else {
					task.Status = "done"
				}
				s.UpdateTask(originalTasks, *task)
			}
		}

		if char == 'f' {
			keyboard.Close()
			tasks = filterTasks(originalTasks)
			selectedIndex = 0
			startIndex = 0
			if err := keyboard.Open(); err != nil {
				panic(err)
			}
		}

		if char == 's' {
			keyboard.Close()
			tasks = sortTasks(tasks)
			selectedIndex = 0
			startIndex = 0
			if err := keyboard.Open(); err != nil {
				panic(err)
			}
		}

		if char == 'a' {
			keyboard.Close()
			newTask, ok := interactiveCreateTask(s)
			if ok {
				// Reload tasks from storage to ensure consistency
				updated, err := s.ReadTasks()
				if err == nil {
					tasks = updated
					originalTasks = make([]models.Task, len(tasks))
					copy(originalTasks, tasks)
					// Find index of new task
					for i, t := range tasks {
						if t.ID == newTask.ID {
							selectedIndex = i
							break
						}
					}
					// Adjust startIndex if needed
					if selectedIndex < startIndex {
						startIndex = selectedIndex
					}
					width, height := getTerminalSize()
					_ = width
					if selectedIndex >= startIndex+height-2 {
						startIndex = selectedIndex - height + 3
						if startIndex < 0 {
							startIndex = 0
						}
					}
				}
			}
			if err := keyboard.Open(); err != nil {
				panic(err)
			}
		}
	}
}

func interactiveCreateTask(s *storage.Storage) (models.Task, bool) {
	// Title (required)
	titlePrompt := promptui.Prompt{Label: "Title (required)"}
	title, err := titlePrompt.Run()
	if err != nil || strings.TrimSpace(title) == "" {
		return models.Task{}, false
	}

	// Priority select
	prioritySelect := promptui.Select{Label: "Priority", Items: []string{"high", "medium", "low"}, Size: 3}
	_, priority, err := prioritySelect.Run()
	if err != nil {
		priority = "medium"
	}

	// Status select
	statusSelect := promptui.Select{Label: "Status", Items: []string{"todo", "in-progress", "done"}, Size: 3}
	_, status, err := statusSelect.Run()
	if err != nil {
		status = "todo"
	}

	// Link
	linkPrompt := promptui.Prompt{Label: "Link (optional)", Default: ""}
	link, _ := linkPrompt.Run()

	// Tags
	tagsPrompt := promptui.Prompt{Label: "Tags (comma separated, optional)", Default: ""}
	tagsStr, _ := tagsPrompt.Run()
	var tags []string
	for _, t := range strings.Split(tagsStr, ",") {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}

	// Notes
	notesPrompt := promptui.Prompt{Label: "Notes (optional)", Default: ""}
	notes, _ := notesPrompt.Run()

	// Due date
	duePrompt := promptui.Prompt{Label: "Due date (RFC3339, optional)", Default: ""}
	due, _ := duePrompt.Run()

	newTask := models.Task{
		ID:       uuid.New().String(),
		Title:    title,
		Status:   status,
		Priority: priority,
		Link:     link,
		Tags:     tags,
		Notes:    notes,
		DueDate:  due,
	}

	tasks, err := s.ReadTasks()
	if err != nil {
		fmt.Printf("Error reading tasks: %v\n", err)
		return models.Task{}, false
	}
	tasks = append(tasks, newTask)
	if err := s.WriteTasks(tasks); err != nil {
		fmt.Printf("Error writing tasks: %v\n", err)
		return models.Task{}, false
	}

	fmt.Printf("Added task: %s\n", newTask.Title)
	return newTask, true
}

func filterTasks(tasks []models.Task) []models.Task {
	prompt := promptui.Select{
		Label: "Filter by",
		Items: []string{"Status", "Priority", "Tags", "Title Contains", "Clear Filters"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		return tasks
	}

	if result == "Clear Filters" {
		return tasks
	}

	prompt2 := promptui.Prompt{
		Label: fmt.Sprintf("Enter %s", result),
	}
	value, err := prompt2.Run()
	if err != nil {
		return tasks
	}

	var filteredTasks []models.Task
	for _, task := range tasks {
		switch result {
		case "Status":
			if task.Status == value {
				filteredTasks = append(filteredTasks, task)
			}
		case "Priority":
			if task.Priority == value {
				filteredTasks = append(filteredTasks, task)
			}
		case "Tags":
			for _, t := range task.Tags {
				if t == value {
					filteredTasks = append(filteredTasks, task)
					break
				}
			}
		case "Title Contains":
			words := strings.Fields(strings.ToLower(value))
			titleLower := strings.ToLower(task.Title)
			all := true
			for _, w := range words {
				if !strings.Contains(titleLower, w) {
					all = false
					break
				}
			}
			if all {
				filteredTasks = append(filteredTasks, task)
			}
		}
	}
	return filteredTasks
}

func sortTasks(tasks []models.Task) []models.Task {
	prompt := promptui.Select{
		Label: "Sort by",
		Items: []string{"Priority", "Status", "Default"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		return tasks
	}

	switch result {
	case "Priority":
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].Priority > tasks[j].Priority
		})
	case "Status":
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].Status < tasks[j].Status
		})
	case "Default":
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].ID < tasks[j].ID
		})
	}
	return tasks
}

func showTaskDetails(s *storage.Storage, originalTasks []models.Task, task *models.Task) {
	fields := []string{"Title", "Status", "Priority", "Link", "Tags", "Notes"}
	selectedIndex := 0

	for {
		clearScreen()
		fmt.Printf("Task Details (use arrow keys to navigate, Enter to edit, 'q' or Esc to return)\n\n")

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
		fmt.Printf("\nID: %s\n", task.ID)

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
			s.UpdateTask(originalTasks, *task)
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
