// Command marmara is an unofficial, personal-use CLI for Marmara Üniversitesi's
// student-facing and public APIs. It also runs as an MCP server (`marmara mcp`)
// so AI agents can call the same operations directly.
//
// Not affiliated with or endorsed by Marmara Üniversitesi. Use with your own
// credentials, at your own risk.
package main

import "marmara-cli/internal/cli"

func main() {
	cli.Execute()
}
