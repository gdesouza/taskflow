
package main

import (
	"fmt"
	"os"
	"taskflow/cmd"
	"taskflow/internal/config"
	"taskflow/internal/plugin"
	_ "taskflow/internal/plugins/hello" // Import to register the plugin
)

func main() {
	if err := config.Init(); err != nil {
		fmt.Printf("Error initializing config: %v\n", err)
		os.Exit(1)
	}

	// Initialize plugins
	for _, p := range plugin.GetAllPlugins() {
		p.Init(cmd.RootCmd)
	}

	cmd.Execute()
}

