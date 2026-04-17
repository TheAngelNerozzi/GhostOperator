package executor

import (
        "testing"
        "time"

        "github.com/TheAngelNerozzi/ghostoperator/internal/eml"
)

func TestFastMover_PathLength(t *testing.T) {
        fm := NewFastMover(ProfileNormal)
        path := fm.GeneratePath(0, 0, 500, 500)
        if len(path) != 13 { // Steps + 1
                t.Errorf("Expected 13 points, got %d", len(path))
        }
}

func TestFastMover_PathEndpoints(t *testing.T) {
        fm := NewFastMover(ProfileNormal)
        path := fm.GeneratePath(100, 200, 400, 300)

        if len(path) < 2 {
                t.Fatal("Path too short")
        }

        // First point should be near start
        if path[0][0] < 90 || path[0][0] > 110 {
                t.Errorf("Start X: %v, expected ~100", path[0][0])
        }

        // Last point should be near end
        if path[len(path)-1][0] < 390 || path[len(path)-1][0] > 410 {
                t.Errorf("End X: %v, expected ~400", path[len(path)-1][0])
        }
}

func TestFastMover_PathIsSmooth(t *testing.T) {
        fm := NewFastMover(ProfileNormal)
        path := fm.GeneratePath(0, 0, 500, 500)

        // Check that consecutive points don't have huge jumps
        for i := 1; i < len(path); i++ {
                dx := path[i][0] - path[i-1][0]
                dy := path[i][1] - path[i-1][1]
                dist := eml.Sqrt(dx*dx + dy*dy)
                maxStep := 200.0 // reasonable max step for 500px travel
                if dist > maxStep {
                        t.Errorf("Step %d: distance %v exceeds max %v", i, dist, maxStep)
                }
        }
}

func TestFastMover_SimulateMove(t *testing.T) {
        fm := NewFastMover(ProfileNormal)
        result := fm.SimulateMove(0, 0, 300, 400)

        if result.Interrupted {
                t.Error("Move should not be interrupted")
        }
        if result.Duration <= 0 {
                t.Error("Duration should be positive")
        }
        if result.Steps == 0 {
                t.Error("Should have steps")
        }
}

func TestFastMover_Distance(t *testing.T) {
        fm := NewFastMover(ProfileNormal)
        d := fm.DistanceTo(0, 0, 3, 4)
        if d < 4.9 || d > 5.1 {
                t.Errorf("Distance: %v, expected ~5", d)
        }
}

func TestFastMover_EstimateDuration(t *testing.T) {
        fm := NewFastMover(ProfileNormal)
        d := fm.EstimateDuration()
        // 12 steps * 8ms = 96ms
        if d < 90*time.Millisecond || d > 100*time.Millisecond {
                t.Errorf("Duration: %v, expected ~96ms", d)
        }
}

func TestFastMover_ProfileFast(t *testing.T) {
        fm := NewFastMover(ProfileFast)
        if fm.Steps != 8 {
                t.Errorf("Fast steps: %d, expected 8", fm.Steps)
        }
        d := fm.EstimateDuration()
        if d > 55*time.Millisecond {
                t.Errorf("Fast duration: %v, expected ~48ms", d)
        }
}

func TestFastMover_ProfileStealth(t *testing.T) {
        fm := NewFastMover(ProfileStealth)
        if fm.Steps != 25 {
                t.Errorf("Stealth steps: %d, expected 25", fm.Steps)
        }
}

func TestStepsForDistance(t *testing.T) {
        tests := []struct {
                dist int
                min  int
                max  int
        }{
                {0, 5, 5},
                {10, 5, 15},
                {100, 10, 25},
                {1000, 20, 30},
        }
        for _, tt := range tests {
                s := StepsForDistance(tt.dist)
                if s < tt.min || s > tt.max {
                        t.Errorf("StepsForDistance(%d) = %d, expected [%d,%d]", tt.dist, s, tt.min, tt.max)
                }
        }
}
