package examples_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ref-cli/ref-cli/internal/examples"
)

func TestLoadAllEmpty(t *testing.T) {
	tmp := t.TempDir()
	exs, err := examples.LoadAll(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exs) != 0 {
		t.Errorf("expected 0 examples, got %d", len(exs))
	}
}

func TestLoadAllSkipsNonTxt(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"tar.txt":  "# Extract\ntar -xvf foo.tar\n",
		"git.txt":  "# Clone\ngit clone https://example.com/repo\n",
		"README":   "not a command file",
		"notes.md": "also not a command file",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	exs, err := examples.LoadAll(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exs) != 2 {
		t.Fatalf("expected 2 examples, got %d", len(exs))
	}
	// Results are sorted by name.
	if exs[0].Name != "git" {
		t.Errorf("expected first to be 'git', got %q", exs[0].Name)
	}
	if exs[1].Name != "tar" {
		t.Errorf("expected second to be 'tar', got %q", exs[1].Name)
	}
}

func TestLoadAllSorted(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"zzz.txt", "aaa.txt", "mmm.txt"} {
		content := "# Use\ncmd\n"
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	exs, err := examples.LoadAll(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if exs[0].Name != "aaa" || exs[1].Name != "mmm" || exs[2].Name != "zzz" {
		names := make([]string, len(exs))
		for i, e := range exs {
			names[i] = e.Name
		}
		t.Errorf("not sorted: %v", names)
	}
}

func TestFindExists(t *testing.T) {
	tmp := t.TempDir()
	content := "# Extract\ntar -xvf foo.tar\n"
	if err := os.WriteFile(filepath.Join(tmp, "tar.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	ex, err := examples.Find(tmp, "tar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ex == nil {
		t.Fatal("expected non-nil example")
	}
	if ex.Name != "tar" {
		t.Errorf("expected name 'tar', got %q", ex.Name)
	}
	if len(ex.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(ex.Entries))
	}
}

func TestFindMissing(t *testing.T) {
	tmp := t.TempDir()
	ex, err := examples.Find(tmp, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if ex != nil {
		t.Error("expected nil for missing example")
	}
}

func TestDirEnvOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("REF_EXAMPLES_DIR", tmp)
	got := examples.Dir("")
	if got != tmp {
		t.Errorf("expected %q, got %q", tmp, got)
	}
}
