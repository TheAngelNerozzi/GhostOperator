package input

import (
	"fmt"
	"time"

	"github.com/robotn/hotkey"
)

// ListenForHotkey starts a listener for Alt+Space and calls the callback when triggered.
func ListenForHotkey(callback func()) {
	// Note: robotn/hotkey is cross-platform.
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModAlt}, hotkey.KeySpace)
	
	err := hk.Register()
	if err != nil {
		fmt.Printf("Failed to register hotkey: %v\n", err)
		return
	}

	fmt.Println("Hotkey registered: Alt+Space")
	
	// Start listening in a loop
	go func() {
		for {
			<-hk.Listen()
			callback()
			// Debounce
			time.Sleep(500 * time.Millisecond)
		}
	}()
}
