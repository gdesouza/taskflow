package calendar

import (
	"fmt"
	"taskflow/internal/config"
	"taskflow/internal/storage"
	"time"

	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List calendar events",
	Run: func(cmd *cobra.Command, args []string) {
		storagePath := config.GetCalendarStoragePath()
		s, err := storage.NewStorage(storagePath)
		if err != nil {
			fmt.Printf("Error creating storage: %v\n", err)
			return
		}

		events, err := s.ReadCalendarEvents()
		if err != nil {
			fmt.Printf("Error reading calendar events: %v\n", err)
			return
		}

		if len(events) == 0 {
			fmt.Println("No calendar events found.")
			return
		}

		for _, event := range events {
			startTime, _ := time.Parse(time.RFC3339, event.StartTime)
			fmt.Printf("%s: %s\n", startTime.Format("2006-01-02 15:04"), event.Title)
		}
	},
}

func init() {
	CalendarCmd.AddCommand(ListCmd)
}