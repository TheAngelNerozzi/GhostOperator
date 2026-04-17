package core

import (
        "fmt"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
        "runtime"
        "time"
)

// healthClient is a shared HTTP client for Ollama health checks.
var healthClient = &http.Client{Timeout: 2 * time.Second}

// getMarkerPath returns the absolute path for the first-run marker file,
// using the OS-specific user config directory.
func getMarkerPath() (string, error) {
        configDir, err := os.UserConfigDir()
        if err != nil {
                return "", fmt.Errorf("failed to determine user config dir: %w", err)
        }
        markerDir := filepath.Join(configDir, "ghostoperator")
        if err := os.MkdirAll(markerDir, 0755); err != nil {
                return "", fmt.Errorf("failed to create config directory %s: %w", markerDir, err)
        }
        return filepath.Join(markerDir, ".ghost_setup_done"), nil
}

// IsFirstRun returns true if the setup marker file doesn't exist.
func IsFirstRun() bool {
        markerPath, err := getMarkerPath()
        if err != nil {
                fmt.Fprintf(os.Stderr, "IsFirstRun: %v\n", err)
                return true
        }
        _, err = os.Stat(markerPath)
        return os.IsNotExist(err)
}

// MarkSetupDone creates the marker file so we don't prompt again.
func MarkSetupDone() {
        markerPath, err := getMarkerPath()
        if err != nil {
                fmt.Fprintf(os.Stderr, "MarkSetupDone: %v\n", err)
                return
        }
        if err := os.WriteFile(markerPath, []byte("v1.1.0-local"), 0644); err != nil {
                fmt.Fprintf(os.Stderr, "MarkSetupDone: failed to write marker file: %v\n", err)
        }
}

// resolveOllamaBinary tries to find the ollama binary in known install locations
// before falling back to PATH lookup.
func resolveOllamaBinary() string {
        candidates := []string{}

        switch runtime.GOOS {
        case "windows":
                if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
                        candidates = append(candidates, filepath.Join(dir, "Programs", "Ollama", "ollama.exe"))
                }
        case "darwin":
                candidates = append(candidates, "/usr/local/bin/ollama", "/opt/homebrew/bin/ollama")
        case "linux":
                candidates = append(candidates, "/usr/local/bin/ollama", "/usr/bin/ollama", "/snap/bin/ollama")
        }

        for _, p := range candidates {
                if _, err := os.Stat(p); err == nil {
                        return p
                }
        }

        // Fallback to PATH, but log a warning
        fmt.Fprintf(os.Stderr, "\033[1;33m[WARN]\033[0m ollama binary not found in known locations; falling back to PATH\n")
        return "ollama"
}

// EnsureOllamaRunning attempts to start Ollama if it's not reachable.
func EnsureOllamaRunning() bool {
        fmt.Print("  - Verificando servidor Ollama... ")
        _, err := healthClient.Get("http://127.0.0.1:11434/api/version")
        if err == nil {
                fmt.Println("\033[1;32mOK\033[0m")
                return true
        }

        ollamaBin := resolveOllamaBinary()

        fmt.Print("\033[1;33mIntentando iniciar...\033[0m ")
        var cmd *exec.Cmd
        if runtime.GOOS == "windows" {
                cmd = exec.Command("cmd", "/C", "start", "", ollamaBin, "serve")
        } else {
                cmd = exec.Command(ollamaBin, "serve")
        }

        // On Unix, detach the child process into its own process group
        // so it doesn't become a zombie when the parent exits.
        setProcessGroup(cmd)

        if err := cmd.Start(); err != nil {
                fmt.Println("\033[1;31mError\033[0m")
                return false
        }

        // Wait for server to wake up
        for i := 0; i < 5; i++ {
                time.Sleep(2 * time.Second)
                _, err := healthClient.Get("http://127.0.0.1:11434/api/version")
                if err == nil {
                        fmt.Println("\033[1;32mOK\033[0m")
                        return true
                }
        }

        fmt.Println("\033[1;31mNo responde\033[0m")
        return false
}

// EnsureModel pulls the required moondream model if missing.
func EnsureModel(modelName string) {
        fmt.Printf("  - Verificando modelo %s... ", modelName)
        ollamaBin := resolveOllamaBinary()
        cmd := exec.Command(ollamaBin, "pull", modelName)
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        if err := cmd.Run(); err != nil {
                fmt.Printf("\033[1;31mError al descargar %s\033[0m\n", modelName)
        } else {
                fmt.Println("\033[1;32mListo\033[0m")
        }
}
