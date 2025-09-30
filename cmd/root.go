package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
	"taskflow/cmd/task"
)

var rootCmd = &cobra.Command{
	Use:   "taskflow",
	Short: "Unified Task and Calendar Management Suite",
	Long:  `TaskFlow is an integrated CLI for tasks, calendars, and visualization.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Add subcommands here in later phases
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Task management commands",
	}
	taskCmd.AddCommand(task.InteractiveCmd)
	rootCmd.AddCommand(taskCmd)
}
