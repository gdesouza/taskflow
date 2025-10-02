package task_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"taskflow/cmd"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"
)

func TestPrioritizeCommand(t *testing.T) {
	now := time.Now().UTC()
	dueSoon := now.Add(2 * time.Hour).Format(time.RFC3339)
	eventSoon := now.Add(3 * time.Hour).Format(time.RFC3339)
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	cfgDir := filepath.Join(tempHome, ".config", config.AppName)
	_ = os.MkdirAll(cfgDir, 0755)
	viper.Reset()
	if err := config.Init(); err != nil {
		t.Fatalf("config init: %v", err)
	}
	// seed tasks
	tasksPath := filepath.Join(cfgDir, "tasks.yaml")
	stTasks, _ := storage.NewStorage(tasksPath)
	_ = stTasks.WriteTasks([]models.Task{
		{ID: "1", Title: "DueSoon", DueDate: dueSoon, Priority: "low", Status: "todo"},
		{ID: "2", Title: "EventSoon", Priority: "low", Status: "todo"},
	})
	// seed calendar events
	calPath := filepath.Join(cfgDir, "calendar.yaml")
	stCal, _ := storage.NewStorage(calPath)
	_ = stCal.WriteCalendarEvents([]models.CalendarEvent{{Title: "EventSoon", StartTime: eventSoon}})
	// run command
	buf := new(bytes.Buffer)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"task", "prioritize"})
	_ = root.Execute()
	w.Close()
	os.Stdout = old
	_, _ = io.ReadAll(r)
	// verify tasks updated
	updated, _ := stTasks.ReadTasks()
	foundDue := false
	foundEvent := false
	for _, tk := range updated {
		if tk.ID == "1" && tk.Priority == "highest" {
			foundDue = true
		}
		if tk.ID == "2" && tk.Priority == "highest" {
			foundEvent = true
		}
	}
	if !foundDue || !foundEvent {
		t.Fatalf("expected both tasks promoted, got %+v", updated)
	}
}
