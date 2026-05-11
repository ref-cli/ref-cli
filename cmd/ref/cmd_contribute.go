package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ref-cli/ref-cli/internal/config"
)

var contributeCmd = &cobra.Command{
	Use:   "contribute [command]",
	Short: "Print contribution instructions for an example",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			printGeneralContributeGuide()
			return nil
		}
		return printCommandContribute(args[0])
	},
}

func printGeneralContributeGuide() {
	fmt.Print(`━━━ Contributing to ref examples ━━━

ref-examples repo: https://github.com/ref-cli/ref-examples

Each example file is a plain .txt file named after the command (e.g. tar.txt).

Example file format:
  ---
  tags: [ compression, archiving ]
  variants: [ bsd, gnu ]
  ---
  # To extract an archive:
  tar -xvf /path/to/foo.tar

  # [GNU] GNU-only flag example:
  tar -xvf foo.tar --checkpoint=100

Format rules:
  • One blank line between entries (comment + command pairs)
  • Comment lines must start with '# ' (hash + space)
  • Inline tags must be uppercase: [GNU], [BSD], [v2], [macOS], [Linux]
  • No trailing whitespace; LF line endings; UTF-8; final newline

Contribution steps:
  1. Fork https://github.com/ref-cli/ref-examples on GitHub
  2. Clone your fork:
       git clone git@github.com:<your-username>/ref-examples.git
  3. Add or edit the .txt file for your command
  4. Commit: git commit -m "examples: add <command>"
  5. Push and open a PR against ref-cli/ref-examples main

To contribute a specific command you have locally, run:
  ref contribute <command>
`)
}

func printCommandContribute(name string) error {
	path := filepath.Join(config.ExamplesDir(), name+".txt")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no example found for %q — run 'ref %s' first to generate one", name, name)
		}
		return err
	}
	content := strings.TrimSpace(string(data))

	divider := strings.Repeat("─", 50)

	fmt.Printf("━━━ Contributing '%s' to ref examples ━━━\n\n", name)
	fmt.Printf("Your file: %s\n", path)
	fmt.Println(divider)
	fmt.Println(content)
	fmt.Println(divider)

	fmt.Printf(`
── For AI Agents (Claude Code / Codex) ──────────────

Paste the following as a prompt to your AI agent:

  Fork https://github.com/ref-cli/ref-examples and clone it locally.
  Copy the file below into %s.txt (create if it doesn't exist;
  append if it does, avoiding duplicates). Commit with message
  "examples: add/update %s". Push to your fork and open a pull request
  against ref-cli/ref-examples main with title "examples: contribute %s"
  and a brief description of what was added.

  File content to add:
%s

── For Humans ────────────────────────────────────────

1. Fork https://github.com/ref-cli/ref-examples on GitHub
2. Clone your fork:
     git clone git@github.com:<your-username>/ref-examples.git
3. Copy your example:
     cp %s %s.txt
   (Or append to an existing file, removing duplicates.)
4. Commit:
     git add %s.txt
     git commit -m "examples: add/update %s"
5. Push and open a PR against ref-cli/ref-examples main:
     git push origin main
   Then visit https://github.com/ref-cli/ref-examples to open the pull request.
`,
		name, name, name, indent(content, "  "),
		path, name,
		name, name,
	)
	return nil
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}
