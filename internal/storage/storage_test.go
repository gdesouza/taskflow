package storage

import (
	"os"
	"path/filepath"
	"taskflow/internal/models"
	"testing"
)

func TestReadTasks_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	st, _ := NewStorage(path)
	tasks, err := st.ReadTasks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected empty slice, got %#v", tasks)
	}
}

func TestWriteAndReadTasks_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	st, _ := NewStorage(path)
	in := []models.Task{
		{ID: "2", Title: "Second", Status: "todo", Priority: "medium"},
		{ID: "1", Title: "First", Status: "done", Priority: "high"},
	}
	if err := st.WriteTasks(in); err != nil {
		t.Fatalf("write error: %v", err)
	}
	out, err := st.ReadTasks()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	// Should be sorted by ID ascending: 1,2
	if len(out) != 2 || out[0].ID != "1" || out[1].ID != "2" {
		t.Fatalf("unexpected ordering: %#v", out)
	}
	// Completed and PriorityInt computed
	if !out[0].Completed || out[0].PriorityInt != 3 { // high => 3
		t.Fatalf("expected task 1 computed fields, got %#v", out[0])
	}
}

func TestUpdateTask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	st, _ := NewStorage(path)
	orig := []models.Task{{ID: "1", Title: "A", Status: "todo", Priority: "low"}}
	if err := st.WriteTasks(orig); err != nil {
		t.Fatalf("write: %v", err)
	}
	orig[0].Title = "Updated"
	orig[0].Status = "done"
	if err := st.UpdateTask(orig, orig[0]); err != nil {
		t.Fatalf("update: %v", err)
	}
	read, err := st.ReadTasks()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read[0].Title != "Updated" || read[0].Status != "done" || !read[0].Completed {
		t.Fatalf("task not updated correctly: %#v", read[0])
	}
}

func TestBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	st, _ := NewStorage(path)
	if err := os.WriteFile(path, []byte("tasks: []"), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := st.Backup(); err != nil {
		t.Fatalf("backup: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
}

func TestDescriptionFallbackToLink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	st, _ := NewStorage(path)
	content := "tasks:\n  - id: 1\n    title: Test\n    status: todo\n    priority: low\n    link: https://example.com\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	read, err := st.ReadTasks()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(read) != 1 || read[0].Description != "https://example.com" {
		t.Fatalf("expected description fallback to link, got %#v", read[0])
	}
}
