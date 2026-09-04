package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"marmara-cli/internal/web"

	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var port string
	var open bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the local web dashboard (localhost only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := web.New()
			if err != nil {
				return err
			}
			url := "http://127.0.0.1:" + port
			fmt.Fprintln(os.Stderr, "serving on", url, "(Ctrl+C to stop)")
			if open {
				go openBrowser(url)
			}
			return s.ListenAndServe(port)
		},
	}
	cmd.Flags().StringVar(&port, "port", "8080", "port to listen on")
	cmd.Flags().BoolVar(&open, "open", false, "open the dashboard in your browser")
	return cmd
}

func openBrowser(url string) {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		c = exec.Command("open", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	_ = c.Start()
}
