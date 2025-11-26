package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// Model is the root Bubble Tea model for the new interactive UI.
type Model struct {
	storage     *storage.Storage
	initialHash string

	width, height int

	// Task state
	allTasks []models.Task // canonical slice
	view     []models.Task // filtered/sorted slice
	cursor   int

	// Filter / search
	filterActive   bool
	filterKind     string
	filterValue    string
	enteringFilter bool
	filterInput    textinput.Model

	// Sort
	sortActive bool
	sortKind   string // Priority | Status

	// Detail box edit mode
	viewingDetail    bool
	detailFieldIndex int
	editingField     bool
	editInput        textinput.Model
	detailTask       *models.Task

	// Add task mode
	addingTask      bool
	addFieldIndex   int
	addEditingField bool
	newTask         models.Task

	// Delete confirmation mode
	confirmingDelete bool
	taskToDelete     *models.Task

	// Help screen mode
	showingHelp      bool
	helpScrollOffset int

	// File polling
	storagePath string
	lastMod     time.Time

	// Quit message
	quitMessage string
}

var fieldNames = []string{"Title", "Status", "Priority", "Link", "Tags", "Notes", "DueDate"}

// New constructs a new Model.
func New(s *storage.Storage, initialHash string, storagePath string) *Model {
	all, _ := s.ReadTasks()
	m := &Model{
		storage:     s,
		initialHash: initialHash,
		allTasks:    all,
		view:        all,
		storagePath: storagePath,
	}
	if fi, err := os.Stat(storagePath); err == nil {
		m.lastMod = fi.ModTime()
	}
	return m
}

// polling message
type filePollMsg struct{}

func pollFileCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return filePollMsg{} })
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	m.editInput = textinput.New()
	m.editInput.Prompt = "> "

	m.filterInput = textinput.New()
	m.filterInput.Prompt = "Filter: "
	m.filterInput.Placeholder = "Enter text to filter tasks..."

	return tea.Batch(pollFileCmd(), tea.EnterAltScreen)
}

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case filePollMsg:
		if fi, err := os.Stat(m.storagePath); err == nil && fi.ModTime().After(m.lastMod) {
			updated, err2 := m.storage.ReadTasks()
			if err2 == nil {
				m.lastMod = fi.ModTime()
				m.allTasks = updated
				m.rebuild("")
			}
		}
		return m, pollFileCmd()
	case tea.KeyMsg:
		if m.enteringFilter {
			return m.handleFilterInputKey(msg)
		}
		if m.showingHelp {
			return m.handleHelpKey(msg)
		}
		if m.confirmingDelete {
			return m.handleConfirmDeleteKey(msg)
		}
		if m.addingTask {
			return m.handleAddTaskKey(msg)
		}
		if m.viewingDetail {
			return m.handleDetailKey(msg)
		}
		return m.handleListKey(msg)
	}
	return m, nil
}

func (m *Model) handleDetailKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editingField {
		// editing a field value
		switch k.Type {
		case tea.KeyEsc:
			m.editingField = false
			return m, nil
		case tea.KeyEnter:
			val := strings.TrimSpace(m.editInput.Value())
			m.applyFieldEdit(val)
			m.editingField = false
			return m, nil
		}
		var cmd tea.Cmd
		m.editInput, cmd = m.editInput.Update(k)
		return m, cmd
	}

	// navigating detail box fields
	switch k.String() {
	case "esc", "q":
		m.viewingDetail = false
		m.detailTask = nil
		m.detailFieldIndex = 0
		return m, nil
	case "up", "k":
		if m.detailFieldIndex > 0 {
			m.detailFieldIndex--
		}
	case "down", "j":
		if m.detailFieldIndex < len(fieldNames)-1 {
			m.detailFieldIndex++
		}
	case "enter", "e":
		// start editing current field
		m.editingField = true
		m.editInput.SetValue(m.getFieldValue(fieldNames[m.detailFieldIndex]))
		m.editInput.Focus()
	}
	return m, nil
}

