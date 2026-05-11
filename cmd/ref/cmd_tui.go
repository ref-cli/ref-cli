package main

import (
	"fmt"
	"os"

	"github.com/ref-cli/ref-cli/internal/config"
	"github.com/ref-cli/ref-cli/internal/examples"
	"github.com/ref-cli/ref-cli/internal/tui"
)

// runTUI loads examples and starts the interactive TUI.
func runTUI() error {
	dir := examples.Dir(config.ExamplesDir())
	exs, err := examples.LoadAll(dir)
	if err != nil {
		return err
	}
	if len(exs) == 0 {
		fmt.Fprintln(os.Stderr, "No examples found. Run 'ref init' to initialize.")
		return nil
	}
	return tui.Run(exs, dir)
}
