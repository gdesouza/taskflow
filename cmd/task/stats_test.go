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

func seedTasksGeneric(t *testing.T, tasks []models.Task) {
	t.Helper()
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	configDir := filepath.Join(tempHome, ".config", config.AppName)
	_ = os.MkdirAll(configDir, 0755)
	viper.Reset()
	_ = config.Init()
	path := filepath.Join(configDir, "tasks.yaml")
	st, _ := storage.NewStorage(path)
	_ = st.WriteTasks(tasks)
}

func execSimple(t *testing.T, args ...string) string {
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

func TestStatsCommand(t *testing.T) {
	seedTasksGeneric(t, []models.Task{
		{ID: "1", Title: "T1", Status: "todo"},
		{ID: "2", Title: "T2", Status: "done"},
		{ID: "3", Title: "T3", Status: "todo"},
	})
	out := execSimple(t, "task", "stats")
	if !strings.Contains(out, "Total tasks: 3") || !strings.Contains(out, "Completed tasks: 1") || !strings.Contains(out, "Pending tasks: 2") {
		t.Fatalf("unexpected stats output: %s", out)
	}
}
