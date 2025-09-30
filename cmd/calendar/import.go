

package calendar

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"taskflow/internal/config"
	"taskflow/internal/gcal"
	"taskflow/internal/ics"
	"taskflow/internal/storage"

	"github.com/spf13/cobra"
)

var ImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import calendar events",
}

var GCalCmd = &cobra.Command{
	Use:   "gcal",
	Short: "Import from Google Calendar",
	Run: func(cmd *cobra.Command, args []string) {
		storagePath := config.GetCalendarStoragePath()
		s, err := storage.NewStorage(storagePath)
		if err != nil {
			fmt.Printf("Error creating storage: %v\n", err)
			return
		}

		events, err := gcal.ParseGcalcliTSV(bufio.NewReader(os.Stdin))
		if err != nil {
			fmt.Printf("Error parsing gcalcli TSV: %v\n", err)
			return
		}

		if err := s.WriteCalendarEvents(events); err != nil {
			fmt.Printf("Error writing calendar events: %v\n", err)
			return
		}

		fmt.Printf("Imported %d events from Google Calendar.\n", len(events))
	},
}

var IcsCmd = &cobra.Command{
	Use:   "ics [file] [days_ahead]",
	Short: "Import from ICS file",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		daysAhead := 7
		if len(args) > 1 {
			days, err := strconv.Atoi(args[1])
			if err == nil {
				daysAhead = days
			}
		}

		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			return
		}
		defer file.Close()

		tasks, err := ics.ParseICS(file, daysAhead)
		if err != nil {
			fmt.Printf("Error parsing ICS file: %v\n", err)
			return
		}

		storagePath := config.GetStoragePath()
		s, err := storage.NewStorage(storagePath)
		if err != nil {
			fmt.Printf("Error creating storage: %v\n", err)
			return
		}

		existingTasks, err := s.ReadTasks()
		if err != nil {
			fmt.Printf("Error reading tasks: %v\n", err)
			return
		}

		existingTasks = append(existingTasks, tasks...)

		if err := s.WriteTasks(existingTasks); err != nil {
			fmt.Printf("Error writing tasks: %v\n", err)
			return
		}

		fmt.Printf("Imported %d tasks from ICS file.\n", len(tasks))
	},
}

func init() {
	ImportCmd.AddCommand(GCalCmd)
	ImportCmd.AddCommand(IcsCmd)
	CalendarCmd.AddCommand(ImportCmd)
}
