package core

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
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
	os.WriteFile(firstRunMarker, []byte("v1.1.0-local"), 0644)
}

// EnsureOllamaRunning attempts to start Ollama if it's not reachable.
func EnsureOllamaRunning() bool {
	fmt.Print("  - Verificando servidor Ollama... ")
	client := http.Client{Timeout: 2 * time.Second}
	_, err := client.Get("http://127.0.0.1:11434/api/version")
	if err == nil {
		fmt.Println("\033[1;32mOK\033[0m")
		return true
	}

	fmt.Print("\033[1;33mIntentando iniciar...\033[0m ")
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "start", "ollama", "serve")
	} else {
		cmd = exec.Command("ollama", "serve")
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("\033[1;31mError\033[0m")
		return false
	}

	// Wait for server to wake up
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)
		_, err := client.Get("http://127.0.0.1:11434/api/version")
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
	cmd := exec.Command("ollama", "pull", modelName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("\033[1;31mError al descargar %s\033[0m\n", modelName)
	} else {
		fmt.Println("\033[1;32mListo\033[0m")
	}
}
