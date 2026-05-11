package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ref-cli/ref-cli/internal/bundled"
	"github.com/ref-cli/ref-cli/internal/config"
	"github.com/ref-cli/ref-cli/internal/update"
)

var examplesCmd = &cobra.Command{
	Use:   "examples",
	Short: "Show examples version info or sync to a newer version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		fmt.Printf("Bundled:   %s\n", bundled.ExamplesVersion)
		installed := cfg.ExamplesVersion
		if installed == "" {
			installed = "(not initialized)"
		}
		fmt.Printf("Installed: %s\n", installed)
		return nil
	},
}

var examplesSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Download latest ref-examples from GitHub, skipping locally modified files",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ver, _ := cmd.Flags().GetString("version")
		return runExamplesSync(ver)
	},
}

func init() {
	examplesSyncCmd.Flags().String("version", "", "download a specific ref-examples version (e.g. v0.1.0)")
	examplesCmd.AddCommand(examplesSyncCmd)
}

func runExamplesSync(requestedVersion string) error {
	cfg, _ := config.Load()
	from := cfg.ExamplesVersion

	fmt.Fprintln(os.Stderr, "Syncing examples…")
	result, err := update.Sync(config.ExamplesDir(), config.ChecksumsFile(), requestedVersion)
	if err != nil {
		return fmt.Errorf("ref: %w\nYour examples are unchanged. Try again later or check your connection", err)
	}

	cfg.ExamplesVersion = result.Version
	_ = cfg.Save()

	updated := len(result.Updated)
	skipped := len(result.Skipped)

	if updated == 0 && skipped == 0 {
		fmt.Printf("Already up to date (%s)\n", result.Version)
		return nil
	}

	msg := fmt.Sprintf("Updated %d examples", updated)
	if from != "" && from != result.Version {
		msg += fmt.Sprintf(" (%s → %s)", from, result.Version)
	}
	if len(result.Updated) > 0 {
		msg += fmt.Sprintf(": %v", result.Updated)
	}
	fmt.Println(msg)

	if skipped > 0 {
		fmt.Printf("Skipped %d (locally modified): %v\n", skipped, result.Skipped)
	}
	return nil
}