func (m *Model) handleAddTaskKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.addEditingField {
		// editing a field value
		switch k.Type {
		case tea.KeyEsc:
			m.addEditingField = false
			return m, nil
		case tea.KeyEnter:
			val := strings.TrimSpace(m.editInput.Value())
			m.applyAddFieldEdit(val)
			m.addEditingField = false
			return m, nil
		}
		var cmd tea.Cmd
		m.editInput, cmd = m.editInput.Update(k)
		return m, cmd
	}

	// navigating add task form fields
	switch k.String() {
	case "esc":
		// cancel adding task
		m.addingTask = false
		m.addFieldIndex = 0
		m.newTask = models.Task{}
		return m, nil
	case "up", "k":
		if m.addFieldIndex > 0 {
			m.addFieldIndex--
		}
	case "down", "j":
		if m.addFieldIndex < len(fieldNames)-1 {
			m.addFieldIndex++
		}
	case "enter", "e":
		// start editing current field
		m.addEditingField = true
		m.editInput.SetValue(m.getAddFieldValue(fieldNames[m.addFieldIndex]))
		m.editInput.Focus()
	case "ctrl+s":
		// save the task
		if m.newTask.Title == "" {
			// Title is required, don't save
			return m, nil
		}
		tasks := append(m.allTasks, m.newTask)
		if err := m.storage.WriteTasks(tasks); err == nil {
			m.allTasks = tasks
			m.reloadAfterMutation(m.newTask.ID)
		}
		// reset add mode
		m.addingTask = false
		m.addFieldIndex = 0
		m.newTask = models.Task{}
	}
	return m, nil
}

func (m *Model) handleConfirmDeleteKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "y", "Y":
		// Confirm delete
		if m.taskToDelete != nil {
			// Remove task from allTasks
			newTasks := make([]models.Task, 0, len(m.allTasks)-1)
			for _, t := range m.allTasks {
				if t.ID != m.taskToDelete.ID {
					newTasks = append(newTasks, t)
				}
			}
			// Write to storage
			if err := m.storage.WriteTasks(newTasks); err == nil {
				m.allTasks = newTasks
				m.rebuild("")
			}
		}
		// Reset confirmation state
		m.confirmingDelete = false
		m.taskToDelete = nil
	case "n", "N", "esc":
		// Cancel delete
		m.confirmingDelete = false
		m.taskToDelete = nil
	}
	return m, nil
}

func (m *Model) handleFilterInputKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEsc:
		// Cancel filter input
		m.enteringFilter = false
		m.filterInput.Blur()
		return m, nil
	case tea.KeyEnter:
		// Apply filter
		val := strings.TrimSpace(m.filterInput.Value())
		if val == "" {
			m.filterActive = false
			m.filterValue = ""
		} else {
			m.filterActive = true
			m.filterKind = "Title Contains"
			m.filterValue = val
		}
		m.rebuild("")
		m.enteringFilter = false
		m.filterInput.Blur()
		return m, nil
	}
	// Pass input to the text input bubble
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(k)
	return m, cmd
}

func (m *Model) handleHelpKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "q", "h", "?":
		m.showingHelp = false
		m.helpScrollOffset = 0 // reset scroll when closing
	case "up", "k":
		if m.helpScrollOffset > 0 {
			m.helpScrollOffset--
		}
	case "down", "j":
		m.helpScrollOffset++
		// The render function will clamp this to valid bounds
	}
	return m, nil
}

