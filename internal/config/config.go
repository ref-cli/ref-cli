package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ExamplesVersion       string `yaml:"examples_version"`
	LastUpdateCheck       string `yaml:"last_update_check"`
	InstallMethod         string `yaml:"install_method"`
	AIBackend             string `yaml:"ai_backend"`
	PendingBinaryUpdate   string `yaml:"pending_binary_update,omitempty"`
	PendingExamplesUpdate string `yaml:"pending_examples_update,omitempty"`
}

func Dir() string {
	if d := os.Getenv("REF_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ref")
}

func ExamplesDir() string {
	return filepath.Join(Dir(), "examples")
}

func File() string {
	return filepath.Join(Dir(), "conf.yml")
}

func ChecksumsFile() string {
	return filepath.Join(Dir(), ".checksums")
}

func IsInitialized() bool {
	_, err := os.Stat(File())
	return err == nil
}

func Load() (*Config, error) {
	data, err := os.ReadFile(File())
	if err != nil {
		return &Config{}, nil
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		fmt.Fprintf(os.Stderr, "ref: config file is malformed (%v); using defaults\n", err)
		return &Config{}, err
	}
	return &c, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(File(), data, 0o644)
}

func (c *Config) IsUpdateCheckDue() bool {
	if strings.TrimSpace(c.LastUpdateCheck) == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, c.LastUpdateCheck)
	if err != nil {
		return true
	}
	return time.Since(t) > 7*24*time.Hour
}

// DetectInstallMethod returns "homebrew", "manual", or "source".
func DetectInstallMethod() string {
	exe, err := os.Executable()
	if err != nil {
		return "manual"
	}
	if strings.Contains(exe, "Cellar") || strings.Contains(exe, "homebrew") || strings.Contains(exe, "linuxbrew") {
		return "homebrew"
	}
	gopath := os.Getenv("GOPATH")
	if gopath != "" && strings.HasPrefix(exe, gopath) {
		return "source"
	}
	return "manual"
}
