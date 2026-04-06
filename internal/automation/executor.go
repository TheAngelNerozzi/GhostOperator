package automation

import (
	"encoding/json"
	"fmt"
	"time"
)

// Command represents a single action to be executed by GO.
type Command struct {
	Type   string                 `json:"command"`
	Params map[string]interface{} `json:"params"`
}

// ActionResult represents the feedback from an action.
type ActionResult struct {
	Status   string      `json:"status"`
	Message  string      `json:"message,omitempty"`
	Action   string      `json:"action"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// ActionExecutor handles the execution of commands and the safety loop.
type ActionExecutor struct {
	IsRunning      bool
	LastTargetX    int32
	LastTargetY    int32
	Interrupted    bool
	OnInterruption func() // Callback for the UI/CLI
}

// ParseCommand converts a JSON string into a Command struct.
func (e *ActionExecutor) ParseCommand(jsonData string) (Command, error) {
	var cmd Command
	err := json.Unmarshal([]byte(jsonData), &cmd)
	return cmd, err
}

// Execute handles command routing
func (e *ActionExecutor) Execute(cmd Command) ActionResult {
	fmt.Printf("Executing Action: %s with params %v\n", cmd.Type, cmd.Params)

	switch cmd.Type {
	case "CLICK":
		return e.handleClick(cmd.Params)
	case "DOUBLE_CLICK":
		return e.handleDoubleClick(cmd.Params)
	case "TYPE":
		return e.handleType(cmd.Params)
	case "WAIT":
		return e.handleWait(cmd.Params)
	default:
		return ActionResult{Status: "error", Message: "Unknown command type", Action: cmd.Type}
	}
}

func (e *ActionExecutor) handleWait(params map[string]interface{}) ActionResult {
	ms, ok := params["ms"].(float64)
	if !ok {
		return ActionResult{Status: "error", Message: "Missing ms parameter", Action: "WAIT"}
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return ActionResult{Status: "success", Action: "WAIT"}
}

// Handlers for CLICK and TYPE will be implemented in OS-specific files
// using syscalls to keep the binary CGO-free.
