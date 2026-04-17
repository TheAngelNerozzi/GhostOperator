package automation

import (
        "encoding/json"
        "fmt"
        "sync"
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
        mu             sync.Mutex
        IsRunning      bool
        LastTargetX    int32
        LastTargetY    int32
        hasLastTarget  bool
        Interrupted    bool
        OnInterruption func() // Callback for the UI/CLI
}

// SetLastTarget safely sets the last target coordinates.
func (e *ActionExecutor) SetLastTarget(x, y int32) {
        e.mu.Lock()
        defer e.mu.Unlock()
        e.LastTargetX = x
        e.LastTargetY = y
        e.hasLastTarget = true
}

// GetLastTarget safely returns the last target coordinates.
func (e *ActionExecutor) GetLastTarget() (int32, int32) {
        e.mu.Lock()
        defer e.mu.Unlock()
        return e.LastTargetX, e.LastTargetY
}

// GetLastTargetWithFlag returns whether a last target exists and its coordinates.
func (e *ActionExecutor) GetLastTargetWithFlag() (bool, int32, int32) {
        e.mu.Lock()
        defer e.mu.Unlock()
        return e.hasLastTarget, e.LastTargetX, e.LastTargetY
}

// SetInterrupted safely sets the interrupted flag.
func (e *ActionExecutor) SetInterrupted(v bool) {
        e.mu.Lock()
        defer e.mu.Unlock()
        e.Interrupted = v
}

// GetInterrupted safely returns the interrupted flag.
func (e *ActionExecutor) GetInterrupted() bool {
        e.mu.Lock()
        defer e.mu.Unlock()
        return e.Interrupted
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

        e.mu.Lock()
        if e.IsRunning {
                e.mu.Unlock()
                return ActionResult{Status: "error", Message: "executor already busy", Action: cmd.Type}
        }
        e.IsRunning = true
        e.mu.Unlock()
        defer func() {
                e.mu.Lock()
                e.IsRunning = false
                e.mu.Unlock()
        }()

        // Safety check before dispatching any command
        if err := e.CheckSafety(); err != nil {
                return ActionResult{Status: "safety_violation", Message: err.Error(), Action: cmd.Type}
        }

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
        if ms <= 0 || ms > 60000 {
                return ActionResult{Status: "error", Message: "ms must be between 0 and 60000", Action: "WAIT"}
        }
        remaining := time.Duration(ms) * time.Millisecond
        for remaining > 0 {
                if e.GetInterrupted() {
                        return ActionResult{Status: "safety_violation", Message: "wait interrupted", Action: "WAIT"}
                }
                if err := e.CheckSafety(); err != nil {
                        return ActionResult{Status: "safety_violation", Message: err.Error(), Action: "WAIT"}
                }
                chunk := 50 * time.Millisecond
                if chunk > remaining {
                        chunk = remaining
                }
                time.Sleep(chunk)
                remaining -= chunk
        }
        return ActionResult{Status: "success", Action: "WAIT"}
}

// Handlers for CLICK and TYPE will be implemented in OS-specific files
// using syscalls to keep the binary CGO-free.
