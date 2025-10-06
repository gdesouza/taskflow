package task

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"
	"time"

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

// filterState stores the currently active filter so it can persist across
// task detail views, edits, auto-reloads, etc.
type filterState struct {
	Active bool
	Kind   string // Status | Priority | Tags | Title Contains
	Value  string // raw value entered
}

func applyFilter(all []models.Task, fs filterState) []models.Task {
	if !fs.Active {
		return all
	}
	var filtered []models.Task
	value := fs.Value
	lowerValue := strings.ToLower(value)
	words := strings.Fields(lowerValue)

	for _, task := range all {
		switch fs.Kind {
		case "Status":
			if task.Status == value {
				filtered = append(filtered, task)
			}
		case "Priority":
			if task.Priority == value {
				filtered = append(filtered, task)
			}
		case "Tags":
			for _, t := range task.Tags {
				if t == value {
					filtered = append(filtered, task)
					break
				}
			}
		case "Title Contains":
			// All words must be contained (case-insensitive)
			titleLower := strings.ToLower(task.Title)
			ok := true
			for _, w := range words {
				if !strings.Contains(titleLower, w) {
					ok = false
					break
				}
			}
			if ok {
				filtered = append(filtered, task)
			}
		}
	}
	return filtered
}

// runFilterPrompt prompts user and returns new filter state.
func runFilterPrompt(current filterState) filterState {
	prompt := promptui.Select{
		Label: "Filter by",
		Items: []string{"Status", "Priority", "Tags", "Title Contains", "Clear Filters"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		return current
	}
	if result == "Clear Filters" {
		return filterState{}
	}
	prompt2 := promptui.Prompt{Label: fmt.Sprintf("Enter %s", result)}
	value, err := prompt2.Run()
	if err != nil || strings.TrimSpace(value) == "" {
		return current
	}
	return filterState{Active: true, Kind: result, Value: strings.TrimSpace(value)}
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

	helpVisible := false

	// File modification tracking for auto-reload
	storagePath := config.GetStoragePath()
	lastMod := time.Time{}
	if fi, err := os.Stat(storagePath); err == nil {
		lastMod = fi.ModTime()
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	type keyEvent struct {
		char rune
		key  keyboard.Key
	}
	newKeyEvents := func() chan keyEvent {
		ch := make(chan keyEvent)
		go func() {
			for {
				char, key, err := keyboard.GetKey()
				if err != nil {
					close(ch)
					return
				}
				ch <- keyEvent{char: char, key: key}
			}
		}()
		return ch
	}
	keyEvents := newKeyEvents()

	// Persistent filter state
	var fState filterState

	for {
		// Render
		clearScreen()
		width, height := getTerminalSize()
		if helpVisible {
			fmt.Println("Tasks - Help")
			fmt.Println("  ↑/↓ : navigate")
			fmt.Println("  Enter: view details")
			fmt.Println("  a    : add task")
			fmt.Println("  x    : toggle done")
			fmt.Println("  f    : filter (persists until cleared)")
			fmt.Println("  s    : sort")
			fmt.Println("  q    : quit list view")
			fmt.Println("  h    : hide help")
			fmt.Println("  (auto-reloads on external file changes)")
		} else {
			if fState.Active {
				fmt.Printf("Tasks (filter: %s = '%s') (press 'h' for help)\n", fState.Kind, fState.Value)
			} else {
				fmt.Println("Tasks (press 'h' for help)")
			}
		}

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
			if fState.Active {
				fmt.Println("No tasks match current filter.")
			} else {
				fmt.Println("No tasks found.")
			}
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

		// Event handling
		select {
		case ev, ok := <-keyEvents:
			if !ok { // restart after a close (e.g., detail view reopening keyboard)
				if err := keyboard.Open(); err == nil {
					keyEvents = newKeyEvents()
					continue
				} else {
					return
				}
			}
			char := ev.char
			key := ev.key
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
					// Close current keyboard listener to avoid concurrent GetKey calls
					keyboard.Close()
					showTaskDetails(s, originalTasks, &tasks[selectedIndex])
					// Reopen keyboard and recreate keyEvents after returning from details view
					if err := keyboard.Open(); err != nil {
						panic(err)
					}
					keyEvents = newKeyEvents()
					// If filter active, task may now not match; re-apply filter
					if fState.Active {
						// Rebuild from originalTasks then re-apply
						fresh := applyFilter(originalTasks, fState)
						// Try to keep selection on same ID
						prevID := tasks[selectedIndex].ID
						tasks = fresh
						selectedIndex = 0
						for i, t := range tasks {
							if t.ID == prevID {
								selectedIndex = i
								break
							}
						}
						if selectedIndex >= len(tasks) {
							selectedIndex = len(tasks) - 1
							if selectedIndex < 0 {
								selectedIndex = 0
							}
						}
					}
				}
			case keyboard.KeyEsc:
				return
			}

			if char == 'q' {
				return
			}
			if char == 'h' {
				helpVisible = !helpVisible
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
					// Re-apply filter after status change
					if fState.Active {
						// refresh originalTasks from storage for accurate state
						if updated, err := s.ReadTasks(); err == nil {
							originalTasks = make([]models.Task, len(updated))
							copy(originalTasks, updated)
							if fi, err := os.Stat(storagePath); err == nil {
								lastMod = fi.ModTime()
							}
						}
						tasks = applyFilter(originalTasks, fState)
						if selectedIndex >= len(tasks) {
							selectedIndex = len(tasks) - 1
							if selectedIndex < 0 {
								selectedIndex = 0
							}
						}
					}
				}
			}
			if char == 'f' {
				keyboard.Close()
				fState = runFilterPrompt(fState)
				// Always rebuild tasks from originalTasks when (re)setting filter
				if fState.Active {
					tasks = applyFilter(originalTasks, fState)
				} else {
					// Clear -> show all
					tasks = make([]models.Task, len(originalTasks))
					copy(tasks, originalTasks)
				}
				selectedIndex = 0
				startIndex = 0
				if err := keyboard.Open(); err != nil {
					panic(err)
				}
				keyEvents = newKeyEvents()
			}
			if char == 's' {
				keyboard.Close()
				tasks = sortTasks(tasks)
				selectedIndex = 0
				startIndex = 0
				if err := keyboard.Open(); err != nil {
					panic(err)
				}
				keyEvents = newKeyEvents()
			}
			if char == 'a' {
				keyboard.Close()
				newTask, ok := interactiveCreateTask(s)
				if ok {
					updated, err := s.ReadTasks()
					if err == nil {
						originalTasks = make([]models.Task, len(updated))
						copy(originalTasks, updated)
						// Re-apply filter if active
						if fState.Active {
							tasks = applyFilter(originalTasks, fState)
						} else {
							tasks = originalTasks
						}
						for i, t := range tasks {
							if t.ID == newTask.ID {
								selectedIndex = i
								break
							}
						}
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
				keyEvents = newKeyEvents()
			}
		case <-ticker.C:
			if fi, err := os.Stat(storagePath); err == nil && fi.ModTime().After(lastMod) {
				if updated, err2 := s.ReadTasks(); err2 == nil {
					lastMod = fi.ModTime()
					originalTasks = make([]models.Task, len(updated))
					copy(originalTasks, updated)
					if fState.Active {
						tasks = applyFilter(originalTasks, fState)
					} else {
						tasks = updated
					}
					if selectedIndex >= len(tasks) {
						if len(tasks) == 0 {
							selectedIndex = 0
						} else {
							selectedIndex = len(tasks) - 1
						}
					}
				}
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

// sortTasks unchanged, used to sort the currently displayed slice
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
	// Ensure keyboard is opened exclusively for this details view
	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer keyboard.Close()
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
				fmt.Println("\x1b[7m" + line + "\x1b[0m")
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
			// Temporarily close raw mode for promptui interaction
			keyboard.Close()
			newValue := promptForValue(fields[selectedIndex], getFieldValue(task, fields[selectedIndex]))
			setFieldValue(task, fields[selectedIndex], newValue)
			s.UpdateTask(originalTasks, *task)
			// Re-enter raw mode for navigation
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
