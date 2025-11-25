package task

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"taskflow/cmd/remote"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// Bubble Tea interactive mode (with persistent filter + sort state)

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
		m := newModel(s, initialHash)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running program: %v\n", err)
		}
	},
}

type uiMode int

const (
	modeMenu uiMode = iota
	modeList
)

type filterState struct {
	Active      bool
	Kind, Value string
}

type model struct {
	storage     *storage.Storage
	initialHash string
	mode        uiMode
	menuIndex   int
	menuItems   []string
	actions     []string
	original    []models.Task
	tasks       []models.Task
	cursor      int
	start       int
	width       int
	height      int
	help        bool
	filter      filterState
	// sort persistence
	sortActive  bool
	sortKind    string // Priority | Status
	storagePath string
	lastMod     time.Time
	quitMessage string
}

func newModel(s *storage.Storage, initialHash string) model {
	menuItems := []string{"📋 List tasks", "➕ Add task", "✏️ Edit task", "🔍 Search tasks", "📊 View statistics", "🚪 Quit"}
	actions := []string{"list", "add", "edit", "search", "stats", "quit"}
	storagePath := config.GetStoragePath()
	fi, _ := os.Stat(storagePath)
	last := time.Time{}
	if fi != nil {
		last = fi.ModTime()
	}
	all, _ := s.ReadTasks()
	orig := make([]models.Task, len(all))
	copy(orig, all)
	return model{storage: s, initialHash: initialHash, mode: modeMenu, menuItems: menuItems, actions: actions, original: orig, tasks: orig, storagePath: storagePath, lastMod: last}
}

func (m model) Init() tea.Cmd { return tea.Batch(pollFileCmd(), tea.EnterAltScreen) }

func pollFileCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return filePollMsg{} })
}

type filePollMsg struct{}

type detailReturnMsg struct{ id string }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case filePollMsg:
		// preserve currently selected task ID across external reloads
		selectedID := ""
		if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
			selectedID = m.tasks[m.cursor].ID
		}
		if fi, err := os.Stat(m.storagePath); err == nil && fi.ModTime().After(m.lastMod) {
			updated, err2 := m.storage.ReadTasks()
			if err2 == nil {
				m.lastMod = fi.ModTime()
				m.original = make([]models.Task, len(updated))
				copy(m.original, updated)
				m.rebuildTasksPreservingSelection(selectedID)
			}
		}
		return m, pollFileCmd()
	case tea.KeyMsg:
		if m.mode == modeMenu {
			return m.handleMenuKey(msg)
		}
		return m.handleListKey(msg)
	case detailReturnMsg:
		updated, err := m.storage.ReadTasks()
		if err == nil {
			m.original = make([]models.Task, len(updated))
			copy(m.original, updated)
			m.rebuildTasksPreservingSelection(msg.id)
		}
	}
	return m, nil
}

// rebuildTasksPreservingSelection applies filter then sort; if focusID provided tries to keep selection on that task.
func (m *model) rebuildTasksPreservingSelection(focusID string) {
	// apply filter
	if m.filter.Active {
		m.tasks = applyFilter(m.original, m.filter)
	} else {
		m.tasks = m.original
	}
	// apply sort
	if m.sortActive {
		m.tasks = applySort(m.tasks, m.sortKind)
	}
	if focusID != "" {
		for i, t := range m.tasks {
			if t.ID == focusID {
				m.cursor = i
				break
			}
		}
	}
	if m.cursor >= len(m.tasks) {
		m.cursor = len(m.tasks) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
}

func (m model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		m.quitMessage = "👋 Goodbye!"
		return m, tea.Quit
	case "up", "k":
		if m.menuIndex > 0 {
			m.menuIndex--
		}
	case "down", "j":
		if m.menuIndex < len(m.menuItems)-1 {
			m.menuIndex++
		}
	case "enter":
		action := m.actions[m.menuIndex]
		switch action {
		case "list":
			m.mode = modeList
		case "add":
			newTask, ok := interactiveCreateTask(m.storage)
			if ok {
				m.reloadAfterMutation(newTask.ID)
			}
		case "edit":
			EditCmd.Run(EditCmd, []string{})
			m.reloadAfterMutation("")
		case "search":
			prompt := promptui.Prompt{Label: "Search query"}
			if q, err := prompt.Run(); err == nil {
				SearchCmd.Run(SearchCmd, []string{q})
			}
		case "stats":
			StatsCmd.Run(StatsCmd, []string{})
		case "quit":
			_ = promptSyncIfUnsynced(m.initialHash)
			m.quitMessage = "👋 Goodbye!"
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) reloadAfterMutation(focusID string) {
	updated, err := m.storage.ReadTasks()
	if err != nil {
		return
	}
	m.original = make([]models.Task, len(updated))
	copy(m.original, updated)
	m.rebuildTasksPreservingSelection(focusID)
}

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "ctrl+c":
		m.quitMessage = "👋 Goodbye!"
		return m, tea.Quit
	case "esc", "q":
		m.mode = modeMenu
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.tasks) > 0 {
			t := &m.tasks[m.cursor]
			id := t.ID
			showTaskDetails(m.storage, m.original, t)
			return m, func() tea.Msg { return detailReturnMsg{id: id} }
		}

	case "x":
		if len(m.tasks) > 0 {
			t := &m.tasks[m.cursor]
			if t.Status == "done" {
				t.Status = "todo"
			} else {
				t.Status = "done"
			}
			m.storage.UpdateTask(m.original, *t)
			m.reloadAfterMutation(t.ID)
		}
	case "/":
		prompt := promptui.Prompt{Label: "Filter text (Title Contains)", Default: m.filter.Value}
		if v, err := prompt.Run(); err == nil && strings.TrimSpace(v) != "" {
			m.filter = filterState{Active: true, Kind: "Title Contains", Value: strings.TrimSpace(v)}
			m.rebuildTasksPreservingSelection("")
			m.cursor = 0
			m.start = 0
		}
	case "f":
		m.filter = runFilterPrompt(m.filter)
		m.rebuildTasksPreservingSelection("")
		m.cursor, m.start = 0, 0
	case "s":
		m.promptSort()
		m.rebuildTasksPreservingSelection("")
		m.cursor, m.start = 0, 0
	case "a":
		newTask, ok := interactiveCreateTask(m.storage)
		if ok {
			m.reloadAfterMutation(newTask.ID)
		}
	case "h":
		m.help = !m.help
	}
	visible := m.height - 3
	if visible < 1 {
		visible = 1
	}
	if m.cursor < m.start {
		m.start = m.cursor
	}
	if m.cursor >= m.start+visible {
		m.start = m.cursor - visible + 1
	}
	return m, nil
}