func (m *Model) handleListKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := k.String()
	switch key {
	case "ctrl+c", "q":
		m.quitMessage = "👋 Goodbye"
		return m, tea.Quit
	case "h", "?": // show help
		m.showingHelp = true
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.view)-1 {
			m.cursor++
		}
	case "x": // toggle status
		if len(m.view) > 0 {
			t := &m.view[m.cursor]
			// Cycle through: todo -> in-progress -> done -> todo
			switch t.Status {
			case "todo":
				t.Status = "in-progress"
			case "in-progress":
				t.Status = "done"
			case "done":
				t.Status = "todo"
			default:
				t.Status = "todo"
			}
			m.storage.UpdateTask(m.allTasks, *t)
			m.reloadAfterMutation(t.ID)
		}
	case "/": // text filter
		// Enter filter input mode
		m.enteringFilter = true
		m.filterInput.SetValue(m.filterValue)
		m.filterInput.Focus()
		return m, nil
	case "c": // clear filter
		if m.filterActive {
			m.filterActive = false
			m.filterValue = ""
			m.filterInput.SetValue("")
			m.rebuild("")
		}
	case "s": // cycle sort
		if !m.sortActive {
			m.sortActive = true
			m.sortKind = "Priority"
		} else if m.sortKind == "Priority" {
			m.sortKind = "Status"
		} else {
			m.sortActive = false
			m.sortKind = ""
		}
		m.rebuild("")
	case "a": // add task
		// Initialize new task with defaults
		m.newTask = models.Task{
			ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
			Status:   "todo",
			Priority: "medium",
		}
		m.addingTask = true
		m.addFieldIndex = 0
		m.addEditingField = false
	case "d": // delete task
		if len(m.view) > 0 {
			// Copy current task for confirmation
			current := m.view[m.cursor]
			m.taskToDelete = &current
			m.confirmingDelete = true
		}
	case "A": // archive task (Shift+A)
		if len(m.view) > 0 {
			current := m.view[m.cursor]
			if err := m.archiveTask(&current); err == nil {
				// Remove from allTasks
				newTasks := make([]models.Task, 0, len(m.allTasks)-1)
				for _, t := range m.allTasks {
					if t.ID != current.ID {
						newTasks = append(newTasks, t)
					}
				}
				// Write to storage
				if err := m.storage.WriteTasks(newTasks); err == nil {
					m.allTasks = newTasks
					m.rebuild("")
				}
			}
		}
	case "enter", "e": // open detail box
		if len(m.view) > 0 {
			// copy current task for detail view
			current := m.view[m.cursor]
			m.detailTask = &current
			m.viewingDetail = true
			m.detailFieldIndex = 0
		}
	}
	return m, nil
}

func (m *Model) getFieldValue(fieldName string) string {
	if m.detailTask == nil {
		return ""
	}
	switch fieldName {
	case "Title":
		return m.detailTask.Title
	case "Status":
		return m.detailTask.Status
	case "Priority":
		return m.detailTask.Priority
	case "Link":
		return m.detailTask.Link
	case "Tags":
		return strings.Join(m.detailTask.Tags, ", ")
	case "Notes":
		return m.detailTask.Notes
	case "DueDate":
		return m.detailTask.DueDate
	}
	return ""
}

func (m *Model) applyFieldEdit(val string) {
	if m.detailTask == nil || val == "" {
		return
	}
	fieldName := fieldNames[m.detailFieldIndex]
	// apply to detailTask
	switch fieldName {
	case "Title":
		m.detailTask.Title = val
	case "Status":
		if val == "todo" || val == "in-progress" || val == "done" {
			m.detailTask.Status = val
		}
	case "Priority":
		if val == "high" || val == "medium" || val == "low" {
			m.detailTask.Priority = val
		}
	case "Link":
		m.detailTask.Link = val
	case "Tags":
		m.detailTask.Tags = splitTags(val)
	case "Notes":
		m.detailTask.Notes = val
	case "DueDate":
		m.detailTask.DueDate = val
	}
	// persist to storage
	m.storage.UpdateTask(m.allTasks, *m.detailTask)
	m.reloadAfterMutation(m.detailTask.ID)
}

func (m *Model) getAddFieldValue(fieldName string) string {
	switch fieldName {
	case "Title":
		return m.newTask.Title
	case "Status":
		return m.newTask.Status
	case "Priority":
		return m.newTask.Priority
	case "Link":
		return m.newTask.Link
	case "Tags":
		return strings.Join(m.newTask.Tags, ", ")
	case "Notes":
		return m.newTask.Notes
	case "DueDate":
		return m.newTask.DueDate
	}
	return ""
}

