package task_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/spf13/viper"
	"taskflow/cmd"
	"taskflow/internal/config"
	"taskflow/internal/models"
	"taskflow/internal/storage"
)

func seedAndExec(t *testing.T, tasks []models.Task, args ...string) string {
	buf := new(bytes.Buffer)
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	configDir := filepath.Join(tempHome, ".config", config.AppName)
	_ = os.MkdirAll(configDir, 0755)
	viper.Reset()
	_ = config.Init()
	path := filepath.Join(configDir, "tasks.yaml")
	st, _ := storage.NewStorage(path)
	_ = st.WriteTasks(tasks)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	// Reset list flags to defaults to avoid persistence across tests
	if c, _, errFind := root.Find([]string{"task", "list"}); errFind == nil && c != nil {
		f := c.Flags()
		_ = f.Set("status", "")
		_ = f.Set("priority", "")
		_ = f.Set("tags", "")
		_ = f.Set("contains", "")
		_ = f.Set("contains-fields", "title")
	}
	_ = root.Execute()
	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	buf.Write(data)
	return buf.String()
}

// helper to extract the order of task titles from list output lines
var listLine = regexp.MustCompile(`\[[ x]\] \((.*?)\) (.*)$`)

func extractOrder(t *testing.T, out string) []string {
	var order []string
	for _, line := range bytes.Split([]byte(out), []byte("\n")) {
		m := listLine.FindSubmatch(line)
		if len(m) == 3 {
			order = append(order, string(m[2]))
		}
	}
	return order
}

func TestListSortingByPriorityAndStatus(t *testing.T) {
	tasks := []models.Task{
		{ID: "1", Title: "Low", Priority: "low", Status: "todo"},
		{ID: "2", Title: "High", Priority: "high", Status: "todo"},
		{ID: "3", Title: "Medium", Priority: "medium", Status: "done"},
		{ID: "4", Title: "Highest", Priority: "highest", Status: "in-progress"},
	}
	out := seedAndExec(t, tasks, "task", "list", "--sort-by", "priority")
	order := extractOrder(t, out)
	if len(order) != 4 || order[0] != "Highest" || order[1] != "High" || order[2] != "Medium" || order[3] != "Low" {
		t.Fatalf("priority sort wrong order: %v output=%s", order, out)
	}
	out = seedAndExec(t, tasks, "task", "list", "--sort-by", "status")
	order = extractOrder(t, out)
	// done should appear after todo because we sort ascending on status string; 'done' > 'todo'
	if len(order) != 4 || order[0] != "Low" || order[1] != "High" || order[2] != "Highest" || order[3] != "Medium" {
		t.Fatalf("status sort wrong order: %v output=%s", order, out)
	}
}
