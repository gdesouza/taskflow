package tasks

import (
	"strings"
	"taskflow/internal/models"
)

type FilterOptions struct {
	Status         string
	Priority       string
	Tags           []string
	ContainsWords  []string // lowercased words
	ContainsFields map[string]bool
}

// ApplyFilters filters the provided tasks slice according to the options.
// All filters are ANDed together. Tag filter matches if ANY provided tag is present.
// ContainsWords uses AND logic: every word must appear in at least one of the chosen fields.
func ApplyFilters(all []models.Task, opts FilterOptions) []models.Task {

	filtered := make([]models.Task, 0, len(all))

TaskLoop:
	for _, t := range all {
		// Status
		if opts.Status != "" && t.Status != opts.Status {
			continue
		}
		// Priority
		if opts.Priority != "" && t.Priority != opts.Priority {
			continue
		}
		// Tags (ANY)
		if len(opts.Tags) > 0 {
			match := false
			for _, need := range opts.Tags {
				for _, have := range t.Tags {
					if need == have {
						match = true
						break
					}
				}
				if match {
					break
				}
			}
			if !match {
				continue
			}
		}
		// Contains words (AND across words, across selected fields)
		if len(opts.ContainsWords) > 0 {
			if len(opts.ContainsFields) == 0 {
				// default to title
				opts.ContainsFields = map[string]bool{"title": true}
			}
			var hayParts []string
			if opts.ContainsFields["title"] {
				hayParts = append(hayParts, t.Title)
			}
			if opts.ContainsFields["description"] {
				hayParts = append(hayParts, t.Description)
			}
			if opts.ContainsFields["notes"] {
				hayParts = append(hayParts, t.Notes)
			}
			if opts.ContainsFields["link"] {
				hayParts = append(hayParts, t.Link)
			}
			if opts.ContainsFields["tags"] {
				hayParts = append(hayParts, strings.Join(t.Tags, " "))
			}
			joined := strings.ToLower(strings.Join(hayParts, " \n "))
			for _, w := range opts.ContainsWords {
				if !strings.Contains(joined, w) {
					continue TaskLoop
				}
			}
		}
		filtered = append(filtered, t)
	}
	return filtered
}
