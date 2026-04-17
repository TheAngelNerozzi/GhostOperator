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
var healthClient = &http.Client{Timeout: 3 * time.Second}

// CheckOllamaReady quickly verifies if Ollama is reachable at the given endpoint.
// This is called before every mission to avoid cryptic connection errors.
func CheckOllamaReady(endpoint string) bool {
        resp, err := healthClient.Get(endpoint + "/api/version")
        if err != nil {
                return false
        }
        resp.Body.Close()
        return resp.StatusCode == http.StatusOK
}

// getMarkerPath returns the absolute path for the first-run marker file,
// using the OS-specific user config directory.
func getMarkerPath() (string, error) {
        configDir, err := os.UserConfigDir()
        if err != nil {
                return "", fmt.Errorf("failed to determine user config dir: %w", err)
        }
        markerDir := filepath.Join(configDir, "ghostoperator")
        if err := os.MkdirAll(markerDir, 0700); err != nil {
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
        if err := os.WriteFile(markerPath, []byte("v1.1.0-local"), 0600); err != nil {
                fmt.Fprintf(os.Stderr, "MarkSetupDone: failed to write marker file: %v\n", err)
        }
}

// resolveOllamaBinary tries to find the ollama binary in known install locations
// before falling back to PATH lookup.
func resolveOllamaBinary() (string, bool) {
        candidates := []string{}

        switch runtime.GOOS {
        case "windows":
                // Common Windows install paths for Ollama
                if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
                        candidates = append(candidates, filepath.Join(dir, "Programs", "Ollama", "ollama.exe"))
                }
                if dir := os.Getenv("PROGRAMFILES"); dir != "" {
                        candidates = append(candidates, filepath.Join(dir, "Ollama", "ollama.exe"))
                }
                if dir := os.Getenv("PROGRAMFILES(X86)"); dir != "" {
                        candidates = append(candidates, filepath.Join(dir, "Ollama", "ollama.exe"))
                }
                // Also check user home
                if home, err := os.UserHomeDir(); err == nil {
                        candidates = append(candidates, filepath.Join(home, "AppData", "Local", "Programs", "Ollama", "ollama.exe"))
                }
        case "darwin":
                candidates = append(candidates, "/usr/local/bin/ollama", "/opt/homebrew/bin/ollama", filepath.Join(os.Getenv("HOME"), ".local", "bin", "ollama"))
        case "linux":
                candidates = append(candidates, "/usr/local/bin/ollama", "/usr/bin/ollama", "/snap/bin/ollama", filepath.Join(os.Getenv("HOME"), ".local", "bin", "ollama"))
        }

        for _, p := range candidates {
                if _, err := os.Stat(p); err == nil {
                        return p, true
                }
        }

        // Fallback to PATH, but log a warning
        fmt.Fprintf(os.Stderr, "\033[1;33m[WARN]\033[0m ollama binary not found in known locations; falling back to PATH\n")
        return "ollama", false
}

// EnsureOllamaRunning attempts to start Ollama if it's not reachable.
// Returns true if Ollama is running and reachable after this call.
func EnsureOllamaRunning() bool {
        fmt.Print("  - Verificando servidor Ollama... ")
        resp, err := healthClient.Get("http://127.0.0.1:11434/api/version")
        if err == nil {
                resp.Body.Close()
                fmt.Println("\033[1;32mOK\033[0m")
                return true
        }

        ollamaBin, found := resolveOllamaBinary()
        if !found {
                fmt.Println("\033[1;31mNo instalado\033[0m")
                fmt.Fprintf(os.Stderr, "  \033[1;33m[INFO]\033[0m No se encontró Ollama en el sistema.\n")
                return false
        }

        fmt.Printf("\033[1;33mIniciando Ollama (%s)...\033[0m ", filepath.Base(ollamaBin))
        var cmd *exec.Cmd
        if runtime.GOOS == "windows" {
                // Use cmd /C start /B to launch Ollama in background without a new window
                cmd = exec.Command("cmd", "/C", "start", "/B", "", ollamaBin, "serve")
        } else {
                cmd = exec.Command(ollamaBin, "serve")
        }

        // On Unix, detach the child process into its own process group
        // so it doesn't become a zombie when the parent exits.
        setProcessGroup(cmd)

        if err := cmd.Start(); err != nil {
                fmt.Println("\033[1;31mError\033[0m")
                fmt.Fprintf(os.Stderr, "  \033[1;33m[ERROR]\033[0m No se pudo iniciar Ollama: %v\n", err)
                return false
        }

        // Parent no longer needs to track this detached process
        if cmd.Process != nil {
                cmd.Process.Release()
        }

        // Wait for server to wake up (up to 30 seconds with better polling)
        fmt.Print("\033[1;33mEsperando...\033[0m ")
        for i := 0; i < 15; i++ {
                time.Sleep(2 * time.Second)
                healthResp, err := healthClient.Get("http://127.0.0.1:11434/api/version")
                if err == nil {
                        healthResp.Body.Close()
                        fmt.Println("\033[1;32mOK\033[0m")
                        return true
                }
        }

        fmt.Println("\033[1;31mNo responde\033[0m")
        fmt.Fprintf(os.Stderr, "  \033[1;33m[WARN]\033[0m Ollama se inició pero no responde en el puerto 11434.\n")
        return false
}

// EnsureModel pulls the required moondream model if missing.
func EnsureModel(modelName string) {
        fmt.Printf("  - Verificando modelo %s... ", modelName)
        ollamaBin, _ := resolveOllamaBinary()
        cmd := exec.Command(ollamaBin, "pull", modelName)
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        if err := cmd.Run(); err != nil {
                fmt.Printf("\033[1;31mError al descargar %s\033[0m\n", modelName)
        } else {
                fmt.Println("\033[1;32mListo\033[0m")
        }
}
