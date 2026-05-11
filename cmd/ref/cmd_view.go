package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/ref-cli/ref-cli/internal/ai"
	"github.com/ref-cli/ref-cli/internal/config"
	"github.com/ref-cli/ref-cli/internal/example"
	"github.com/ref-cli/ref-cli/internal/examples"
)

var (
	commentColor = lipgloss.AdaptiveColor{Light: "#555555", Dark: "#888888"}
	tagColor     = lipgloss.AdaptiveColor{Light: "#CC7700", Dark: "#FFAF5F"}
)

// runView handles `ref <command>` and `ref <command> --ai`.
func runView(cmd *cobra.Command, name string) error {
	cfg, _ := config.Load()
	dir := examples.Dir(config.ExamplesDir())

	forceAI, _ := cmd.Flags().GetBool("ai")
	backendOverride, _ := cmd.Flags().GetString("ai-backend")
	save, _ := cmd.Flags().GetBool("save")

	if !forceAI {
		ex, err := examples.Find(dir, name)
		if err != nil {
			return err
		}
		if ex != nil {
			printExample(ex)
			return nil
		}
	}

	// No example found (or --ai forced): try AI generation.
	return generateWithAI(cfg, name, backendOverride, save)
}

func printExample(ex *example.Example) {
	cs := lipgloss.NewStyle().Foreground(commentColor)
	ts := lipgloss.NewStyle().Foreground(tagColor)

	for i, e := range ex.Entries {
		if i > 0 {
			fmt.Println()
		}
		// Render comment line with optional tags
		line := "# "
		for _, t := range e.Tags {
			line += ts.Render("["+t+"]") + " "
		}
		line += cs.Render(e.Comment)
		fmt.Println(line)
		fmt.Println(e.Command)
	}
}

func generateWithAI(cfg *config.Config, name, backendOverride string, save bool) error {
	backend, err := resolveBackend(cfg, name, backendOverride)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Using %s…\n", backend)
	content, err := ai.Generate(backend, name)
	if err != nil {
		return fmt.Errorf("AI generation failed: %w", err)
	}

	ex := example.Parse(name, content)
	fmt.Printf("\n[AI: %s]\n", backend)
	printExample(ex)

	if save {
		return saveAIExample(name, content)
	}

	fmt.Print("\nSave to examples? [y/N]: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" {
		return saveAIExample(name, content)
	}
	return nil
}

func resolveBackend(cfg *config.Config, name, override string) (ai.Backend, error) {
	if override != "" {
		b := ai.Backend(override)
		if b != ai.Claude && b != ai.Codex {
			return "", fmt.Errorf("unknown AI backend %q: must be 'claude' or 'codex'", override)
		}
		return b, nil
	}
	if cfg.AIBackend != "" {
		return ai.Backend(cfg.AIBackend), nil
	}

	detected := ai.Detect()
	if len(detected) == 0 {
		return "", fmt.Errorf("no example found for %q\nInstall 'claude' (Claude Code) or 'codex' (OpenAI Codex CLI) to enable AI generation", name)
	}
	if len(detected) == 1 {
		// Save preference for next time.
		cfg.AIBackend = string(detected[0])
		_ = cfg.Save()
		return detected[0], nil
	}

	// Multiple backends: prompt user.
	fmt.Print("Multiple AI CLIs detected. Choose backend [claude/codex]: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	choice := strings.ToLower(strings.TrimSpace(scanner.Text()))
	backend := ai.Backend(choice)
	cfg.AIBackend = string(backend)
	_ = cfg.Save()
	return backend, nil
}

func saveAIExample(name, content string) error {
	path := filepath.Join(config.ExamplesDir(), name+".txt")
	if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("Saved to %s\n", path)
	return nil
}

// runList handles `ref -l`.
func runList() error {
	dir := examples.Dir(config.ExamplesDir())
	exs, err := examples.LoadAll(dir)
	if err != nil {
		return err
	}
	if len(exs) == 0 {
		fmt.Println("No examples found. Run 'ref init' to initialize.")
		return nil
	}

	// Print in columns (5 per row).
	const cols = 5
	names := make([]string, len(exs))
	for i, ex := range exs {
		names[i] = ex.Name
	}

	// Find max name width.
	maxW := 0
	for _, n := range names {
		if len(n) > maxW {
			maxW = len(n)
		}
	}
	colW := maxW + 2

	for i, n := range names {
		fmt.Printf("%-*s", colW, n)
		if (i+1)%cols == 0 {
			fmt.Println()
		}
	}
	if len(names)%cols != 0 {
		fmt.Println()
	}
	return nil
}
