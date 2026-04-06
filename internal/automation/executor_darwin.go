//go:build darwin
package automation

import "fmt"

func (e *ActionExecutor) handleClick(params map[string]interface{}) ActionResult {
	x, _ := params["x"].(float64)
	y, _ := params["y"].(float64)
	fmt.Printf("[macOS] Click at %.2f, %.2f\n", x, y)
	return ActionResult{Status: "success", Action: "CLICK"}
}

func (e *ActionExecutor) handleDoubleClick(params map[string]interface{}) ActionResult {
	x, _ := params["x"].(float64)
	y, _ := params["y"].(float64)
	fmt.Printf("[macOS] Double-click at %.2f, %.2f\n", x, y)
	return ActionResult{Status: "success", Action: "DOUBLE_CLICK"}
}

func (e *ActionExecutor) handleType(params map[string]interface{}) ActionResult {
	text, _ := params["text"].(string)
	fmt.Printf("[macOS] Typing: %s\n", text)
	return ActionResult{Status: "success", Action: "TYPE"}
}

func (e *ActionExecutor) SmoothMove(targetX, targetY int32) {
	fmt.Printf("[macOS] Moving mouse to %d, %d\n", targetX, targetY)
	e.LastTargetX = targetX
	e.LastTargetY = targetY
}
