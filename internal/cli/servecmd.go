package cli

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"

	"marmara-cli/internal/web"

	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var host string
	var port string
	var open bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the local web dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := web.New()
			if err != nil {
				return err
			}
			url := fmt.Sprintf("http://%s:%s", displayHost(host), port)
			if listenAll(host) {
				fmt.Fprintln(os.Stderr, "serving on all interfaces (Ctrl+C to stop)")
				for _, ip := range lanIPs() {
					fmt.Fprintln(os.Stderr, "  LAN: http://"+ip+":"+port)
				}
				fmt.Fprintln(os.Stderr, "  local: http://127.0.0.1:"+port)
			} else {
				fmt.Fprintln(os.Stderr, "serving on", url, "(Ctrl+C to stop)")
			}
			if open {
				go openBrowser(url)
			}
			return s.ListenAndServe(host, port)
		},
	}
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "host/IP to listen on (use 0.0.0.0 for LAN access)")
	cmd.Flags().StringVar(&port, "port", "8080", "port to listen on")
	cmd.Flags().BoolVar(&open, "open", false, "open the dashboard in your browser")
	return cmd
}

func listenAll(host string) bool {
	return host == "" || host == "0.0.0.0" || host == "::" || host == "[::]"
}

func displayHost(host string) string {
	if listenAll(host) {
		return "127.0.0.1"
	}
	return host
}

func lanIPs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	var ips []string
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if !ok || n.IP.IsLoopback() || n.IP.To4() == nil {
			continue
		}
		ips = append(ips, n.IP.String())
	}
	return ips
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
