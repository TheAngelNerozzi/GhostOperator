package eml

import "math"

// Sin returns sin(θ) using the EML binary operator (CORDIC circular rotation).
//
// Range reduction uses quadrant analysis to map θ to [0, π/2],
// within the CORDIC convergence range of ~1.74 radians.
func Sin(theta float64) float64 {
        twoPi := 2.0 * math.Pi
        theta = math.Mod(theta, twoPi)
        if theta < 0 {
                theta += twoPi
        }

        var sign float64 = 1.0
        switch {
        case theta <= math.Pi/2:
                // Q1: direct
        case theta <= math.Pi:
                // Q2: sin(θ) = sin(π - θ)
                theta = math.Pi - theta
        case theta <= 3*math.Pi/2:
                // Q3: sin(θ) = -sin(θ - π)
                theta = theta - math.Pi
                sign = -1.0
        default:
                // Q4: sin(θ) = -sin(2π - θ)
                theta = twoPi - theta
                sign = -1.0
        }

        K := precomputedTables.gainCircular
        _, y := cordicRotateCircular(1.0/K, 0.0, theta)
        return sign * y
}

// Cos returns cos(θ) = sin(θ + π/2) using EML.
func Cos(theta float64) float64 {
        return Sin(theta + math.Pi/2)
}

// Tan returns tan(θ) = sin(θ)/cos(θ) using EML.
func Tan(theta float64) float64 {
        s := Sin(theta)
        c := Cos(theta)
        if Abs(c) < 1e-15 {
                if s >= 0 {
                        return 1e15
                }
                return -1e15
        }
        return s / c
}

// ArcTan returns arctan(y/x) in radians using EML vectoring (CORDIC).
// Handles all four quadrants correctly.
func ArcTan(y, x float64) float64 {
        if x == 0 && y == 0 {
                return 0
        }
        if x > 0 && y == 0 {
                return 0
        }
        if x < 0 && y == 0 {
                return math.Pi
        }
        if y > 0 && x == 0 {
                return math.Pi / 2
        }
        if y < 0 && x == 0 {
                return -math.Pi / 2
        }

        var angle float64
        absY, absX := Abs(y), Abs(x)

        if absY <= absX {
                // |y/x| <= 1: direct CORDIC on (|x|, y)
                _, rawAngle := cordicVectorCircular(absX, y)
                angle = -rawAngle
        } else {
                // |y/x| > 1: use arctan(y/x) = sign(y)*π/2 - arctan(x/y)
                _, rawAngle := cordicVectorCircular(absY, x)
                baseAngle := -rawAngle
                if y > 0 {
                        angle = math.Pi/2 - baseAngle
                } else {
                        angle = -math.Pi/2 + baseAngle
                }
        }

        // Quadrant adjustment: angle = arctan(y/|x|), but original x may be negative
        if x < 0 {
                if y >= 0 {
                        angle = math.Pi - angle
                } else {
                        angle = -math.Pi - angle
                }
        }

        return angle
}

// ArcTan2 returns arctan2(y, x) in the range [-π, π].
func ArcTan2(y, x float64) float64 {
        return ArcTan(y, x)
}

// ArcSin returns arcsin(x) in radians for x ∈ [-1, 1].
func ArcSin(x float64) float64 {
        if x < -1 {
                x = -1
        }
        if x > 1 {
                x = 1
        }
        // arcsin(x) = arctan(x / √(1 - x²))
        denom := Sqrt(1 - x*x)
        if denom < 1e-15 {
                if x >= 0 {
                        return math.Pi / 2
                }
                return -math.Pi / 2
        }
        return ArcTan(x, denom)
}

// ArcCos returns arccos(x) = π/2 - arcsin(x).
func ArcCos(x float64) float64 {
        return math.Pi/2 - ArcSin(x)
}
