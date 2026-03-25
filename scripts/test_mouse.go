package main

import (
	"fmt"
	"time"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/action"
)

func main() {
	fmt.Println("🖱️ Iniciando Prueba de Movimiento Ghost (1 segundo)...")
	executor := &action.ActionExecutor{}
	
	// Mover de 0,0 a 500,500 en la escala 0-1000
	mockParams := map[string]interface{}{
		"x": 500.0,
		"y": 500.0,
	}
	
	fmt.Println("🚀 Moviendo mouse suavemente hacia el centro...")
	executor.Execute(action.Command{Type: "CLICK", Params: mockParams})
	
	fmt.Println("✅ Prueba finalizada.")
}
