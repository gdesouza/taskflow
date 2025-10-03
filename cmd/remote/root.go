package remote

import "github.com/spf13/cobra"

// RemoteCmd groups remote sync related subcommands.
var RemoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Remote synchronization commands",
}
