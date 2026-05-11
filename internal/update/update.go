package update

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	binaryReleasesURL   = "https://api.github.com/repos/ref-cli/ref-cli/releases/latest"
	examplesReleasesURL = "https://api.github.com/repos/ref-cli/ref-examples/releases/latest"
	examplesZipURL      = "https://github.com/ref-cli/ref-examples/archive/refs/tags/%s.zip"
	timeout             = 3 * time.Second
)

type CheckResult struct {
	BinaryUpdate   string // new binary version if available, empty otherwise
	ExamplesUpdate string // new examples version if available, empty otherwise
}

type releaseResponse struct {
	TagName string `json:"tag_name"`
}

func latestTag(url string) (string, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ref-cli")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	var r releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	return r.TagName, nil
}

// Check queries both GitHub release APIs (with 3s timeout each) and returns
// available updates. Returns nil if both calls fail.
func Check(currentBinaryVersion, installedExamplesVersion string) *CheckResult {
	result := &CheckResult{}
	gotAny := false

	if tag, err := latestTag(binaryReleasesURL); err == nil {
		gotAny = true
		if tag != "" && tag != currentBinaryVersion && tag != "v"+currentBinaryVersion {
			result.BinaryUpdate = tag
		}
	}

	if tag, err := latestTag(examplesReleasesURL); err == nil {
		gotAny = true
		if tag != "" && tag != installedExamplesVersion {
			result.ExamplesUpdate = tag
		}
	}

	if !gotAny {
		return nil
	}
	return result
}

// SyncResult is returned by Sync.
type SyncResult struct {
	Version string
	Updated []string
	Skipped []string
}

// Sync downloads the given ref-examples tag (or latest if empty) and extracts
// .txt files, skipping locally modified ones.
func Sync(examplesDir, checksumFile, requestedVersion string) (*SyncResult, error) {
	version := requestedVersion
	if version == "" {
		var err error
		version, err = latestTag(examplesReleasesURL)
		if err != nil {
			return nil, fmt.Errorf("examples sync failed: %w", err)
		}
	}

	if err := os.MkdirAll(examplesDir, 0o755); err != nil {
		return nil, fmt.Errorf("examples sync: %w", err)
	}

	zipURL := fmt.Sprintf(examplesZipURL, version)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(zipURL)
	if err != nil {
		return nil, fmt.Errorf("examples sync failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("examples sync: HTTP %d from %s", resp.StatusCode, zipURL)
	}

	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("examples sync failed: %w", err)
	}

	checksums, err := loadChecksums(checksumFile)
	if err != nil {
		checksums = map[string]string{}
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("bad zip: %w", err)
	}

	result := &SyncResult{Version: version}

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// GitHub zips wrap in a top-level dir: ref-examples-v0.1.0/examples/tar.txt
		parts := strings.SplitN(f.Name, "/", 3)
		if len(parts) != 3 || parts[1] != "examples" {
			continue
		}
		basename := parts[2]
		if !strings.HasSuffix(basename, ".txt") {
			continue
		}
		cmdName := strings.TrimSuffix(basename, ".txt")

		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			continue
		}

		dest := filepath.Join(examplesDir, basename)
		newSum := fmt.Sprintf("%x", sha256.Sum256(data))

		// If a checksum exists and the file on disk has been modified, skip it.
		if knownSum, ok := checksums[basename]; ok {
			existingData, err := os.ReadFile(dest)
			if err == nil {
				existingSum := fmt.Sprintf("%x", sha256.Sum256(existingData))
				if existingSum != knownSum {
					result.Skipped = append(result.Skipped, cmdName)
					continue
				}
			}
		}

		if err := os.WriteFile(dest, data, 0o644); err != nil {
			continue
		}
		checksums[basename] = newSum
		result.Updated = append(result.Updated, cmdName)
	}

	_ = SaveChecksums(checksumFile, checksums)
	return result, nil
}

func loadChecksums(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		// format: "<sha256hex>  <filename>"
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) == 2 {
			m[parts[1]] = parts[0]
		}
	}
	return m, nil
}

// SaveChecksums writes the checksum map to path in sorted, deterministic order.
func SaveChecksums(path string, m map[string]string) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&sb, "%s  %s\n", m[k], k)
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}
