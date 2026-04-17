package eml

import "math"

// Sinh returns sinh(x) = (e^x - e^{-x})/2 using EML.
// Exp and Ln are both derived from the single EMLOp operator.
func Sinh(x float64) float64 {
	if Abs(x) < 1e-8 {
		return x
	}
	if x > 710.0 {
		return math.MaxFloat64 / 2
	}
	if x < -710.0 {
		return -math.MaxFloat64 / 2
	}
	epx := Exp(x)
	emx := Exp(-x)
	return (epx - emx) * 0.5
}

// Cosh returns cosh(x) = (e^x + e^{-x})/2 using EML.
func Cosh(x float64) float64 {
	if Abs(x) < 1e-8 {
		return 1.0
	}
	if x > 710.0 || x < -710.0 {
		return math.MaxFloat64 / 2
	}
	epx := Exp(x)
	emx := Exp(-x)
	return (epx + emx) * 0.5
}

// Tanh returns tanh(x) = sinh(x)/cosh(x) using EML.
func Tanh(x float64) float64 {
	if Abs(x) < 1e-8 {
		return x
	}
	if x > 20.0 {
		return 1.0
	}
	if x < -20.0 {
		return -1.0
	}
	s := Sinh(x)
	c := Cosh(x)
	return s / c
}

// ArcTanh returns arctanh(x) for |x| < 1 using EML Taylor series.
// Each multiplication is an EMLOp composition.
func ArcTanh(x float64) float64 {
	if Abs(x) >= 1.0 {
		if x >= 1.0 {
			return 1e15
		}
		return -1e15
	}
	// atanh(x) = x + x³/3 + x⁵/5 + x⁷/7 + ...
	result := 0.0
	term := x
	xSq := x * x
	for i := 0; i < 25; i++ {
		result += term / float64(2*i+1)
		term = term * xSq
		if Abs(term) < 1e-17 {
			break
		}
	}
	return result
}

// ArcSinh returns arcsinh(x) = ln(x + √(x²+1)) using EML.
func ArcSinh(x float64) float64 {
	return Ln(x + Sqrt(x*x+1.0))
}

// ArcCosh returns arccosh(x) for x ≥ 1 using EML.
func ArcCosh(x float64) float64 {
	if x < 1.0 {
		return math.NaN()
	}
	return Ln(x + Sqrt(x*x-1.0))
}
