package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTasks_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	if err := os.WriteFile(path, []byte("not: [valid"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	st, _ := NewStorage(path)
	_, err := st.ReadTasks()
	if err == nil {
		t.Fatalf("expected unmarshal error")
	}
}

func TestBackup_NoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	st, _ := NewStorage(path)
	if err := st.Backup(); err != nil {
		t.Fatalf("expected nil for missing file, got %v", err)
	}
}

func TestReadTasks_UnknownPriority(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	content := "tasks:\n  - id: 1\n    title: Test\n    status: todo\n    priority: mystery\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	st, _ := NewStorage(path)
	read, err := st.ReadTasks()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read[0].PriorityInt != 0 {
		t.Fatalf("expected PriorityInt 0 for unknown priority, got %d", read[0].PriorityInt)
	}
}