func (m *Model) applyAddFieldEdit(val string) {
	fieldName := fieldNames[m.addFieldIndex]
	// apply to newTask
	switch fieldName {
	case "Title":
		m.newTask.Title = val
	case "Status":
		if val == "todo" || val == "in-progress" || val == "done" {
			m.newTask.Status = val
		}
	case "Priority":
		if val == "high" || val == "medium" || val == "low" {
			m.newTask.Priority = val
		}
	case "Link":
		m.newTask.Link = val
	case "Tags":
		m.newTask.Tags = splitTags(val)
	case "Notes":
		m.newTask.Notes = val
	case "DueDate":
		m.newTask.DueDate = val
	}
}

func splitTags(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ",") {
		v := strings.TrimSpace(t)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (m *Model) reloadAfterMutation(focusID string) {
	updated, err := m.storage.ReadTasks()
	if err != nil {
		return
	}
	m.allTasks = updated
	m.rebuild(focusID)
	// update detailTask if viewing
	if m.viewingDetail && m.detailTask != nil {
		for _, t := range m.allTasks {
			if t.ID == m.detailTask.ID {
				m.detailTask = &t
				break
			}
		}
	}
}

func (m *Model) rebuild(focusID string) {
	filtered := m.allTasks
	if m.filterActive {
		var tmp []models.Task
		words := strings.Fields(strings.ToLower(m.filterValue))
		for _, t := range filtered {
			if m.filterKind == "Title Contains" {
				lower := strings.ToLower(t.Title)
				ok := true
				for _, w := range words {
					if !strings.Contains(lower, w) {
						ok = false
						break
					}
				}
				if ok {
					tmp = append(tmp, t)
				}
			} else {
				tmp = append(tmp, t)
			}
		}
		filtered = tmp
	}
	if m.sortActive {
		switch m.sortKind {
		case "Priority":
			sort.Slice(filtered, func(i, j int) bool { return filtered[i].Priority > filtered[j].Priority })
		case "Status":
			sort.Slice(filtered, func(i, j int) bool { return filtered[i].Status < filtered[j].Status })
		}
	}
	m.view = filtered
	if focusID != "" {
		for i, t := range m.view {
			if t.ID == focusID {
				m.cursor = i
				break
			}
		}
	}
	if m.cursor >= len(m.view) {
		m.cursor = len(m.view) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// archiveTask appends a task to the archive file.
func (m *Model) archiveTask(task *models.Task) error {
	// Create backup before modifying
	if err := m.storage.Backup(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	archivePath := config.GetArchiveFilePath()
	if archivePath == "" {
		return fmt.Errorf("archive path not configured")
	}

	var existing models.TaskList
	if data, err := os.ReadFile(archivePath); err == nil {
		if len(data) > 0 {
			_ = yaml.Unmarshal(data, &existing)
		}
	}

	existing.Tasks = append(existing.Tasks, *task)
	out, err := yaml.Marshal(existing)
	if err != nil {
		return err
	}
	return os.WriteFile(archivePath, out, 0644)
}

// View renders UI.
func (m *Model) View() string {
	if m.quitMessage != "" {
		return m.quitMessage
	}

	if m.enteringFilter {
		return m.renderFilterInput()
	}

	if m.showingHelp {
		return m.renderHelpBox()
	}

	if m.confirmingDelete {
		return m.renderDeleteConfirmation()
	}

	if m.addingTask {
		return m.renderAddTaskBox()
	}

	if m.viewingDetail {
		return m.renderDetailBox()
	}

	return m.renderTaskList()
}

func (m *Model) renderTaskList() string {
	var content strings.Builder

	// Header
	header := "Tasks"
	if m.filterActive {
		header += fmt.Sprintf(" [filter: %s]", m.filterValue)
	}
	if m.sortActive {
		header += fmt.Sprintf(" [sort: %s]", m.sortKind)
	}
	content.WriteString(lipgloss.NewStyle().Bold(true).Render(header) + "\n\n")

	if len(m.view) == 0 {
		content.WriteString("No tasks.\n")
	} else {
		visible := m.height - 8 // account for box border and padding
		if visible < 1 {
			visible = 1
		}
		start := 0
		if m.cursor >= visible {
			start = m.cursor - visible + 1
		}
		end := start + visible
		if end > len(m.view) {
			end = len(m.view)
		}
		for i := start; i < end; i++ {
			t := m.view[i]

			// Status indicator
			statusIcon := "○"
			if t.Status == "done" {
				statusIcon = "✓"
			} else if t.Status == "in-progress" {
				statusIcon = "◐"
			}

			// Priority badge
			priorityBadge := ""
			switch t.Priority {
			case "high":
				priorityBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("[H]")
			case "medium":
				priorityBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("[M]")
			case "low":
				priorityBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Render("[L]")
			}

			// Status label
			statusLabel := ""
			switch t.Status {
			case "done":
				statusLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("28")).Render("Done")
			case "in-progress":
				statusLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("In Progress")
			case "todo":
				statusLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Todo")
			}

			line := fmt.Sprintf("%s %s %-11s %s", statusIcon, priorityBadge, statusLabel, t.Title)
			if i == m.cursor {
				line = invert(line)
			}
			content.WriteString(line + "\n")
		}
	}

	content.WriteString("\n")
	content.WriteString(statusStyle.Render(" q:quit h:help ↑/↓:nav x:toggle /:filter c:clear s:sort a:add d:delete A:archive e:edit "))

	// Box style for task list
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(m.width - 4)

	return boxStyle.Render(content.String())
}

func (m *Model) renderDetailBox() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Task Details") + "\n\n")

	for i, fieldName := range fieldNames {
		val := m.getFieldValue(fieldName)
		line := fmt.Sprintf("%-10s: %s", fieldName, val)
		if i == m.detailFieldIndex {
			if m.editingField {
				line = fmt.Sprintf("%-10s: %s", fieldName, m.editInput.View())
			} else {
				line = invert(line)
			}
		}
		content.WriteString(line + "\n")
	}

	content.WriteString("\n")
	if m.editingField {
		content.WriteString(statusStyle.Render(" [Enter:save Esc:cancel] "))
	} else {
		content.WriteString(statusStyle.Render(" [↑/↓:navigate e:edit Esc:close] "))
	}

	box := boxStyle.Render(content.String())

	// center the box
	h := lipgloss.Height(box)
	w := lipgloss.Width(box)
	vPad := (m.height - h) / 2
	hPad := (m.width - w) / 2
	if vPad < 0 {
		vPad = 0
	}
	if hPad < 0 {
		hPad = 0
	}

	positioned := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	return positioned
}

func (m *Model) renderAddTaskBox() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Add New Task") + "\n\n")

	for i, fieldName := range fieldNames {
		val := m.getAddFieldValue(fieldName)

		// Add hints for specific fields
		hint := ""
		switch fieldName {
		case "Status":
			hint = " (todo/in-progress/done)"
		case "Priority":
			hint = " (high/medium/low)"
		case "Tags":
			hint = " (comma separated)"
		case "DueDate":
			hint = " (RFC3339 format)"
		}

		line := fmt.Sprintf("%-10s: %s%s", fieldName, val, hint)
		if i == m.addFieldIndex {
			if m.addEditingField {
				line = fmt.Sprintf("%-10s: %s", fieldName, m.editInput.View())
			} else {
				line = invert(line)
			}
		}
		content.WriteString(line + "\n")
	}

	content.WriteString("\n")
	if m.addEditingField {
		content.WriteString(statusStyle.Render(" [Enter:save Esc:cancel] "))
	} else {
		if m.newTask.Title == "" {
			content.WriteString(statusStyle.Render(" [↑/↓:navigate e:edit Esc:cancel] (Title required to save) "))
		} else {
			content.WriteString(statusStyle.Render(" [↑/↓:navigate e:edit Ctrl+S:save Esc:cancel] "))
		}
	}

	box := boxStyle.Render(content.String())

	// center the box
	positioned := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	return positioned
}

func (m *Model) renderDeleteConfirmation() string {
	if m.taskToDelete == nil {
		return "Error: no task to delete"
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")). // Red border for warning
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("Delete Task") + "\n\n")
	content.WriteString("Are you sure you want to delete this task?\n\n")

	// Show task details
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Title: ") + m.taskToDelete.Title + "\n")
	if m.taskToDelete.Status != "" {
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Status: ") + m.taskToDelete.Status + "\n")
	}
	if m.taskToDelete.Priority != "" {
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Priority: ") + m.taskToDelete.Priority + "\n")
	}

	content.WriteString("\n")
	content.WriteString(statusStyle.Render(" [Y:confirm N/Esc:cancel] "))

	box := boxStyle.Render(content.String())

	// center the box
	positioned := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	return positioned
}

func (m *Model) renderFilterInput() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Filter Tasks") + "\n\n")
	content.WriteString("Enter text to filter tasks by title:\n\n")
	content.WriteString(m.filterInput.View() + "\n\n")
	content.WriteString(statusStyle.Render(" [Enter:apply Esc:cancel] "))

	box := boxStyle.Render(content.String())

	// center the box
	positioned := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	return positioned
}

