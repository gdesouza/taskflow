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
	"taskflow/internal/storage"
)

// execute helper reused (simplified local version)
func execRoot(t *testing.T, args ...string) string {
	buf := new(bytes.Buffer)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		w.Close()
		os.Stdout = old
		data, _ := io.ReadAll(r)
		buf.Write(data)
		return buf.String()
	}
	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	buf.Write(data)
	return buf.String()
}

func seedConfig(t *testing.T) string {
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
	return filepath.Join(configDir, "tasks.yaml")
}

func readTasksFile(t *testing.T, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read tasks: %v", err)
	}
	return string(data)
}

func TestAddCommandBasic(t *testing.T) {
	tasksPath := seedConfig(t)
	out := execRoot(t, "task", "add", "Write", "tests")
	if !strings.Contains(out, "Added task: Write tests") {
		t.Fatalf("output mismatch: %s", out)
	}
	content := readTasksFile(t, tasksPath)
	if !strings.Contains(content, "Write tests") {
		t.Fatalf("tasks file missing added task: %s", content)
	}
}

func TestAddCommandDueDate(t *testing.T) {
	tasksPath := seedConfig(t)
	date := "2025-10-02T15:00:00Z"
	_ = execRoot(t, "task", "add", "Deploy", "service", "--due-date", date)
	st, _ := storage.NewStorage(tasksPath)
	tasks, _ := st.ReadTasks()
	if len(tasks) != 1 || tasks[0].DueDate != date {
		t.Fatalf("expected due date stored, got %+v", tasks)
	}
}
