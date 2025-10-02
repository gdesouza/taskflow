package storage

import (
	"path/filepath"
	"taskflow/internal/models"
	"testing"
)

// Ensures UpdateTask still writes file even if ID not found
func TestUpdateTaskIDNotFound(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tasks.yaml")
	st, _ := NewStorage(file)

	orig := []models.Task{{ID: "a", Title: "A"}}
	if err := st.WriteTasks(orig); err != nil {
		t.Fatalf("write initial: %v", err)
	}

	err := st.UpdateTask(orig, models.Task{ID: "missing", Title: "Missing"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	after, err := st.ReadTasks()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(after) != 1 || after[0].ID != "a" {
		t.Fatalf("unexpected tasks after update: %+v", after)
	}
}
