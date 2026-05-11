package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/ref-cli/ref-cli/internal/config"
	"github.com/ref-cli/ref-cli/internal/examples"
)

var openCmd = &cobra.Command{
	Use:   "open [command]",
	Short: "Open the examples directory or a specific example file",
	Long: `Open the examples directory in Finder/xdg-open, or open a specific
example file in $EDITOR.

  ref open         # open ~/.config/ref/examples/ in Finder (macOS) or xdg-open (Linux)
  ref open tar     # open tar.txt in $EDITOR`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := examples.Dir(config.ExamplesDir())

		if len(args) == 0 {
			return openPath(dir)
		}

		name := args[0]
		path := filepath.Join(dir, name+".txt")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("no example found for %q", name)
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			return openPath(path)
		}
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

func openPath(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
