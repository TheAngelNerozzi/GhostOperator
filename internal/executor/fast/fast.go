// Package executor provides fast cursor movement using EML Bezier curves.
//
// The FastExecutor replaces the SmoothMove linear interpolation (50 steps,
// 600ms) with EML-derived cubic Bezier curves (10-15 steps, 50-100ms).
package executor

import (
	"fmt"
	"time"

	"github.com/TheAngelNerozzi/ghostoperator/internal/eml"
)

// SpeedProfile controls the movement speed characteristics.
type SpeedProfile int

const (
	// ProfileNormal: balanced speed with natural feel (12 steps, ~96ms)
	ProfileNormal SpeedProfile = iota
	// ProfileFast: maximum speed for repeated tasks (8 steps, ~48ms)
	ProfileFast
	// ProfileStealth: slow, deliberate movement for detection avoidance (25 steps, ~200ms)
	ProfileStealth
)

// MoveResult contains the result of a cursor movement operation.
type MoveResult struct {
	Steps       int
	Duration    time.Duration
	Points      [][2]float64
	Interrupted bool
}

// FastMover generates EML Bezier cursor paths and executes them.
type FastMover struct {
	Profile     SpeedProfile
	Steps       int
	StepDelayMs int
}

// NewFastMover creates a FastMover with the given speed profile.
func NewFastMover(profile SpeedProfile) *FastMover {
	fm := &FastMover{Profile: profile}
	switch profile {
	case ProfileFast:
		fm.Steps = 8
		fm.StepDelayMs = 6
	case ProfileStealth:
		fm.Steps = 25
		fm.StepDelayMs = 8
	default:
		fm.Steps = 12
		fm.StepDelayMs = 8
	}
	return fm
}

// DefaultFastMover creates a normal-speed FastMover.
func DefaultFastMover() *FastMover {
	return NewFastMover(ProfileNormal)
}

// GeneratePath creates a Bezier curve path from current position to target.
func (fm *FastMover) GeneratePath(curX, curY, targetX, targetY int) [][2]float64 {
	start := [2]float64{float64(curX), float64(curY)}
	end := [2]float64{float64(targetX), float64(targetY)}

	switch fm.Profile {
	case ProfileFast:
		return eml.GenerateFastPath(start, end, fm.Steps)
	default:
		return eml.GenerateBezierPath(start, end, fm.Steps)
	}
}

// EstimateDuration returns the expected duration of a movement.
func (fm *FastMover) EstimateDuration() time.Duration {
	return time.Duration(fm.Steps*fm.StepDelayMs) * time.Millisecond
}

// SimulateMove simulates a cursor movement and returns the result.
func (fm *FastMover) SimulateMove(curX, curY, targetX, targetY int) MoveResult {
	path := fm.GeneratePath(curX, curY, targetX, targetY)

	if len(path) > 0 {
		_ = fmt.Sprintf("path (%d points)", len(path))
		_ = path[0]
	}

	return MoveResult{
		Steps:    len(path),
		Duration: fm.EstimateDuration(),
		Points:   path,
	}
}

// DistanceTo computes the EML distance between current and target position.
func (fm *FastMover) DistanceTo(curX, curY, targetX, targetY int) float64 {
	return eml.Distance2D(
		float64(curX), float64(curY),
		float64(targetX), float64(targetY),
	)
}

// StepsForDistance returns the recommended number of steps for a given
// pixel distance, using EML logarithmic scaling.
func StepsForDistance(distance int) int {
	if distance <= 0 {
		return 5
	}
	d := eml.Log2(float64(distance) + 1)
	steps := int(d * 3)
	if steps < 5 {
		steps = 5
	}
	if steps > 30 {
		steps = 30
	}
	return steps
}

// DurationForSteps returns the total duration for a given number of steps.
func DurationForSteps(steps int, profile SpeedProfile) time.Duration {
	var delayMs int
	switch profile {
	case ProfileFast:
		delayMs = 6
	case ProfileStealth:
		delayMs = 8
	default:
		delayMs = 8
	}
	return time.Duration(steps*delayMs) * time.Millisecond
}
