package table

import (
	"fmt"
	"taskflow/internal/models"
)

// RenderTasks renders a slice of tasks as a table.
func RenderTasks(tasks []models.Task, compact bool) {
	if compact {
		for _, t := range tasks {
			status := " "
			if t.Completed {
				status = "x"
			}
			fmt.Printf("[%s] %s\n", status, t.Title)
		}
	} else {
		for _, t := range tasks {
			status := " "
			if t.Completed {
				status = "x"
			}
			fmt.Printf("[%s] %s - %s - %s - %d\n", status, t.Title, t.Description, t.DueDate, t.PriorityInt)
		}
	}
}