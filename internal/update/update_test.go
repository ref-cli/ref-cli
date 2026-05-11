package update

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestSaveAndLoadChecksums(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "checksums")

	m := map[string]string{
		"tar.txt":  "abc123",
		"git.txt":  "def456",
		"curl.txt": "ghi789",
	}

	if err := SaveChecksums(path, m); err != nil {
		t.Fatalf("SaveChecksums error: %v", err)
	}

	loaded, err := loadChecksums(path)
	if err != nil {
		t.Fatalf("loadChecksums error: %v", err)
	}
	if len(loaded) != len(m) {
		t.Fatalf("expected %d entries, got %d", len(m), len(loaded))
	}
	for k, v := range m {
		if loaded[k] != v {
			t.Errorf("key %q: expected %q, got %q", k, v, loaded[k])
		}
	}
}

func TestSaveChecksumsIsDeterministic(t *testing.T) {
	tmp := t.TempDir()

	m := map[string]string{
		"zzz.txt": "sum1",
		"aaa.txt": "sum2",
		"mmm.txt": "sum3",
	}

	path1 := filepath.Join(tmp, "cs1")
	path2 := filepath.Join(tmp, "cs2")

	if err := SaveChecksums(path1, m); err != nil {
		t.Fatal(err)
	}
	if err := SaveChecksums(path2, m); err != nil {
		t.Fatal(err)
	}

	d1, _ := os.ReadFile(path1)
	d2, _ := os.ReadFile(path2)
	if string(d1) != string(d2) {
		t.Errorf("SaveChecksums is non-deterministic:\n%s\nvs\n%s", d1, d2)
	}

	// Verify output is sorted by filename.
	lines := strings.Split(strings.TrimSpace(string(d1)), "\n")
	names := make([]string, len(lines))
	for i, l := range lines {
		parts := strings.SplitN(l, "  ", 2)
		if len(parts) == 2 {
			names[i] = parts[1]
		}
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("checksums not in sorted order: %v", names)
	}
}

func TestLoadChecksumsEmpty(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "checksums")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := loadChecksums(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestLoadChecksumsMissing(t *testing.T) {
	_, err := loadChecksums("/nonexistent/path/checksums")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestSaveChecksumsEmpty(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "checksums")
	if err := SaveChecksums(path, map[string]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(path)
	if len(strings.TrimSpace(string(data))) != 0 {
		t.Errorf("expected empty file, got %q", data)
	}
}
