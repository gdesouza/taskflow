
package hello

import (
	"fmt"
	"taskflow/internal/plugin"

	"github.com/spf13/cobra"
)

type HelloPlugin struct{}

func (p *HelloPlugin) Name() string {
	return "hello"
}

func (p *HelloPlugin) Description() string {
	return "A simple hello world plugin."
}

func (p *HelloPlugin) Init(rootCmd *cobra.Command) {
	helloCmd := &cobra.Command{
		Use:   "hello",
		Short: "Says hello",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello from the plugin!")
		},
	}
	rootCmd.AddCommand(helloCmd)
}

func init() {
	plugin.RegisterPlugin(&HelloPlugin{})
}
