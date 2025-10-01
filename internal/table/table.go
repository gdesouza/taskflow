
package table

import (
	"os"
	"strconv"
	"taskflow/internal/models"

	"github.com/olekukonko/tablewriter"
)

// RenderTasks renders a slice of tasks as a table.
func RenderTasks(tasks []models.Task, compact bool) {
	table := tablewriter.NewTable(os.Stdout)

	if compact {
		table.Header([]string{"Status", "Title"})
		for _, t := range tasks {
			status := " "
			if t.Completed {
				status = "x"
			}
			table.Append([]string{status, t.Title})
		}
	} else {
		table.Header([]string{"Status", "Title", "Description", "Due Date", "Priority"})
		for _, t := range tasks {
			status := " "
			if t.Completed {
				status = "x"
			}
			table.Append([]string{
				status,
				t.Title,
				t.Description,
				t.DueDate,
				strconv.Itoa(t.PriorityInt),
			})
		}
	}

	table.Render()
}
