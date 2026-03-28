package core

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const firstRunMarker = ".ghost_setup_done"

// IsFirstRun returns true if the setup marker file doesn't exist.
func IsFirstRun() bool {
	_, err := os.Stat(firstRunMarker)
	return os.IsNotExist(err)
}

// MarkSetupDone creates the marker file so we don't prompt again.
func MarkSetupDone() {
	os.WriteFile(firstRunMarker, []byte("v2.0"), 0644)
}

// EnsureOllamaRunning checks connectivity and starts Ollama if needed.
func EnsureOllamaRunning() bool {
	// Quick check: is Ollama API reachable?
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get("http://127.0.0.1:11434/api/version")
	if err == nil {
		return true // Already running
	}

	// Try to start it
	fmt.Println("\033[1;33m[BOOT]\033[0m Iniciando Ollama...")
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		ollamaPath := localAppData + `\Programs\Ollama\ollama.exe`
		if _, err := os.Stat(ollamaPath); err == nil {
			cmd = exec.Command(ollamaPath, "serve")
		} else {
			cmd = exec.Command("ollama", "serve")
		}
	} else {
		cmd = exec.Command("ollama", "serve")
	}

	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		fmt.Printf("\033[1;31m[ERR]\033[0m No se pudo iniciar Ollama: %v\n", err)
		fmt.Println("      Descargalo de: https://ollama.com/download")
		return false
	}

	// Wait for it to be ready
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		_, err := client.Get("http://127.0.0.1:11434/api/version")
		if err == nil {
			fmt.Println("\033[1;32m[OK]\033[0m Ollama listo.")
			return true
		}
	}

	fmt.Println("\033[1;31m[ERR]\033[0m Ollama no respondio a tiempo.")
	return false
}

// EnsureModel checks if a model is available and optionally pulls it interactively.
func EnsureModel(model string) bool {
	localAppData := os.Getenv("LOCALAPPDATA")
	ollamaExe := localAppData + `\Programs\Ollama\ollama.exe`
	if _, err := os.Stat(ollamaExe); err != nil {
		ollamaExe = "ollama"
	}

	// Check if model is already present
	out, err := exec.Command(ollamaExe, "list").Output()
	if err == nil {
		if strings.Contains(string(out), model) {
			return true
		}
	}

	// Ask user
	fmt.Printf("\n\033[1;37m  GhostOperator necesita el modelo '%s' para ver tu pantalla.\033[0m\n", model)
	fmt.Println("  Es un modelo de vision ligero (≈900MB) que corre 100% local.")
	fmt.Printf("\n  ¿Instalar %s ahora? [s/n]: ", model)

	var respuesta string
	fmt.Scanln(&respuesta)

	if respuesta == "s" || respuesta == "S" {
		fmt.Printf("\033[1;36m[>>]\033[0m Descargando %s...\n", model)
		cmd := exec.Command(ollamaExe, "pull", model)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("\033[1;31m[ERR]\033[0m Error: %v\n", err)
			return false
		}
		fmt.Println("\033[1;32m[OK]\033[0m Modelo instalado exitosamente.")
		return true
	}

	fmt.Println("\033[1;33m[--]\033[0m Sin modelo, Ghost no podra razonar visualmente.")
	return false
}
