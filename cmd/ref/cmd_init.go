package main

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ref-cli/ref-cli/internal/bundled"
	"github.com/ref-cli/ref-cli/internal/config"
	"github.com/ref-cli/ref-cli/internal/update"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Extract bundled examples to ~/.config/ref/examples/",
	Long:  "Extract the bundled example files to ~/.config/ref/examples/ (no network call). Safe to re-run — existing files are not overwritten.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return doInit(true)
	},
}

// doInit extracts bundled examples to ~/.config/ref/examples/.
// If verbose is true, prints a summary line.
func doInit(verbose bool) error {
	exDir := config.ExamplesDir()
	if err := os.MkdirAll(exDir, 0o755); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	checksums := map[string]string{}
	written := 0

	err := fs.WalkDir(bundled.FS, "examples", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		basename := filepath.Base(path)
		if !strings.HasSuffix(basename, ".txt") {
			return nil
		}

		dest := filepath.Join(exDir, basename)
		data, err := bundled.FS.ReadFile(path)
		if err != nil {
			return err
		}

		// No-overwrite: skip files that already exist.
		f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			if os.IsExist(err) {
				return nil // already present, leave it alone
			}
			return err
		}
		_, writeErr := f.Write(data)
		if err := f.Close(); writeErr == nil {
			writeErr = err
		}
		if writeErr != nil {
			return writeErr
		}

		sum := fmt.Sprintf("%x", sha256.Sum256(data))
		checksums[basename] = sum
		written++
		return nil
	})
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}

	// Persist checksums so sync can detect user edits later.
	if err := update.SaveChecksums(config.ChecksumsFile(), checksums); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	cfg, _ := config.Load()
	cfg.ExamplesVersion = bundled.ExamplesVersion
	if cfg.InstallMethod == "" {
		cfg.InstallMethod = config.DetectInstallMethod()
	}
	if cfg.LastUpdateCheck == "" {
		cfg.LastUpdateCheck = time.Now().UTC().Format(time.RFC3339)
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Initialized: %d examples written to %s\n", written, exDir)
	}
	return nil
}
