package cmd

import (
	"fmt"
	"os"
	"taskflow/cmd/calendar"
	"taskflow/cmd/display"
	"taskflow/cmd/task"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "taskflow",
	Short: "Unified Task and Calendar Management Suite",
	Long:  `TaskFlow is an integrated CLI for tasks, calendars, and visualization.`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "taskflow",
		Short: "Unified Task and Calendar Management Suite",
		Long:  `TaskFlow is an integrated CLI for tasks, calendars, and visualization.`,
	}
	addSubcommands(root)
	return root
}

func addSubcommands(root *cobra.Command) {
	// Add subcommands here in later phases
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Task management commands",
	}
	taskCmd.AddCommand(task.InteractiveCmd)
	taskCmd.AddCommand(task.AddCmd)
	taskCmd.AddCommand(task.DoneCmd)
	taskCmd.AddCommand(task.EditCmd)
	taskCmd.AddCommand(task.ListCmd)
	taskCmd.AddCommand(task.SearchCmd)
	taskCmd.AddCommand(task.StatsCmd)
	taskCmd.AddCommand(task.UndoCmd)
	taskCmd.AddCommand(task.ConfigCmd)
	taskCmd.AddCommand(task.CompletionCmd)
	taskCmd.AddCommand(task.PrioritizeCmd)
	taskCmd.AddCommand(task.ScheduleCmd)
	root.AddCommand(taskCmd)
	root.AddCommand(calendar.CalendarCmd)
	root.AddCommand(display.DisplayCmd)
}

func init() {
	addSubcommands(RootCmd)
}
