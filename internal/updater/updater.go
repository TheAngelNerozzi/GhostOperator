// Package updater implements a secure, cross-platform auto-update system
// for GhostOperator. It checks GitHub releases for new versions, verifies
// SHA256 checksums, and performs atomic binary replacement.
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ReleaseManifest describes the latest available release.
// This file is generated and published by the GitHub Actions webhook workflow.
type ReleaseManifest struct {
	Version     string            `json:"version"`       // Semantic version (e.g. "2.0.0")
	Commit      string            `json:"commit"`        // Git commit SHA
	BuildTime   string            `json:"build_time"`    // ISO 8601 build timestamp
	Channel     string            `json:"channel"`       // "stable" or "beta"
	Assets      map[string]Asset  `json:"assets"`        // Key: filename, Value: metadata
	Changelog   string            `json:"changelog"`     // Release notes
	MinVersion  string            `json:"min_version"`   // Minimum version that can auto-update
	Mandatory   bool              `json:"mandatory"`     // Force update
	PublishedAt string            `json:"published_at"`  // RFC 3339 timestamp
}

// Asset describes a single downloadable binary artifact.
type Asset struct {
	URL      string `json:"url"`       // Direct download URL
	SHA256   string `json:"sha256"`    // Hex-encoded SHA256 checksum
	Size     int64  `json:"size"`      // File size in bytes
	Platform string `json:"platform"`  // e.g. "windows-amd64"
}

// UpdateStatus represents the result of an update check.
type UpdateStatus struct {
	Available    bool   `json:"available"`
	CurrentVer   string `json:"current_version"`
	LatestVer    string `json:"latest_version"`
	Changelog    string `json:"changelog"`
	Mandatory    bool   `json:"mandatory"`
	DownloadSize int64  `json:"download_size"`
	Error        string `json:"error,omitempty"`
}

// UpdateResult represents the outcome of an update download+install.
type UpdateResult struct {
	Success  bool   `json:"success"`
	Version  string `json:"version"`
	Path     string `json:"path"`
	Error    string `json:"error,omitempty"`
	Restart  bool   `json:"restart"` // True if restart is needed
}

// Config holds updater settings.
type Config struct {
	CheckInterval    time.Duration // How often to check for updates (default: 5 min)
	UpdateURL        string        // Base URL for the manifest (GitHub raw)
	CurrentVersion   string        // Version of the running binary
	AutoUpdate       bool          // Apply updates automatically without prompt
	Channel          string        // "stable" or "beta"
	EnablePrerelease bool          // Include pre-release versions
}

// DefaultConfig returns sensible defaults for the updater.
func DefaultConfig(currentVersion string) *Config {
	return &Config{
		CheckInterval:  5 * time.Minute,
		UpdateURL:      "https://raw.githubusercontent.com/TheAngelNerozzi/GhostOperator/main/releases/latest.json",
		CurrentVersion: currentVersion,
		AutoUpdate:     false,
		Channel:        "stable",
	}
}

// Updater is the main auto-update engine.
type Updater struct {
	config     *Config
	httpClient *http.Client
}

