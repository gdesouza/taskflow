package tasks

import (
	"reflect"
	"taskflow/internal/models"
	"testing"
)

func makeTask(id, title, status, priority string, tags []string, desc, notes, link string) models.Task {
	return models.Task{ID: id, Title: title, Status: status, Priority: priority, Tags: tags, Description: desc, Notes: notes, Link: link}
}

func TestApplyFilters_StatusPriorityTags(t *testing.T) {
	all := []models.Task{
		makeTask("1", "Task One", "todo", "high", []string{"work", "urgent"}, "", "", ""),
		makeTask("2", "Task Two", "in-progress", "medium", []string{"home"}, "", "", ""),
		makeTask("3", "Task Three", "done", "low", []string{"work"}, "", "", ""),
	}

	res := ApplyFilters(all, FilterOptions{Status: "todo"})
	if len(res) != 1 || res[0].ID != "1" {
		t.Fatalf("expected only task 1, got %#v", res)
	}

	res = ApplyFilters(all, FilterOptions{Priority: "medium"})
	if len(res) != 1 || res[0].ID != "2" {
		t.Fatalf("expected only task 2 by priority, got %#v", res)
	}

	res = ApplyFilters(all, FilterOptions{Tags: []string{"urgent"}})
	if len(res) != 1 || res[0].ID != "1" {
		t.Fatalf("expected only task 1 by tag urgent, got %#v", res)
	}

	res = ApplyFilters(all, FilterOptions{Tags: []string{"work"}})
	if len(res) != 2 {
		t.Fatalf("expected two tasks with tag work, got %#v", res)
	}
}

func TestApplyFilters_ContainsWords_DefaultField(t *testing.T) {
	all := []models.Task{
		makeTask("1", "Implement OAuth login", "todo", "high", nil, "", "", ""),
		makeTask("2", "Write docs", "todo", "low", nil, "", "", ""),
	}

	// words ANDed: both implement and login in title
	res := ApplyFilters(all, FilterOptions{ContainsWords: []string{"implement", "login"}})
	if len(res) != 1 || res[0].ID != "1" {
		t.Fatalf("expected only task 1 for implement login, got %#v", res)
	}
}

func TestApplyFilters_ContainsWords_MultiFields(t *testing.T) {
	all := []models.Task{
		makeTask("1", "Refactor module", "todo", "high", []string{"backend"}, "Improve performance", "Investigate slow queries", ""),
		makeTask("2", "Add search", "todo", "medium", []string{"feature"}, "Implement basic text search", "Add filters", ""),
		makeTask("3", "Research DB", "todo", "low", []string{"investigation"}, "", "Compare postgres and mysql", "https://db.example"),
	}

	// Search across description and notes for words 'slow' and 'queries'
	res := ApplyFilters(all, FilterOptions{ContainsWords: []string{"slow", "queries"}, ContainsFields: map[string]bool{"notes": true}})
	if len(res) != 1 || res[0].ID != "1" {
		t.Fatalf("expected only task 1 for slow queries in notes, got %#v", res)
	}

	// Search across multiple fields (description + notes) requiring both words anywhere
	res = ApplyFilters(all, FilterOptions{ContainsWords: []string{"basic", "filters"}, ContainsFields: map[string]bool{"description": true, "notes": true}})
	if len(res) != 1 || res[0].ID != "2" {
		t.Fatalf("expected only task 2 for basic filters, got %#v", res)
	}

	// Tag search via contains fields: searching tag value
	res = ApplyFilters(all, FilterOptions{ContainsWords: []string{"backend"}, ContainsFields: map[string]bool{"tags": true}})
	if len(res) != 1 || res[0].ID != "1" {
		t.Fatalf("expected only task 1 for tag backend contains, got %#v", res)
	}

	// Link field search
	res = ApplyFilters(all, FilterOptions{ContainsWords: []string{"db.example"}, ContainsFields: map[string]bool{"link": true}})
	if len(res) != 1 || res[0].ID != "3" {
		t.Fatalf("expected only task 3 for link contains db.example, got %#v", res)
	}
}

func TestApplyFilters_AllFiltersCombined(t *testing.T) {
	all := []models.Task{
		makeTask("1", "Implement login", "todo", "high", []string{"auth", "backend"}, "Add OAuth", "Use PKCE", ""),
		makeTask("2", "Implement logout", "todo", "high", []string{"auth"}, "Add button", "Clear session", ""),
		makeTask("3", "Write docs", "in-progress", "medium", []string{"docs"}, "Update README", "", ""),
	}

	opts := FilterOptions{
		Status:         "todo",
		Priority:       "high",
		Tags:           []string{"backend"}, // ANY logic means task 1 qualifies
		ContainsWords:  []string{"oauth", "pkce"},
		ContainsFields: map[string]bool{"description": true, "notes": true},
	}

	res := ApplyFilters(all, opts)
	if len(res) != 1 || res[0].ID != "1" {
		t.Fatalf("expected only task 1 for combined filters, got %#v", res)
	}
}

func TestApplyFilters_NoMatches(t *testing.T) {
	all := []models.Task{makeTask("1", "Test", "todo", "low", nil, "", "", "")}
	res := ApplyFilters(all, FilterOptions{Status: "done"})
	if len(res) != 0 {
		t.Fatalf("expected no results, got %#v", res)
	}
}

func TestApplyFilters_EmptyOptionsReturnsAll(t *testing.T) {
	all := []models.Task{makeTask("1", "Test", "todo", "low", nil, "", "", "")}
	res := ApplyFilters(all, FilterOptions{})
	if !reflect.DeepEqual(res, all) {
		t.Fatalf("expected all tasks back, got %#v", res)
	}
}
