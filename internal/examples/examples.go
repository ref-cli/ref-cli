package examples

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ref-cli/ref-cli/internal/example"
)

func resolveExamplesDir(dir string) string {
	if entries, err := os.ReadDir(dir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
				return dir
			}
		}
	}
	if info, err := os.Stat(filepath.Join(dir, "examples")); err == nil && info.IsDir() {
		return filepath.Join(dir, "examples")
	}
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	return dir
}

// Dir returns the examples directory to use:
// 1. $REF_EXAMPLES_DIR if set
// 2. ../ref-examples/ if it exists (dev mode)
// 3. ~/.config/ref/examples/
func Dir(configExamplesDir string) string {
	if d := os.Getenv("REF_EXAMPLES_DIR"); d != "" {
		return resolveExamplesDir(d)
	}
	// Dev mode: sibling repo's examples/ subdir exists relative to CWD
	if info, err := os.Stat("../ref-examples/examples"); err == nil && info.IsDir() {
		abs, err := filepath.Abs("../ref-examples/examples")
		if err == nil {
			return abs
		}
	}
	return configExamplesDir
}

// LoadAll loads every *.txt file from dir and returns examples sorted by name.
func LoadAll(dir string) ([]*example.Example, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var exs []*example.Example
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".txt")
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		exs = append(exs, example.Parse(name, string(data)))
	}

	sort.Slice(exs, func(i, j int) bool {
		return exs[i].Name < exs[j].Name
	})
	return exs, nil
}

// Find returns the example for the given command name, or nil.
func Find(dir, name string) (*example.Example, error) {
	path := filepath.Join(dir, name+".txt")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return example.Parse(name, string(data)), nil
}
