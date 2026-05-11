package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ref-cli/ref-cli/internal/bundled"
	"github.com/ref-cli/ref-cli/internal/config"
	"github.com/ref-cli/ref-cli/internal/update"
)

// version is set at build time via -ldflags.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "ref [command]",
	Short: "Quick command examples in your terminal",
	Long: `ref — quick command examples in your terminal

  ref              open interactive TUI
  ref <command>    show examples for a command
  ref -s <phrase>  search examples by use case
  ref -l           list all available commands`,
	Args:              cobra.MaximumNArgs(1),
	RunE:              runRoot,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("ref {{.Version}}\n")

	rootCmd.Flags().BoolP("list", "l", false, "list all commands with examples")
	rootCmd.Flags().StringP("search", "s", "", "search examples by use case (non-interactive)")
	rootCmd.Flags().Bool("ai", false, "force AI generation even if example exists")
	rootCmd.Flags().String("ai-backend", "", "override AI backend for this invocation (claude|codex)")
	rootCmd.Flags().Bool("save", false, "save AI-generated example to ~/.config/ref/examples/")

	rootCmd.PersistentPreRunE = persistentPreRun

	rootCmd.AddCommand(initCmd, examplesCmd, contributeCmd, pathCmd, openCmd)
}

func persistentPreRun(cmd *cobra.Command, args []string) error {
	// Auto-init: skip for 'ref init' itself.
	if cmd.Name() != "init" {
		if err := maybeInit(); err != nil {
			return err
		}
	}
	return runUpdateCheck(cmd)
}

// maybeInit runs ref init if conf.yml doesn't exist yet.
func maybeInit() error {
	if config.IsInitialized() {
		return nil
	}
	fmt.Fprintf(os.Stderr, "Initializing ref with bundled examples (%s)…\n", bundled.ExamplesVersion)
	return doInit(false)
}

// runUpdateCheck shows any pending update notifications, then schedules a
// background check if one is due. No network call blocks the hot path.
func runUpdateCheck(cmd *cobra.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}

	if err := showPendingUpdates(cfg, cmd); err != nil {
		return err
	}

	if !cfg.IsUpdateCheckDue() {
		return nil
	}

	installedExamplesVersion := cfg.ExamplesVersion
	go func() {
		result := update.Check(version, installedExamplesVersion)
		if result == nil {
			// Both API calls failed — leave LastUpdateCheck untouched so we retry next invocation.
			return
		}
		pending, err := config.Load()
		if err != nil {
			return
		}
		pending.LastUpdateCheck = time.Now().UTC().Format(time.RFC3339)
		if result.BinaryUpdate != "" {
			pending.PendingBinaryUpdate = result.BinaryUpdate
		}
		if result.ExamplesUpdate != "" {
			pending.PendingExamplesUpdate = result.ExamplesUpdate
		}
		_ = pending.Save()
	}()

	return nil
}

// showPendingUpdates displays any update notifications stored from the
// previous background check and clears them from config.
func showPendingUpdates(cfg *config.Config, cmd *cobra.Command) error {
	binaryVer := cfg.PendingBinaryUpdate
	exVer := cfg.PendingExamplesUpdate

	if binaryVer != "" || exVer != "" {
		cfg.PendingBinaryUpdate = ""
		cfg.PendingExamplesUpdate = ""
		_ = cfg.Save()
	}

	if binaryVer != "" {
		fmt.Fprintf(os.Stderr, "\nA new version of ref is available: %s (you have %s)\n", binaryVer, version)
		if cfg.InstallMethod == "homebrew" {
			fmt.Fprintf(os.Stderr, "  brew upgrade ref\n\n")
		} else {
			fmt.Fprintf(os.Stderr, "  https://github.com/ref-cli/ref-cli/releases/latest\n\n")
		}
	}

	if exVer != "" {
		// Only prompt interactively on bare `ref` (TUI mode); otherwise hint.
		if cmd.Name() == "ref" && len(os.Args) == 1 {
			fmt.Fprintf(os.Stderr, "New examples available: %s (you have %s)\nSync now? [y/N]: ", exVer, cfg.ExamplesVersion)
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			if strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" {
				return runExamplesSync(exVer)
			}
		} else {
			fmt.Fprintf(os.Stderr, "New examples available: %s — run 'ref examples sync' to update.\n", exVer)
		}
	}

	return nil
}

func runRoot(cmd *cobra.Command, args []string) error {
	if list, _ := cmd.Flags().GetBool("list"); list {
		return runList()
	}
	if phrase, _ := cmd.Flags().GetString("search"); phrase != "" {
		return runSearchPhrase(phrase)
	}
	if len(args) == 1 {
		return runView(cmd, args[0])
	}
	return runTUI()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ref:", err)
		os.Exit(1)
	}
}