func (m *Model) renderHelpBox() string {
	// Build help content as lines
	helpLines := []string{
		lipgloss.NewStyle().Bold(true).Render("Keyboard Shortcuts"),
		"",
		lipgloss.NewStyle().Bold(true).Render("Navigation:"),
		"  ↑/k         Move cursor up",
		"  ↓/j         Move cursor down",
		"  h/?         Show this help screen",
		"  q/Ctrl+C    Quit application",
		"",
		lipgloss.NewStyle().Bold(true).Render("Task Actions:"),
		"  x           Toggle task status (todo → in-progress → done)",
		"  a           Add new task",
		"  e/Enter     Edit task details",
		"  d           Delete task",
		"  A (Shift+A) Archive task",
		"",
		lipgloss.NewStyle().Bold(true).Render("Filtering & Sorting:"),
		"  /           Filter tasks by title",
		"  c           Clear filter",
		"  s           Cycle sort (Priority → Status → None)",
		"",
		lipgloss.NewStyle().Bold(true).Render("Detail View:"),
		"  ↑/↓         Navigate between fields",
		"  e/Enter     Edit current field",
		"  Esc         Close detail view",
		"",
		lipgloss.NewStyle().Bold(true).Render("Add Task View:"),
		"  ↑/↓         Navigate between fields",
		"  e/Enter     Edit current field",
		"  Ctrl+S      Save task",
		"  Esc         Cancel",
	}

	// Calculate available height for content (account for border, padding, and status line)
	boxBorder := 2  // top and bottom border
	boxPadding := 2 // top and bottom padding
	statusLine := 2 // status bar and blank line
	availableHeight := m.height - boxBorder - boxPadding - statusLine
	if availableHeight < 5 {
		availableHeight = 5 // minimum height
	}

	totalLines := len(helpLines)

	// Clamp scroll offset
	maxScroll := totalLines - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.helpScrollOffset > maxScroll {
		m.helpScrollOffset = maxScroll
	}
	if m.helpScrollOffset < 0 {
		m.helpScrollOffset = 0
	}

	// Slice visible lines
	start := m.helpScrollOffset
	end := start + availableHeight
	if end > totalLines {
		end = totalLines
	}
	visibleLines := helpLines[start:end]

	// Build content
	var content strings.Builder
	for _, line := range visibleLines {
		content.WriteString(line + "\n")
	}

	// Add scroll indicators and status
	scrollInfo := ""
	if maxScroll > 0 {
		if m.helpScrollOffset > 0 && m.helpScrollOffset < maxScroll {
			scrollInfo = " ↑↓ scroll "
		} else if m.helpScrollOffset == 0 && maxScroll > 0 {
			scrollInfo = " ↓ more "
		} else if m.helpScrollOffset == maxScroll {
			scrollInfo = " ↑ more "
		}
	}

	content.WriteString("\n")
	content.WriteString(statusStyle.Render(fmt.Sprintf(" [h/?/Esc/q: close]%s ", scrollInfo)))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(70)

	box := boxStyle.Render(content.String())

	// center the box
	positioned := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	return positioned
}

func invert(s string) string { return cursorStyle.Render(s) }
