package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ref-cli/ref-cli/internal/config"
)

func setupConfigDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("REF_CONFIG_DIR", tmp)
	return tmp
}

func TestLoadMissingFile(t *testing.T) {
	setupConfigDir(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ExamplesVersion != "" || cfg.InstallMethod != "" {
		t.Errorf("expected empty defaults, got %+v", cfg)
	}
}

func TestLoadValidYAML(t *testing.T) {
	dir := setupConfigDir(t)
	content := "examples_version: v0.2.0\ninstall_method: homebrew\n"
	if err := os.WriteFile(filepath.Join(dir, "conf.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ExamplesVersion != "v0.2.0" {
		t.Errorf("expected v0.2.0, got %q", cfg.ExamplesVersion)
	}
	if cfg.InstallMethod != "homebrew" {
		t.Errorf("expected homebrew, got %q", cfg.InstallMethod)
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	dir := setupConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "conf.yml"), []byte("not:\n  valid:\n    yaml: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	// Malformed YAML → error is returned AND zero-value defaults are used.
	if err == nil {
		t.Fatal("Load() should return an error for malformed YAML")
	}
	if cfg.ExamplesVersion != "" {
		t.Errorf("expected empty ExamplesVersion on malformed YAML, got %q", cfg.ExamplesVersion)
	}
}

func TestSaveAndLoad(t *testing.T) {
	setupConfigDir(t)
	cfg := &config.Config{
		ExamplesVersion: "v1.0.0",
		InstallMethod:   "manual",
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.ExamplesVersion != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %q", loaded.ExamplesVersion)
	}
	if loaded.InstallMethod != "manual" {
		t.Errorf("expected manual, got %q", loaded.InstallMethod)
	}
}

func TestIsUpdateCheckDue(t *testing.T) {
	tests := []struct {
		name    string
		last    string
		wantDue bool
	}{
		{"empty", "", true},
		{"invalid timestamp", "not-a-time", true},
		{"recent (1 hour ago)", time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339), false},
		{"stale (8 days ago)", time.Now().Add(-8 * 24 * time.Hour).UTC().Format(time.RFC3339), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{LastUpdateCheck: tt.last}
			if got := cfg.IsUpdateCheckDue(); got != tt.wantDue {
				t.Errorf("IsUpdateCheckDue() = %v, want %v", got, tt.wantDue)
			}
		})
	}
}

func TestPendingUpdateFields(t *testing.T) {
	setupConfigDir(t)
	cfg := &config.Config{
		PendingBinaryUpdate:   "v2.0.0",
		PendingExamplesUpdate: "v0.5.0",
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.PendingBinaryUpdate != "v2.0.0" {
		t.Errorf("PendingBinaryUpdate: got %q", loaded.PendingBinaryUpdate)
	}
	if loaded.PendingExamplesUpdate != "v0.5.0" {
		t.Errorf("PendingExamplesUpdate: got %q", loaded.PendingExamplesUpdate)
	}
}
