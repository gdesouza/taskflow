package task_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"taskflow/cmd"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"
)

func execRootCapture(t *testing.T, args ...string) string {
	buf := new(bytes.Buffer)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	_ = root.Execute()
	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	buf.Write(data)
	return buf.String()
}

func seedTasks(t *testing.T, tasks []models.Task) string {
	t.Helper()
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	configDir := filepath.Join(tempHome, ".config", config.AppName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	viper.Reset()
	if err := config.Init(); err != nil {
		t.Fatalf("config init: %v", err)
	}
	path := filepath.Join(configDir, "tasks.yaml")
	st, _ := storage.NewStorage(path)
	_ = st.WriteTasks(tasks)
	return path
}

func TestSearchFoundAndNotFound(t *testing.T) {
	seedTasks(t, []models.Task{
		{ID: "1", Title: "Implement login", Status: "todo"},
		{ID: "2", Title: "Write docs", Status: "done"},
	})
	out := execRootCapture(t, "task", "search", "login")
	if !strings.Contains(out, "Implement login") {
		t.Fatalf("expected match, got: %s", out)
	}
	if strings.Contains(out, "Write docs") {
		t.Fatalf("unexpected other task: %s", out)
	}

	out = execRootCapture(t, "task", "search", "missing")
	if !strings.Contains(out, "No tasks found") {
		t.Fatalf("expected not found message: %s", out)
	}
}
