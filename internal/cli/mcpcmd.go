package cli

import (
	"marmara-cli/internal/mcpserver"

	"github.com/spf13/cobra"
)

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run as an MCP stdio server for AI agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcpserver.Serve()
		},
	}
}
