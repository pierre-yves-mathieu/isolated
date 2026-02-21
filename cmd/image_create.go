package cmd

import (
	"fmt"
	"os"

	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

var imageCreateCmd = &cobra.Command{
	Use:   "create <container> <image-name>",
	Short: "Create an image from a container",
	Long: `Create a reusable image from an existing container.

The container will be stopped before creating the image, then restarted.

Example:
  lxc-dev-manager image create dev1 my-base-image

Then create new containers from it:
  lxc-dev-manager container create dev2 my-base-image`,
	Args: cobra.ExactArgs(2),
	RunE: runImageCreate,
}

// imageCreateCmd is registered in image.go init()

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

func stepStart(step, total int, msg string) {
	fmt.Printf("%s[%d/%d]%s %s\n", colorCyan, step, total, colorReset, msg)
}

func stepDone(msg string) {
	fmt.Printf("      %s✓%s %s\n", colorGreen, colorReset, msg)
}

func stepInfo(msg string) {
	fmt.Printf("      %s\n", msg)
}

func runImageCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	imageName := args[1]

	cfg, _, err := requireContainer(name)
	if err != nil {
		return err
	}

	fmt.Printf("Creating image '%s' from container '%s'...\n", imageName, name)

	// Create a prefixed writer to indent LXC output
	stdout := &prefixWriter{prefix: "      ", w: os.Stdout}
	stderr := &prefixWriter{prefix: "      ", w: os.Stderr}

	// Use operations package for core logic
	if err := operations.CreateImage(cfg, name, imageName, stdout, stderr); err != nil {
		return err
	}

	fmt.Printf("\n%sImage '%s' created successfully!%s\n", colorGreen, imageName, colorReset)
	fmt.Printf("\nCreate new containers from it with:\n")
	fmt.Printf("  %s container create <name> %s\n", os.Args[0], imageName)

	return nil
}

// prefixWriter adds a prefix to each line of output
type prefixWriter struct {
	prefix     string
	w          *os.File
	needPrefix bool
}

func (pw *prefixWriter) Write(p []byte) (n int, err error) {
	if pw.needPrefix || len(p) == 0 {
		pw.w.WriteString(pw.prefix)
		pw.needPrefix = false
	}

	for i, b := range p {
		if b == '\n' && i < len(p)-1 {
			pw.w.Write(p[:i+1])
			pw.w.WriteString(pw.prefix)
			p = p[i+1:]
			i = -1
		}
	}

	if len(p) > 0 {
		pw.w.Write(p)
		if p[len(p)-1] == '\n' {
			pw.needPrefix = true
		}
	}

	return len(p), nil
}
