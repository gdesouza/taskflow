
package gcal

import (
	"encoding/csv"
	"fmt"
	"io"
	"taskflow/internal/models"
	"time"
)

// ParseGcalcliTSV parses the TSV output of gcalcli and returns a slice of CalendarEvent
func ParseGcalcliTSV(reader io.Reader) ([]models.CalendarEvent, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = '\t'

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading header: %v", err)
	}

	// Find column indices
	var startDateIdx, startTimeIdx, endDateIdx, endTimeIdx, titleIdx, locationIdx int = -1, -1, -1, -1, -1, -1
	for i, col := range header {
		switch col {
		case "start_date":
			startDateIdx = i
		case "start_time":
			startTimeIdx = i
		case "end_date":
			endDateIdx = i
		case "end_time":
			endTimeIdx = i
		case "title":
			titleIdx = i
		case "location":
			locationIdx = i
		}
	}

	if startDateIdx == -1 || startTimeIdx == -1 || titleIdx == -1 {
		return nil, fmt.Errorf("missing required columns in TSV")
	}

	var events []models.CalendarEvent
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading record: %v", err)
		}

		startTimeStr := record[startDateIdx] + " " + record[startTimeIdx]
		startTime, err := time.Parse("2006-01-02 15:04", startTimeStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing start time: %v", err)
		}

		event := models.CalendarEvent{
			ID:        fmt.Sprintf("gcal-%d", len(events)+1),
			Title:     record[titleIdx],
			StartTime: startTime.Format(time.RFC3339),
		}

		if endDateIdx != -1 && endTimeIdx != -1 {
			endTimeStr := record[endDateIdx] + " " + record[endTimeIdx]
			endTime, err := time.Parse("2006-01-02 15:04", endTimeStr)
			if err == nil {
				event.EndTime = endTime.Format(time.RFC3339)
			}
		}

		if locationIdx != -1 {
			event.Location = record[locationIdx]
		}

		events = append(events, event)
	}

	return events, nil
}
