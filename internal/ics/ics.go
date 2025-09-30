package ics

import (
	"bytes"
	"fmt"
	"io"
	"taskflow/internal/models"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/google/uuid"
)

// ParseICS parses the ICS file and returns a slice of Task
func ParseICS(reader io.Reader, daysAhead int) ([]models.Task, error) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	data := buf.Bytes()

	cal, err := ical.ParseCalendar(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing ICS: %v", err)
	}

	now := time.Now()
	future := now.AddDate(0, 0, daysAhead)
	var tasks []models.Task

	for _, event := range cal.Events() {
		summary := event.GetProperty(ical.ComponentPropertySummary).Value
		dtstart, err := event.GetStartAt()
		if err != nil {
			continue
		}
		if dtstart.After(now) && dtstart.Before(future) {
			task := models.Task{
				ID:       uuid.New().String(),
				Title:    summary,
				DueDate:  dtstart.Format(time.RFC3339),
				Priority: 3, // High
			}
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}