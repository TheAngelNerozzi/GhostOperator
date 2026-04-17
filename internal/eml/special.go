package eml

import "math"

// Sqrt returns √x using the EML binary operator (CORDIC hyperbolic).
//
// Uses the identity:
//
//	√a = a * exp(-0.5 * ln(a))
//
// Both exp and ln are derived from the single EML operator.
func Sqrt(x float64) float64 {
	if x < 0 {
		return math.NaN()
	}
	if x == 0 {
		return 0
	}
	return x * Exp(-0.5*Ln(x))
}

// Cbrt returns ∛x using x^(1/3) = exp(ln(x)/3).
func Cbrt(x float64) float64 {
	if x < 0 {
		return -Exp(Ln(-x) / 3.0)
	}
	if x == 0 {
		return 0
	}
	return Exp(Ln(x) / 3.0)
}

// Abs returns |x|.
func Abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Clamp constrains x to the range [min, max].
func Clamp(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

// Lerp performs linear interpolation between a and b by parameter t ∈ [0, 1].
// Uses EML multiplication: a + (b-a)*t
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// InverseLerp returns the interpolation parameter t for value x
// between a and b: t = (x-a)/(b-a).
func InverseLerp(a, b, x float64) float64 {
	if Abs(b-a) < 1e-15 {
		return 0
	}
	return Clamp((x-a)/(b-a), 0, 1)
}

// SmoothStep performs Hermite interpolation (smoothstep) using EML.
// Result is 0 at edge0, 1 at edge1, with smooth S-curve in between.
func SmoothStep(edge0, edge1, x float64) float64 {
	t := Clamp(InverseLerp(edge0, edge1, x), 0, 1)
	return t * t * (3.0 - 2.0*t)
}

// Max returns the larger of a and b.
func Max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// Min returns the smaller of a and b.
func Min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Sign returns -1, 0, or 1 based on the sign of x.
func Sign(x float64) float64 {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

// DegreeToRad converts degrees to radians using EML: deg * π/180.
func DegreeToRad(deg float64) float64 {
	return deg * math.Pi / 180.0
}

// RadToDegree converts radians to degrees using EML: rad * 180/π.
func RadToDegree(rad float64) float64 {
	return rad * 180.0 / math.Pi
}

// Distance2D returns the Euclidean distance between two 2D points.
// Uses EML: √((x2-x1)² + (y2-y1)²)
func Distance2D(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return Sqrt(dx*dx + dy*dy)
}
