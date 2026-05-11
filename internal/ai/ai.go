package ai

import (
	"fmt"
	"os/exec"
	"strings"
)

type Backend string

const (
	Claude Backend = "claude"
	Codex  Backend = "codex"
)

// Detect returns available backends in preference order.
func Detect() []Backend {
	var found []Backend
	if _, err := exec.LookPath("claude"); err == nil {
		found = append(found, Claude)
	}
	if _, err := exec.LookPath("codex"); err == nil {
		found = append(found, Codex)
	}
	return found
}

// Generate calls the given backend to generate an example file for cmd.
func Generate(backend Backend, cmd string) (string, error) {
	prompt := buildPrompt(cmd)

	var command *exec.Cmd
	switch backend {
	case Claude:
		command = exec.Command("claude", "-p", prompt)
	case Codex:
		command = exec.Command("codex", prompt)
	default:
		return "", fmt.Errorf("unknown backend: %s", backend)
	}

	out, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", backend, err)
	}

	// Strip markdown fences if the model wrapped output anyway.
	result := strings.TrimSpace(string(out))
	result = stripFences(result)
	return result, nil
}

func buildPrompt(cmd string) string {
	return fmt.Sprintf(`Generate a ref example file for the '%s' command.
Format: plain text. Each entry is a comment line starting with '# ' describing the use case,
followed by the exact shell command on the next line.
Include 8-12 practical examples covering common developer workflows.
Output only the file content, no markdown fences, no extra explanation.`, cmd)
}

func stripFences(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return s
	}
	if strings.HasPrefix(lines[0], "```") {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
