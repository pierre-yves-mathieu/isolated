package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy <name>",
	Short: "Proxy ports from localhost to container",
	Long: `Start a TCP proxy to forward ports from localhost to the container.

This allows you to access container services as if they were running locally.
All ports defined in the config will be forwarded.

Press Ctrl+C to stop the proxy.

Example:
  lxc-dev-manager proxy dev1

Then access services at:
  http://localhost:5173  ->  container:5173
  http://localhost:8000  ->  container:8000`,
	Args: cobra.ExactArgs(1),
	RunE: runProxy,
}

func init() {
	rootCmd.AddCommand(proxyCmd)
}

func runProxy(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, _, err := requireRunningContainer(name)
	if err != nil {
		return err
	}

	// Use operations package to start proxy
	manager, ip, ports, err := operations.StartProxy(cfg, name)
	if err != nil {
		return err
	}

	fmt.Printf("Proxying %s (%s):\n", name, ip)
	for _, port := range ports {
		fmt.Printf("  localhost:%d -> %s:%d\n", port, ip, port)
	}

	fmt.Println("\nPress Ctrl+C to stop")

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nStopping proxy...")
	manager.StopAll()

	return nil
}
