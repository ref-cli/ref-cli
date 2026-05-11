package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ref-cli/ref-cli/internal/config"
	"github.com/ref-cli/ref-cli/internal/example"
	"github.com/ref-cli/ref-cli/internal/examples"
)

// runSearchPhrase handles `ref -s <phrase>`.
// Scoring: count how many query words appear (case-insensitive) in the target
// text (command name + entry comment). Entries with all words score highest.
func runSearchPhrase(phrase string) error {
	dir := examples.Dir(config.ExamplesDir())
	exs, err := examples.LoadAll(dir)
	if err != nil {
		return err
	}

	queryWords := strings.Fields(strings.ToLower(phrase))
	if len(queryWords) == 0 {
		return nil
	}

	type scored struct {
		ex    *example.Example
		entry example.Entry
		score int
	}

	var hits []scored
	for _, ex := range exs {
		for _, e := range ex.Entries {
			target := strings.ToLower(ex.Name + " " + e.Comment + " " + e.Command)
			score := 0
			for _, w := range queryWords {
				if strings.Contains(target, w) {
					score++
				}
			}
			if score > 0 {
				hits = append(hits, scored{ex: ex, entry: e, score: score})
			}
		}
	}

	if len(hits) == 0 {
		fmt.Printf("No examples found matching %q\n", phrase)
		return nil
	}

	// Sort by score descending, then by command name for stability.
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].ex.Name < hits[j].ex.Name
	})

	cs := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#555555", Dark: "#888888"})
	ts := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#CC7700", Dark: "#FFAF5F"})
	nameStyle := lipgloss.NewStyle().Bold(true)

	seen := map[string]bool{}
	printed := 0
	for _, h := range hits {
		key := h.ex.Name + "|" + h.entry.Command
		if seen[key] {
			continue
		}
		seen[key] = true

		if printed > 0 {
			fmt.Println()
		}
		fmt.Printf("[%s] ", nameStyle.Render(h.ex.Name))

		commentLine := "# "
		for _, t := range h.entry.Tags {
			commentLine += ts.Render("["+t+"]") + " "
		}
		commentLine += cs.Render(strings.TrimSpace(h.entry.Comment))
		fmt.Println(commentLine)
		fmt.Println(h.entry.Command)
		printed++

		if printed >= 10 {
			break
		}
	}
	return nil
}
