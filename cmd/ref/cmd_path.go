package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ref-cli/ref-cli/internal/config"
	"github.com/ref-cli/ref-cli/internal/examples"
)

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the path to the examples directory",
	Long:  "Print the path to ~/.config/ref/examples/ (or the active dev override). Designed to be scriptable: cd $(ref path)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(examples.Dir(config.ExamplesDir()))
		return nil
	},
}
