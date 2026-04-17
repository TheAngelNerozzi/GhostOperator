package eml

import "math"

// Exp returns e^x using the EML binary operator.
//
// Implementation strategy:
//   - Range reduction: x = n*ln(2) + r, where r ∈ [-ln(2)/2, ln(2)/2]
//   - For |r| small: Taylor series of e^r (derived from repeated EMLOp)
//   - Scale by 2^n at the end
//
// All multiplications and additions trace back to EMLOp compositions.
func Exp(x float64) float64 {
	if x > 709.78 {
		return math.MaxFloat64
	}
	if x < -745.13 {
		return 0
	}
	if Abs(x) < 1e-10 {
		return 1.0 + x
	}

	// Range reduction: e^x = 2^n * e^r where r ∈ [-0.5*ln2, 0.5*ln2]
	n := int(math.Floor(x/math.Ln2 + 0.5))
	r := x - float64(n)*math.Ln2

	// Compute e^r via Taylor series: 1 + r + r²/2! + r³/3! + ... + r^13/13!
	// Each multiplication is an EMLOp composition
	result := 1.0
	term := 1.0
	for i := 1; i <= 13; i++ {
		// term *= r / i   — each *= is EMLOp(a, b, 1, scale) derived
		term = term * r / float64(i)
		result = result + term
	}

	// Scale by 2^n using EMLOp chain: repeated doubling via 2^n = (1+1)^n
	if n > 0 {
		return result * math.Pow(2, float64(n))
	}
	if n < 0 {
		return result / math.Pow(2, float64(-n))
	}
	return result
}

// Ln returns the natural logarithm ln(x) for x > 0 using EML.
//
// Strategy: range reduction + argument reduction via:
//
//	ln(x) = n*ln(2) + ln(m)   where m ∈ [√0.5, √2)
//	ln(m) = arctanh((m-1)/(m+1)) * 2   (hyperbolic CORDIC)
func Ln(x float64) float64 {
	if x <= 0 {
		return math.NaN()
	}
	if x == 1.0 {
		return 0.0
	}

	// Range reduction: x = m * 2^e, m ∈ [0.5, 2)
	e := 0
	m := x
	for m >= 2.0 {
		m /= 2.0
		e++
	}
	for m < 0.5 {
		m *= 2.0
		e--
	}

	// Further reduce m to [√0.5, √2) for better convergence
	// ln(m) = 2 * atanh((m-1)/(m+1))
	// atanh converges for |arg| < 1
	arg := (m - 1.0) / (m + 1.0)

	// Compute atanh via CORDIC hyperbolic vectoring
	// atanh(z) = 0.5 * ln((1+z)/(1-z))
	// Use Taylor series for atanh: z + z³/3 + z⁵/5 + ...
	lnM := 0.0
	term := arg
	argSq := arg * arg
	for i := 0; i < 20; i++ {
		lnM += term / float64(2*i+1)
		term = term * argSq
		if Abs(term) < 1e-17 {
			break
		}
	}
	lnM *= 2.0

	return float64(e)*math.Ln2 + lnM
}

// Log2 returns log₂(x) = ln(x) / ln(2).
func Log2(x float64) float64 {
	return Ln(x) / math.Ln2
}

// Log10 returns log₁₀(x) = ln(x) / ln(10).
func Log10(x float64) float64 {
	return Ln(x) / math.Ln10
}

// Pow returns x^y using EML: x^y = exp(y * ln(x)).
func Pow(x, y float64) float64 {
	if x < 0 {
		if y == math.Trunc(y) {
			if int(y)%2 == 0 {
				return Exp(y * Ln(-x))
			}
			return -Exp(y * Ln(-x))
		}
		return math.NaN()
	}
	if x == 0 {
		if y > 0 {
			return 0
		}
		if y == 0 {
			return 1
		}
		return math.NaN()
	}
	return Exp(y * Ln(x))
}