func (m model) View() string {
	if m.quitMessage != "" {
		return m.quitMessage
	}
	if m.mode == modeMenu {
		return m.renderMenu()
	}
	return m.renderList()
}

func (m model) renderMenu() string {
	var b strings.Builder
	b.WriteString("🚀 Welcome to TaskFlow Interactive Mode\n")
	b.WriteString("What would you like to do? (↑/↓ Enter, q to quit)\n")
	b.WriteString(fmt.Sprintf("Tasks file: %s\n\n", m.storagePath))
	for i, item := range m.menuItems {
		line := item
		if i == m.menuIndex {
			line = inverse(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m model) renderList() string {
	var b strings.Builder
	if m.help {
		b.WriteString("Tasks - Help\n")
		b.WriteString("  ↑/↓ or j/k : navigate\n")
		b.WriteString("  Enter      : view/edit details\n")
		b.WriteString("  a          : add task\n")
		b.WriteString("  x          : toggle done\n")
		b.WriteString("  f          : filter menu\n")
		b.WriteString("  /          : text filter\n")
		b.WriteString("  s          : sort\n")
		b.WriteString("  h          : hide help\n")
		b.WriteString("  q / esc    : back to menu\n")
		b.WriteString("  (auto-reloads on external file changes)\n")
	} else {
		if m.filter.Active {
			b.WriteString(fmt.Sprintf("Tasks (filter: %s = '%s')", m.filter.Kind, m.filter.Value))
		} else {
			b.WriteString("Tasks")
		}
		if m.sortActive {
			b.WriteString(fmt.Sprintf(" (sorted: %s)", m.sortKind))
		}
		b.WriteString(" (press 'h' for help)\n")
	}
	if len(m.tasks) == 0 {
		if m.filter.Active {
			b.WriteString("No tasks match current filter.\n")
		} else {
			b.WriteString("No tasks found.\n")
		}
		return b.String()
	}
	visible := m.height - 3
	if visible < 1 {
		visible = 1
	}
	end := m.start + visible
	if end > len(m.tasks) {
		end = len(m.tasks)
	}
	for i := m.start; i < end; i++ {
		t := m.tasks[i]
		status := " "
		if t.Status == "done" {
			status = "x"
		}
		line := fmt.Sprintf("[%s] (%s) %s", status, t.Priority, t.Title)
		line = truncateText(line, max(10, m.width))
		if i == m.cursor {
			line = inverse(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func inverse(s string) string { return "\x1b[7m" + s + "\x1b[0m" }

func truncateText(text string, width int) string {
	if width <= 0 {
		return text
	}
	if len([]rune(text)) > width {
		return string([]rune(text)[:width-1]) + "…"
	}
	return text
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Filtering
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

// Sorting persistence
func applySort(tasks []models.Task, kind string) []models.Task {
	switch kind {
	case "Priority":
		sort.Slice(tasks, func(i, j int) bool { return tasks[i].Priority > tasks[j].Priority })
	case "Status":
		sort.Slice(tasks, func(i, j int) bool { return tasks[i].Status < tasks[j].Status })
	}
	return tasks
}

func (m *model) promptSort() {
	prompt := promptui.Select{Label: "Sort by", Items: []string{"Priority", "Status", "Default"}}
	_, result, err := prompt.Run()
	if err != nil {
		return
	}
	if result == "Default" {
		m.sortActive = false
		m.sortKind = ""
		return
	}
	m.sortActive = true
	m.sortKind = result
}

func runFilterPrompt(current filterState) filterState {
	prompt := promptui.Select{Label: "Filter by", Items: []string{"Status", "Priority", "Tags", "Title Contains", "Clear Filters"}}
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

// Creation
func interactiveCreateTask(s *storage.Storage) (models.Task, bool) {
	titlePrompt := promptui.Prompt{Label: "Title (required)"}
	title, err := titlePrompt.Run()
	if err != nil || strings.TrimSpace(title) == "" {
		return models.Task{}, false
	}
	prioritySelect := promptui.Select{Label: "Priority", Items: []string{"high", "medium", "low"}, Size: 3}
	_, priority, err := prioritySelect.Run()
	if err != nil {
		priority = "medium"
	}
	statusSelect := promptui.Select{Label: "Status", Items: []string{"todo", "in-progress", "done"}, Size: 3}
	_, status, err := statusSelect.Run()
	if err != nil {
		status = "todo"
	}
	linkPrompt := promptui.Prompt{Label: "Link (optional)", Default: ""}
	link, _ := linkPrompt.Run()
	tagsPrompt := promptui.Prompt{Label: "Tags (comma separated, optional)", Default: ""}
	tagsStr, _ := tagsPrompt.Run()
	var tags []string
	for _, t := range strings.Split(tagsStr, ",") {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	notesPrompt := promptui.Prompt{Label: "Notes (optional)", Default: ""}
	notes, _ := notesPrompt.Run()
	duePrompt := promptui.Prompt{Label: "Due date (RFC3339, optional)", Default: ""}
	due, _ := duePrompt.Run()
	newTask := models.Task{ID: uuid.New().String(), Title: title, Status: status, Priority: priority, Link: link, Tags: tags, Notes: notes, DueDate: due}
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

// Details
func showTaskDetails(s *storage.Storage, originalTasks []models.Task, task *models.Task) {
	fields := []string{"Title", "Status", "Priority", "Link", "Tags", "Notes"}
	selectedIndex := 0
	for {
		fmt.Printf("Task Details (use arrow keys to navigate, Enter to edit, 'q' to return)\n\n")
		for i, f := range fields {
			val := getFieldValue(task, f)
			line := fmt.Sprintf("%s: %s", f, val)
			if i == selectedIndex {
				fmt.Println(inverse(line))
			} else {
				fmt.Println(line)
			}
		}
		fmt.Printf("\nID: %s\n", task.ID)
		prompt := promptui.Select{Label: "Action", Items: []string{"Up", "Down", "Edit", "Done"}}
		_, choice, err := prompt.Run()
		if err != nil {
			return
		}
		switch choice {
		case "Up":
			if selectedIndex > 0 {
				selectedIndex--
			}
		case "Down":
			if selectedIndex < len(fields)-1 {
				selectedIndex++
			}
		case "Edit":
			newValue := promptForValue(fields[selectedIndex], getFieldValue(task, fields[selectedIndex]))
			setFieldValue(task, fields[selectedIndex], newValue)
			s.UpdateTask(originalTasks, *task)
		case "Done":
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
	prompt := promptui.Prompt{Label: fmt.Sprintf("Enter new %s", field), Default: defaultValue}
	res, err := prompt.Run()
	if err != nil {
		return defaultValue
	}
	return res
}

// Hash + sync helpers
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

func promptSyncIfUnsynced(initialHash string) error {
	lastHash := config.GetGistLastLocalHash()
	current, err := computeLocalHash()
	if err != nil {
		return nil
	}
	gistID := os.Getenv("TASKFLOW_GIST_TOKEN")
	changedSinceLast := lastHash != "" && current != lastHash
	changedSinceStart := initialHash != "" && current != initialHash
	if !changedSinceLast && !changedSinceStart {
		return nil
	}
	if gistID == "" {
		fmt.Println("Unsynced changes exist (no gist token set, skipping sync).")
		return nil
	}
	prompt := promptui.Prompt{Label: "Unsynced changes detected. Sync now? (y/N)", Default: "N"}
	ans, err := prompt.Run()
	if err != nil {
		return nil
	}
	ans = strings.ToLower(strings.TrimSpace(ans))
	if ans != "y" && ans != "yes" {
		fmt.Println("Skipped sync.")
		return nil
	}
	remote.GistSyncCmd.Run(remote.GistSyncCmd, []string{})
	return nil
}
