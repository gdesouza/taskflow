package task_test

import (
	"bytes"
	"github.com/spf13/viper"
	"io"
	"os"
	"path/filepath"
	"strings"
	"taskflow/cmd"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"
	// import taskflow/cmd/task indirectly via root command init
	"testing"
)

// helper to execute root command with args and capture output
func execute(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	// Capture global stdout because list command uses fmt.Printf
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	// Reset list flags to defaults each run to avoid persistence
	if c, _, errFind := root.Find([]string{"task", "list"}); errFind == nil && c != nil {
		f := c.Flags()
		_ = f.Set("status", "")
		_ = f.Set("priority", "")
		_ = f.Set("tags", "")
		_ = f.Set("contains", "")
		_ = f.Set("contains-fields", "title")
	}
	err := root.Execute()
	w.Close()
	os.Stdout = old
	captured, _ := io.ReadAll(r)
	buf.Write(captured)
	return buf.String(), err
}

func setupConfig(t *testing.T, seed []models.Task) string {
	t.Helper()
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	configDir := filepath.Join(tempHome, ".config", config.AppName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tasksPath := filepath.Join(configDir, "tasks.yaml")
	viper.Reset()
	if err := config.Init(); err != nil {
		t.Fatalf("config init: %v", err)
	}
	st, _ := storage.NewStorage(tasksPath)
	if err := st.WriteTasks(seed); err != nil {
		t.Fatalf("seed write: %v", err)
	}
	return tasksPath
}

func TestListCommandBasicFilters(t *testing.T) {
	seed := []models.Task{
		{ID: "1", Title: "Login feature", Status: "todo", Priority: "high", Tags: []string{"auth", "backend"}},
		{ID: "2", Title: "Logout feature", Status: "done", Priority: "medium", Tags: []string{"auth"}},
		{ID: "3", Title: "Documentation", Status: "todo", Priority: "low", Tags: []string{"docs"}},
	}
	setupConfig(t, seed)

	out, err := execute("task", "list", "--status", "todo")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Count(out, "Login feature")+strings.Count(out, "Documentation") != 2 {
		t.Fatalf("expected two todo tasks, got output: %s", out)
	}

	out, _ = execute("task", "list", "--priority", "high")
	if !strings.Contains(out, "Login feature") || strings.Contains(out, "Logout feature") {
		t.Fatalf("priority filter mismatch: %s", out)
	}

	out, _ = execute("task", "list", "--tags", "docs")
	if !strings.Contains(out, "Documentation") || strings.Contains(out, "Login feature") {
		t.Fatalf("tags filter mismatch: %s", out)
	}
}

func TestListCommandContainsFields(t *testing.T) {
	seed := []models.Task{
		{ID: "1", Title: "Improve search", Status: "todo", Priority: "high", Description: "Add fuzzy search capability", Tags: []string{"feature"}},
		{ID: "2", Title: "Refactor code", Status: "todo", Priority: "medium", Notes: "reduce allocations in parser", Tags: []string{"cleanup"}},
	}
	setupConfig(t, seed)
	out, _ := execute("task", "list", "--contains", "fuzzy capability", "--contains-fields", "description")
	if !strings.Contains(out, "Improve search") || strings.Contains(out, "Refactor code") {
		t.Fatalf("contains description filter mismatch: %s", out)
	}
	out, _ = execute("task", "list", "--contains", "allocations parser", "--contains-fields", "notes")
	if !strings.Contains(out, "Refactor code") || strings.Contains(out, "Improve search") {
		t.Fatalf("contains notes filter mismatch: %s", out)
	}
}
