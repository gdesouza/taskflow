
package plugin

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Plugin is the interface that all plugins must implement.
type Plugin interface {
	Name() string
	Description() string
	Init(rootCmd *cobra.Command)
}

var ( // Global plugin registry
	plugins = make(map[string]Plugin)
)

// RegisterPlugin registers a new plugin with the application.
func RegisterPlugin(p Plugin) error {
	if _, exists := plugins[p.Name()]; exists {
		return fmt.Errorf("plugin with name '%s' already registered", p.Name())
	}
	plugins[p.Name()] = p
	return nil
}

// GetPlugin returns a registered plugin by its name.
func GetPlugin(name string) (Plugin, bool) {
	p, ok := plugins[name]
	return p, ok
}

// GetAllPlugins returns all registered plugins.
func GetAllPlugins() []Plugin {
	var allPlugins []Plugin
	for _, p := range plugins {
		allPlugins = append(allPlugins, p)
	}
	return allPlugins
}
