package updater

import (
        "context"
        "encoding/hex"
        "encoding/json"
        "fmt"
        "net/http"
        "net/http/httptest"
        "os"
        "path/filepath"
        "runtime"
        "testing"
        "time"

        "crypto/sha256"

        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
        tests := []struct {
                a, b     string
                expected int
        }{
                {"1.0.0", "1.0.0", 0},
                {"1.0.0", "2.0.0", -1},
                {"2.0.0", "1.0.0", 1},
                {"1.0.0", "1.0.1", -1},
                {"1.0.1", "1.0.0", 1},
                {"1.1.0", "1.0.9", 1},
                {"1.0.0", "1.0.0-dev", 1},       // release > prerelease
                {"1.0.0-dev", "1.0.0", -1},       // prerelease < release
                {"2.0.0", "2.0.0-beta.1", 1},     // release > prerelease
                {"v1.0.0", "1.0.0", 0},           // v-prefix normalization
                {"2.0.0-dev", "1.4.0", 1},        // dev > old stable
                {"1.4.0", "2.0.0", -1},           // old < new
                {"0.9.9", "1.0.0", -1},           // 0.x < 1.x
        }

        for _, tt := range tests {
                t.Run(fmt.Sprintf("%s_vs_%s", tt.a, tt.b), func(t *testing.T) {
                        result := compareVersions(tt.a, tt.b)
                        assert.Equal(t, tt.expected, result)
                })
        }
}

func TestSplitPrerelease(t *testing.T) {
        tests := []struct {
                input         string
                wantVersion   string
                wantPrerelease string
        }{
                {"1.0.0", "1.0.0", ""},
                {"1.0.0-dev", "1.0.0", "dev"},
                {"2.0.0-beta.1", "2.0.0", "beta.1"},
                {"3.0.0-rc.2", "3.0.0", "rc.2"},
        }

        for _, tt := range tests {
                v, p := splitPrerelease(tt.input)
                assert.Equal(t, tt.wantVersion, v)
                assert.Equal(t, tt.wantPrerelease, p)
        }
}

func TestPlatformAssetKey(t *testing.T) {
        u := &Updater{config: &Config{}}
        key := u.platformAssetKey()

        switch runtime.GOOS {
        case "windows":
                assert.Contains(t, key, "windows")
                assert.Contains(t, key, ".exe")
        case "linux":
                assert.Contains(t, key, "linux")
                assert.Contains(t, key, "amd64")
        case "darwin":
                assert.Contains(t, key, "darwin")
        default:
                assert.Contains(t, key, runtime.GOOS)
        }
}

func TestCheckForUpdates_Available(t *testing.T) {
        // Create a fake manifest
        manifest := ReleaseManifest{
                Version:     "2.1.0",
                Commit:      "abc123",
                BuildTime:   time.Now().UTC().Format(time.RFC3339),
                Channel:     "stable",
                PublishedAt: time.Now().UTC().Format(time.RFC3339),
                Changelog:   "New features added",
                Assets: map[string]Asset{
                        "ghost-linux-amd64": {
                                URL:      "https://example.com/ghost-linux-amd64",
                                SHA256:   "abc123",
                                Size:     1024000,
                                Platform: "linux-amd64",
                        },
                        "ghost-windows-amd64.exe": {
                                URL:      "https://example.com/ghost-windows-amd64.exe",
                                SHA256:   "def456",
                                Size:     1152000,
                                Platform: "windows-amd64",
                        },
                        "ghost-darwin-amd64": {
                                URL:      "https://example.com/ghost-darwin-amd64",
                                SHA256:   "ghi789",
                                Size:     1080000,
                                Platform: "darwin-amd64",
                        },
                        "ghost-darwin-arm64": {
                                URL:      "https://example.com/ghost-darwin-arm64",
                                SHA256:   "jkl012",
                                Size:     1050000,
                                Platform: "darwin-arm64",
                        },
                },
        }

        manifestBytes, _ := json.Marshal(manifest)

        // Create test server
        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.Write(manifestBytes)
        }))
        defer srv.Close()

        updater := New(&Config{
                CurrentVersion: "1.4.0",
                UpdateURL:      srv.URL,
                CheckInterval:  1 * time.Hour,
        })

        status := updater.CheckForUpdates(context.Background())
        assert.True(t, status.Available)
        assert.Equal(t, "2.1.0", status.LatestVer)
        assert.Equal(t, "1.4.0", status.CurrentVer)
        assert.Equal(t, "New features added", status.Changelog)
        assert.Empty(t, status.Error)
}

func TestCheckForUpdates_AlreadyUpToDate(t *testing.T) {
        manifest := ReleaseManifest{
                Version:   "1.4.0",
                Channel:   "stable",
                BuildTime: time.Now().UTC().Format(time.RFC3339),
                Assets:    map[string]Asset{},
        }

        manifestBytes, _ := json.Marshal(manifest)

        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.Write(manifestBytes)
        }))
        defer srv.Close()

        updater := New(&Config{
                CurrentVersion: "2.0.0-dev",
                UpdateURL:      srv.URL,
        })

        status := updater.CheckForUpdates(context.Background())
        assert.False(t, status.Available)
        assert.Equal(t, "1.4.0", status.LatestVer)
}

func TestCheckForUpdates_ServerError(t *testing.T) {
        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusInternalServerError)
        }))
        defer srv.Close()

        updater := New(&Config{
                CurrentVersion: "1.4.0",
                UpdateURL:      srv.URL,
        })

        status := updater.CheckForUpdates(context.Background())
        assert.False(t, status.Available)
        assert.NotEmpty(t, status.Error)
}

