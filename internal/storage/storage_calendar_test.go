package storage

import (
	"os"
	"path/filepath"
	"taskflow/internal/models"
	"testing"
)

func TestCalendarEventsReadWrite(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "calendar.yaml")
	st, err := NewStorage(file)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}

	// Reading when file does not exist returns empty slice
	events, err := st.ReadCalendarEvents()
	if err != nil {
		t.Fatalf("read empty: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}

	in := []models.CalendarEvent{
		{ID: "2", Title: "Later", StartTime: "2025-10-02T15:00:00Z"},
		{ID: "1", Title: "Earlier", StartTime: "2025-10-02T09:00:00Z"},
	}
	if err := st.WriteCalendarEvents(in); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Ensure file written
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("stat after write: %v", err)
	}

	out, err := st.ReadCalendarEvents()
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 events, got %d", len(out))
	}
	// Should be sorted by StartTime ascending
	if out[0].ID != "1" || out[1].ID != "2" {
		for i, e := range out {
			t.Logf("idx=%d id=%s start=%s", i, e.ID, e.StartTime)
		}
		// Fail with constant format string for vet friendliness
		if out[0].ID != "1" {
			t.Fatalf("events not sorted by StartTime (first=%s second=%s)", out[0].ID, out[1].ID)
		}
	}

}

func TestCalendarEventsUnmarshalError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "bad_events.yaml")
	if err := os.WriteFile(file, []byte("not: a: list"), 0644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}
	st, _ := NewStorage(file)
	_, err := st.ReadCalendarEvents()
	if err == nil {
		t.Fatalf("expected unmarshal error for invalid events YAML")
	}
}