// New creates a new Updater with the given configuration.
func New(cfg *Config) *Updater {
	if cfg == nil {
		return nil
	}
	return &Updater{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			// Follow redirects for GitHub raw URLs
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// CheckForUpdates fetches the remote manifest and compares versions.
// Returns UpdateStatus with availability information.
func (u *Updater) CheckForUpdates(ctx context.Context) *UpdateStatus {
	status := &UpdateStatus{
		CurrentVer: u.config.CurrentVersion,
	}

	// Fetch the release manifest
	manifest, err := u.fetchManifest(ctx)
	if err != nil {
		status.Error = fmt.Sprintf("failed to fetch manifest: %v", err)
		return status
	}

	// Compare versions
	cmp := compareVersions(u.config.CurrentVersion, manifest.Version)
	if cmp >= 0 {
		// Already up to date or running a newer version (dev)
		status.LatestVer = manifest.Version
		return status
	}

	// Check minimum version requirement
	if manifest.MinVersion != "" {
		if compareVersions(u.config.CurrentVersion, manifest.MinVersion) < 0 {
			status.Mandatory = true
		}
	}

	// Find the correct asset for this platform
	assetKey := u.platformAssetKey()
	asset, ok := manifest.Assets[assetKey]
	if !ok {
		status.Error = fmt.Sprintf("no binary available for platform %s", assetKey)
		return status
	}

	status.Available = true
	status.LatestVer = manifest.Version
	status.Changelog = manifest.Changelog
	status.Mandatory = manifest.Mandatory || status.Mandatory
	status.DownloadSize = asset.Size

	return status
}

// DownloadAndUpdate downloads the latest binary and performs atomic replacement.
// It verifies the SHA256 checksum before replacing the current binary.
func (u *Updater) DownloadAndUpdate(ctx context.Context) *UpdateResult {
	result := &UpdateResult{}

	// Fetch manifest
	manifest, err := u.fetchManifest(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("failed to fetch manifest: %v", err)
		return result
	}

	// Find correct asset
	assetKey := u.platformAssetKey()
	asset, ok := manifest.Assets[assetKey]
	if !ok {
		result.Error = fmt.Sprintf("no binary available for platform %s", assetKey)
		return result
	}

	// Download to temporary file
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("ghost-update-%s", manifest.Version))
	if runtime.GOOS == "windows" {
		tempFile += ".exe"
	}

	fmt.Printf("[UPDATE] Descargando GhostOperator v%s...\n", manifest.Version)

	if err := u.downloadFile(ctx, asset.URL, tempFile); err != nil {
		result.Error = fmt.Sprintf("download failed: %v", err)
		return result
	}

	// Verify SHA256 checksum
	if err := verifyChecksum(tempFile, asset.SHA256); err != nil {
		os.Remove(tempFile)
		result.Error = fmt.Sprintf("checksum verification failed: %v", err)
		return result
	}

	// Get the path of the current executable
	execPath, err := os.Executable()
	if err != nil {
		os.Remove(tempFile)
		result.Error = fmt.Sprintf("cannot determine executable path: %v", err)
		return result
	}

	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		os.Remove(tempFile)
		result.Error = fmt.Sprintf("cannot resolve executable path: %v", err)
		return result
	}

	// Platform-specific atomic replacement
	if err := atomicReplace(tempFile, execPath); err != nil {
		os.Remove(tempFile)
		result.Error = fmt.Sprintf("failed to replace binary: %v", err)
		return result
	}

	result.Success = true
	result.Version = manifest.Version
	result.Path = execPath
	result.Restart = true

	fmt.Printf("[UPDATE] Actualizado a v%s exitosamente. Reiniciando...\n", manifest.Version)
	return result
}

// fetchManifest retrieves the remote release manifest.
func (u *Updater) fetchManifest(ctx context.Context) (*ReleaseManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.config.UpdateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GhostOperator-Updater/2.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var manifest ReleaseManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	return &manifest, nil
}

// downloadFile downloads a file from url to the local path.
func (u *Updater) downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "GhostOperator-Updater/2.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create the destination file
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	// Copy with progress tracking
	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Printf("[UPDATE] Descargados %.2f MB\n", float64(written)/(1024*1024))
	return nil
}

// verifyChecksum computes SHA256 of a file and compares it with the expected hash.
func verifyChecksum(filePath, expectedHex string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("hash file: %w", err)
	}

	computed := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(computed, expectedHex) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHex, computed)
	}

	fmt.Println("[UPDATE] Checksum SHA256 verificado correctamente.")
	return nil
}