func TestVerifyChecksum(t *testing.T) {
        // Create a test file with known content
        content := []byte("ghost-operator-test-binary-content")
        tmpDir := t.TempDir()
        testFile := filepath.Join(tmpDir, "test-binary")

        err := os.WriteFile(testFile, content, 0644)
        require.NoError(t, err)

        // Compute the correct hash
        hash := sha256.Sum256(content)
        correctHash := hex.EncodeToString(hash[:])

        // Test correct checksum
        err = verifyChecksum(testFile, correctHash)
        assert.NoError(t, err)

        // Test incorrect checksum
        err = verifyChecksum(testFile, "0000000000000000000000000000000000000000000000000000000000000000")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestDownloadAndUpdate(t *testing.T) {
        if runtime.GOOS == "windows" {
                t.Skip("Skipping atomic replace test on Windows (would need actual binary)")
        }

        // Create fake binary content
        content := []byte("#!/bin/sh\necho 'GhostOperator v3.0.0'\n")
        hash := sha256.Sum256(content)
        hashHex := hex.EncodeToString(hash[:])

        // Create manifest pointing to test server
        manifest := ReleaseManifest{
                Version:   "3.0.0",
                Channel:   "stable",
                BuildTime: time.Now().UTC().Format(time.RFC3339),
                Changelog: "Major update",
                Assets:    map[string]Asset{},
        }

        // Add the current platform's asset
        u := &Updater{config: &Config{}}
        assetKey := u.platformAssetKey()
        manifest.Assets[assetKey] = Asset{
                SHA256:   hashHex,
                Size:     int64(len(content)),
                Platform: fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH),
        }

        // Create test servers
        downloadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Write(content)
        }))
        defer downloadSrv.Close()

        // Set the download URL in the manifest
        asset := manifest.Assets[assetKey]
        asset.URL = downloadSrv.URL + "/binary"
        manifest.Assets[assetKey] = asset
        manifestBytes, _ := json.Marshal(manifest)

        manifestSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.Write(manifestBytes)
        }))
        defer manifestSrv.Close()

        // Create a fake "current executable" in a temp dir
        tmpDir := t.TempDir()
        fakeExe := filepath.Join(tmpDir, "ghost-test")
        err := os.WriteFile(fakeExe, []byte("old-binary"), 0755)
        require.NoError(t, err)

        // Note: We can't fully test atomic replacement since os.Executable()
        // returns the actual test binary path. But we test download + checksum.
        updater := New(&Config{
                CurrentVersion: "1.4.0",
                UpdateURL:      manifestSrv.URL,
        })

        // Just test the download part
        result := updater.DownloadAndUpdate(context.Background())
        // This may fail on atomic replace since we can't override os.Executable(),
        // but download and checksum should work
        if result.Error != "" {
                assert.Contains(t, result.Error, "replace binary")
        } else {
                assert.True(t, result.Success)
        }
}

func TestDefaultConfig(t *testing.T) {
        cfg := DefaultConfig("2.0.0")
        assert.Equal(t, "2.0.0", cfg.CurrentVersion)
        assert.Equal(t, 5*time.Minute, cfg.CheckInterval)
        assert.Equal(t, "stable", cfg.Channel)
        assert.False(t, cfg.AutoUpdate)
        assert.Contains(t, cfg.UpdateURL, "githubusercontent")
}

func TestNew_Nil(t *testing.T) {
        u := New(nil)
        assert.Nil(t, u)
}

func TestNew_Valid(t *testing.T) {
        cfg := DefaultConfig("1.0.0")
        u := New(cfg)
        require.NotNil(t, u)
        assert.NotNil(t, u.httpClient)
}

func TestCheckForUpdates_NoPlatformAsset(t *testing.T) {
        manifest := ReleaseManifest{
                Version:   "9.0.0",
                Channel:   "stable",
                BuildTime: time.Now().UTC().Format(time.RFC3339),
                Assets:    map[string]Asset{}, // Empty — no assets at all
        }

        manifestBytes, _ := json.Marshal(manifest)

        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.Write(manifestBytes)
        }))
        defer srv.Close()

        updater := New(&Config{
                CurrentVersion: "1.4.0",
                UpdateURL:      srv.URL,
        })

        status := updater.CheckForUpdates(context.Background())
        assert.False(t, status.Available)
        assert.Contains(t, status.Error, "no binary available")
}

func TestCheckForUpdates_MandatoryUpdate(t *testing.T) {
        manifest := ReleaseManifest{
                Version:    "5.0.0",
                Channel:    "stable",
                BuildTime:  time.Now().UTC().Format(time.RFC3339),
                MinVersion: "3.0.0",
                Mandatory:  true,
                Assets: map[string]Asset{
                        "ghost-linux-amd64": {
                                URL:    "https://example.com/ghost",
                                SHA256: "abc",
                                Size:   100,
                        },
                        "ghost-windows-amd64.exe": {
                                URL:    "https://example.com/ghost.exe",
                                SHA256: "def",
                                Size:   100,
                        },
                        "ghost-darwin-amd64": {
                                URL:    "https://example.com/ghost",
                                SHA256: "ghi",
                                Size:   100,
                        },
                        "ghost-darwin-arm64": {
                                URL:    "https://example.com/ghost",
                                SHA256: "jkl",
                                Size:   100,
                        },
                },
                Changelog: "Critical security update",
        }

        manifestBytes, _ := json.Marshal(manifest)

        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.Write(manifestBytes)
        }))
        defer srv.Close()

        updater := New(&Config{
                CurrentVersion: "1.4.0", // Below MinVersion
                UpdateURL:      srv.URL,
        })

        status := updater.CheckForUpdates(context.Background())
        assert.True(t, status.Available)
        assert.True(t, status.Mandatory)
        assert.Equal(t, "Critical security update", status.Changelog)
}
