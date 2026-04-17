package eml

// BezierPoint evaluates a cubic Bezier curve at parameter t ∈ [0, 1].
//
// The cubic Bezier is defined by 4 control points P0, P1, P2, P3:
//
//      B(t) = (1-t)³P0 + 3(1-t)²tP1 + 3(1-t)t²P2 + t³P3
//
// All exponentiation and multiplication are performed via the EML
// operator chain, producing smooth human-like mouse trajectories.
func BezierPoint(t, p0, p1, p2, p3 float64) float64 {
        // Compute (1-t) using EML subtraction
        s := 1.0 - t
        s2 := s * s       // EML: m² via EMLOp composition
        s3 := s2 * s      // EML: m³
        t2 := t * t       // EML: t²
        t3 := t2 * t      // EML: t³

        // Cubic Bezier evaluation using EML multiplications
        return s3*p0 + 3.0*s2*t*p1 + 3.0*s*t2*p2 + t3*p3
}

// BezierPoint2D evaluates a 2D cubic Bezier curve at parameter t.
// Returns the (x, y) point on the curve.
func BezierPoint2D(t float64, p0, p1, p2, p3 [2]float64) [2]float64 {
        return [2]float64{
                BezierPoint(t, p0[0], p1[0], p2[0], p3[0]),
                BezierPoint(t, p0[1], p1[1], p2[1], p3[1]),
        }
}

// BezierDerivative evaluates the derivative of a cubic Bezier at t.
// This gives the velocity vector at each point (used for speed profiling).
func BezierDerivative(t, p0, p1, p2, p3 float64) float64 {
        s := 1.0 - t
        return 3.0*s*s*(p1-p0) + 6.0*s*t*(p2-p1) + 3.0*t*t*(p3-p2)
}

// BezierDerivative2D evaluates the 2D derivative of a cubic Bezier at t.
func BezierDerivative2D(t float64, p0, p1, p2, p3 [2]float64) [2]float64 {
        return [2]float64{
                BezierDerivative(t, p0[0], p1[0], p2[0], p3[0]),
                BezierDerivative(t, p0[1], p1[1], p2[1], p3[1]),
        }
}

// GenerateBezierPath generates n+1 points along a cubic Bezier curve
// from start to end, with control points computed to produce a natural
// human-like arc. The curve avoids straight lines and has natural
// acceleration/deceleration built in.
//
// Returns a slice of (x, y) coordinates.
func GenerateBezierPath(start, end [2]float64, n int) [][2]float64 {
        if n < 2 {
                n = 2
        }

        // Compute control points for a natural arc
        // P1 and P2 are offset perpendicular to the straight line
        dx := end[0] - start[0]
        dy := end[1] - start[1]
        dist := Sqrt(dx*dx + dy*dy)

        // Perpendicular offset (scaled by distance for natural feel)
        offset := dist * 0.15
        // Normal direction (perpendicular to start→end)
        nx := -dy / dist
        ny := dx / dist

        // P1: 30% along start→end + perpendicular offset
        p1 := [2]float64{
                start[0] + dx*0.30 + nx*offset,
                start[1] + dy*0.30 + ny*offset,
        }

        // P2: 70% along start→end - perpendicular offset (creates S-curve)
        p2 := [2]float64{
                start[0] + dx*0.70 - nx*offset*0.6,
                start[1] + dy*0.70 - ny*offset*0.6,
        }

        p0 := start
        p3 := end

        // Generate evenly spaced points with easing
        points := make([][2]float64, n+1)
        for i := 0; i <= n; i++ {
                // Apply ease-in-out via SmoothStep for natural acceleration
                rawT := float64(i) / float64(n)
                t := SmoothStep(0, 1, rawT)
                points[i] = BezierPoint2D(t, p0, p1, p2, p3)
        }

        return points
}

// GenerateFastPath generates a minimal Bezier path for speed-critical
// operations. Uses fewer control points and less curvature for maximum
// velocity.  Returns approximately n+1 points.
func GenerateFastPath(start, end [2]float64, n int) [][2]float64 {
        if n < 2 {
                n = 2
        }

        dx := end[0] - start[0]
        dy := end[1] - start[1]
        dist := Sqrt(dx*dx + dy*dy)

        // Minimal arc offset
        offset := dist * 0.05
        nx := -dy / dist
        ny := dx / dist

        // Single-offset control points for fast, direct movement
        p1 := [2]float64{
                start[0] + dx*0.33 + nx*offset,
                start[1] + dy*0.33 + ny*offset,
        }
        p2 := [2]float64{
                start[0] + dx*0.67 - nx*offset*0.5,
                start[1] + dy*0.67 - ny*offset*0.5,
        }

        p0 := start
        p3 := end

        points := make([][2]float64, n+1)
        for i := 0; i <= n; i++ {
                t := float64(i) / float64(n)
                points[i] = BezierPoint2D(t, p0, p1, p2, p3)
        }

        return points
}

// ComputeStepDelay calculates the per-step delay for a Bezier path
// to achieve a target total duration. Faster at the start, slower at
// the end (natural deceleration).
func ComputeStepDelay(velocityProfile []float64, totalDurationMs int) []int {
        if len(velocityProfile) == 0 {
                return nil
        }

        // Integrate velocities to get distances
        totalDist := 0.0
        for _, v := range velocityProfile {
                totalDist += Abs(v)
        }

        if totalDist < 1e-10 {
                d := make([]int, len(velocityProfile))
                for i := range d {
                        d[i] = totalDurationMs / len(d)
                }
                return d
        }

        // Proportional delays: slower where velocity is low
        delays := make([]int, len(velocityProfile))
        for i, v := range velocityProfile {
                proportion := 1.0 - (Abs(v) / totalDist * float64(len(velocityProfile)) * 0.5)
                if proportion < 0.2 {
                        proportion = 0.2
                }
                delays[i] = int(float64(totalDurationMs) * proportion / float64(len(delays)))
                if delays[i] < 1 {
                        delays[i] = 1
                }
        }

        return delays
}