// platformAssetKey returns the asset key for the current OS/ARCH combination.
func (u *Updater) platformAssetKey() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	switch os {
	case "windows":
		if arch == "amd64" {
			return "ghost-windows-amd64.exe"
		}
		return "ghost-windows-386.exe"
	case "linux":
		return "ghost-linux-amd64"
	case "darwin":
		if arch == "arm64" {
			return "ghost-darwin-arm64"
		}
		return "ghost-darwin-amd64"
	default:
		return fmt.Sprintf("ghost-%s-%s", os, arch)
	}
}

// atomicReplace replaces the old binary with the new one atomically.
// On Windows, it uses a move-and-restart strategy since files can't be
// replaced while in use. On Unix, it uses os.Rename which is atomic.
func atomicReplace(newPath, oldPath string) error {
	// Make the new file executable
	if err := os.Chmod(newPath, 0755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	if runtime.GOOS == "windows" {
		// Windows: current binary is locked, use move-and-batch strategy
		return atomicReplaceWindows(newPath, oldPath)
	}

	// Unix: atomic rename
	if err := os.Rename(newPath, oldPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// atomicReplaceWindows handles binary replacement on Windows where the
// running executable is locked. It uses a helper script strategy:
// 1. Create a batch script that waits and then replaces
// 2. The batch script runs after the current process exits
func atomicReplaceWindows(newPath, oldPath string) error {
	batchPath := filepath.Join(os.TempDir(), "ghost-update.bat")

	// Escape paths for batch script
	escapedNew := strings.ReplaceAll(newPath, "/", "\\")
	escapedOld := strings.ReplaceAll(oldPath, "/", "\\")

	// Create batch script: wait for ghost to exit, then copy and restart
	batchContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
move /y "%s" "%s" >nul 2>&1
start "" "%s"
del "%%~f0"
`, escapedNew, escapedOld, escapedOld)

	if err := os.WriteFile(batchPath, []byte(batchContent), 0644); err != nil {
		return fmt.Errorf("create batch script: %w", err)
	}

	// Launch the batch script detached
	cmd := createWindowsCommand(batchPath)
	if err := cmd.Start(); err != nil {
		os.Remove(batchPath)
		return fmt.Errorf("launch update script: %w", err)
	}

	// The actual replacement will happen after this process exits
	return nil
}

// compareVersions compares two semantic version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Supports formats: "1.0.0", "1.0.0-dev", "2.0.0-beta.1"
func compareVersions(a, b string) int {
	// Normalize: strip "v" prefix
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	// Split off prerelease suffixes (everything after "-")
	aVersion, aPre := splitPrerelease(a)
	bVersion, bPre := splitPrerelease(b)

	// Compare main version parts
	aParts := parseVersionParts(aVersion)
	bParts := parseVersionParts(bVersion)

	for i := 0; i < 3; i++ {
		aVal := 0
		bVal := 0
		if i < len(aParts) {
			aVal = aParts[i]
		}
		if i < len(bParts) {
			bVal = bParts[i]
		}
		if aVal < bVal {
			return -1
		}
		if aVal > bVal {
			return 1
		}
	}

	// If main versions are equal, handle prerelease
	// A version WITHOUT a prerelease is newer than one WITH
	if aPre == "" && bPre != "" {
		return 1
	}
	if aPre != "" && bPre == "" {
		return -1
	}

	// Both have prerelease — compare lexicographically
	if aPre < bPre {
		return -1
	}
	if aPre > bPre {
		return 1
	}

	return 0
}

// splitPrerelease splits "1.0.0-beta.1" into ("1.0.0", "beta.1").
func splitPrerelease(v string) (version, prerelease string) {
	idx := strings.Index(v, "-")
	if idx == -1 {
		return v, ""
	}
	return v[:idx], v[idx+1:]
}

// parseVersionParts splits "1.2.3" into [1, 2, 3].
func parseVersionParts(v string) []int {
	parts := strings.Split(v, ".")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		var n int
		for _, c := range p {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			} else {
				break
			}
		}
		result = append(result, n)
	}
	return result
}
