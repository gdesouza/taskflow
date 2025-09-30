package task

import (
	"fmt"
	"taskflow/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ConfigCmd = &cobra.Command{
	Use:     "config",
	Short:   "Manage configuration",
	Aliases: []string{"cfg", "configure"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Config file: %s\n", viper.ConfigFileUsed())
		fmt.Printf("Storage path: %s\n", config.GetStoragePath())
	},
}