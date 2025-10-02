package storage

import (
	"os"
	"path/filepath"
	"regexp"
	"taskflow/internal/models"
	"testing"
)

// TestTasksYAMLFormat ensures WriteTasks produces the expected YAML schema so external integrations remain stable.
func TestTasksYAMLFormat(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tasks.yaml")
	st, _ := NewStorage(file)

	tasks := []models.Task{
		{ID: "2", Title: "Second", Status: "todo", Priority: "medium", Tags: []string{"dev"}, Notes: "note here", Link: "http://ex/2"},
		{ID: "1", Title: "First", Status: "done", Priority: "high", Description: "desc", DueDate: "2025-10-05", Link: "http://ex/1", Tags: []string{"ops", "urgent"}},
	}
	if err := st.WriteTasks(tasks); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)

	// Basic structure: top-level 'tasks:' sequence
	if !regexp.MustCompile(`(?m)^tasks:`).MatchString(content) {
		t.Fatalf("missing top-level tasks key: %s", content)
	}
	// IDs should be sorted (1 before 2) because WriteTasks sorts by ID
	firstIdx := regexp.MustCompile(`id: "1"`).FindStringIndex(content)
	secondIdx := regexp.MustCompile(`id: "2"`).FindStringIndex(content)
	if firstIdx == nil || secondIdx == nil || firstIdx[0] > secondIdx[0] {
		t.Fatalf("expected id 1 before id 2; content=%s", content)
	}
	// Ensure omitted internal fields are not serialized
	if regexp.MustCompile(`Completed:`).MatchString(content) {
		t.Fatalf("internal field Completed serialized: %s", content)
	}
	if regexp.MustCompile(`PriorityInt:`).MatchString(content) {
		t.Fatalf("internal field PriorityInt serialized: %s", content)
	}
	// Ensure optional empty fields are omitted: we did not set Description for task 2
	block2 := content[secondIdx[0]:]
	if regexp.MustCompile(`description:`).MatchString(block2) {
		t.Fatalf("unexpected empty description field present in second task block: %s", block2)
	}
	// Ensure notes present for task 2 (value may be unquoted)
	if !regexp.MustCompile(`(?m)notes: note here`).MatchString(block2) {
		t.Fatalf("expected notes field in second task block: %s", block2)
	}

	// Ensure due date present for task 1
	if !regexp.MustCompile(`due: "2025-10-05"`).MatchString(content) {
		t.Fatalf("expected due field for task 1: %s", content)
	}
	// Ensure tags list formatting (simple dash list or inline) appears; accept YAML sequence style '-'
	if !regexp.MustCompile(`(?m)\s*- ops`).MatchString(content) {
		t.Fatalf("expected tag 'ops' as list item: %s", content)
	}
}
