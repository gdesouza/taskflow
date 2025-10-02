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
)

func TestUndoCommand(t *testing.T) {
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	cfgDir := filepath.Join(tempHome, ".config", config.AppName)
	_ = os.MkdirAll(cfgDir, 0755)
	viper.Reset()
	if err := config.Init(); err != nil {
		t.Fatalf("config init: %v", err)
	}
	tasksPath := filepath.Join(cfgDir, "tasks.yaml")
	orig := "tasks:\n- id: 1\n  title: original\n  status: todo\n  priority: low\n"
	modified := "tasks:\n- id: 1\n  title: changed\n  status: todo\n  priority: low\n"
	if err := os.WriteFile(tasksPath, []byte(modified), 0644); err != nil {
		t.Fatalf("write modified: %v", err)
	}
	if err := os.WriteFile(tasksPath+".bak", []byte(orig), 0644); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	// run undo
	buf := new(bytes.Buffer)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"task", "undo"})
	_ = root.Execute()
	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	buf.Write(data)
	out := buf.String()
	if !strings.Contains(out, "Last operation undone") {
		t.Fatalf("expected undo message: %s", out)
	}
	restored, _ := os.ReadFile(tasksPath)
	if !strings.Contains(string(restored), "original") {
		t.Fatalf("did not restore original: %s", string(restored))
	}
}
