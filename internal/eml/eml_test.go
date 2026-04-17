package eml

import (
	"math"
	"testing"
)

// CORDIC with 32 iterations achieves ~1e-9 precision.
const trigTol = 1e-9
const expTol = 1e-9
const logTol = 1e-9

func approxTol(a, b, tol float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	return math.Abs(a-b) < tol
}

// ━━━ EML Operator Tests ━━━

func TestEMLOp_BasicRotation(t *testing.T) {
	x, y := EMLOp(1.0, 0.0, 1, 1.0)
	if !approxTol(x, 1.0, 1e-15) || !approxTol(y, 1.0, 1e-15) {
		t.Errorf("EMLOp(1,0,1,1) = (%v,%v), want (1,1)", x, y)
	}
	x, y = EMLOp(1.0, 0.0, -1, 1.0)
	if !approxTol(x, 1.0, 1e-15) || !approxTol(y, -1.0, 1e-15) {
		t.Errorf("EMLOp(1,0,-1,1) = (%v,%v), want (1,-1)", x, y)
	}
}

// ━━━ Trigonometric Tests ━━━

func TestSin_KnownValues(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 0},
		{math.Pi / 6, 0.5},
		{math.Pi / 4, math.Sqrt(2) / 2},
		{math.Pi / 3, math.Sqrt(3) / 2},
		{math.Pi / 2, 1.0},
		{math.Pi, 0},
		{-math.Pi / 2, -1.0},
	}
	for _, tt := range tests {
		got := Sin(tt.input)
		if !approxTol(got, tt.expected, trigTol) {
			t.Errorf("Sin(%v) = %v, want %v (diff=%e)", tt.input, got, tt.expected, math.Abs(got-tt.expected))
		}
	}
}

func TestCos_KnownValues(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 1.0},
		{math.Pi / 3, 0.5},
		{math.Pi / 2, 0},
		{math.Pi, -1.0},
		{2 * math.Pi, 1.0},
	}
	for _, tt := range tests {
		got := Cos(tt.input)
		if !approxTol(got, tt.expected, trigTol) {
			t.Errorf("Cos(%v) = %v, want %v (diff=%e)", tt.input, got, tt.expected, math.Abs(got-tt.expected))
		}
	}
}

func TestSin_VsMathSin(t *testing.T) {
	for angle := -3.0; angle <= 3.0; angle += 0.1 {
		want := math.Sin(angle)
		got := Sin(angle)
		relErr := math.Abs(got-want)
		if relErr > trigTol {
			t.Errorf("Sin(%v) = %v, math.Sin = %v, diff = %e", angle, got, want, relErr)
		}
	}
}

func TestCos_VsMathCos(t *testing.T) {
	for angle := -3.0; angle <= 3.0; angle += 0.1 {
		want := math.Cos(angle)
		got := Cos(angle)
		relErr := math.Abs(got-want)
		if relErr > trigTol {
			t.Errorf("Cos(%v) = %v, math.Cos = %v, diff = %e", angle, got, want, relErr)
		}
	}
}

func TestTan_KnownValues(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 0},
		{math.Pi / 4, 1.0},
	}
	for _, tt := range tests {
		got := Tan(tt.input)
		if !approxTol(got, tt.expected, trigTol) {
			t.Errorf("Tan(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestPythagoreanIdentity(t *testing.T) {
	for angle := -3.0; angle <= 3.0; angle += 0.3 {
		s2 := Sin(angle) * Sin(angle)
		c2 := Cos(angle) * Cos(angle)
		if !approxTol(s2+c2, 1.0, trigTol*2) {
			t.Errorf("sin²(%v)+cos²(%v) = %v, want 1.0", angle, angle, s2+c2)
		}
	}
}

// ━━━ Inverse Trig Tests ━━━

func TestArcTan2_Quadrants(t *testing.T) {
	tests := []struct {
		y, x     float64
		expected float64
	}{
		{1, 1, math.Pi / 4},
		{1, 0, math.Pi / 2},
		{0, -1, math.Pi},
		{-1, 0, -math.Pi / 2},
		{-1, -1, -3 * math.Pi / 4},
		{1, -1, 3 * math.Pi / 4},
	}
	for _, tt := range tests {
		got := ArcTan2(tt.y, tt.x)
		if !approxTol(got, tt.expected, trigTol) {
			t.Errorf("ArcTan2(%v,%v) = %v, want %v", tt.y, tt.x, got, tt.expected)
		}
	}
}

func TestArcSin_KnownValues(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 0},
		{1, math.Pi / 2},
		{-1, -math.Pi / 2},
		{0.5, math.Pi / 6},
	}
	for _, tt := range tests {
		got := ArcSin(tt.input)
		if !approxTol(got, tt.expected, trigTol*10) {
			t.Errorf("ArcSin(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestArcCos_KnownValues(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{1, 0},
		{0, math.Pi / 2},
		{-1, math.Pi},
	}
	for _, tt := range tests {
		got := ArcCos(tt.input)
		if !approxTol(got, tt.expected, trigTol*10) {
			t.Errorf("ArcCos(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

// ━━━ Exp / Log Tests ━━━

func TestExp_KnownValues(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 1.0},
		{1, math.E},
		{math.Ln2, 2.0},
	}
	for _, tt := range tests {
		got := Exp(tt.input)
		if !approxTol(got, tt.expected, expTol) {
			t.Errorf("Exp(%v) = %v, want %v (diff=%e)", tt.input, got, tt.expected, math.Abs(got-tt.expected))
		}
	}
}

func TestExp_VsMathExp(t *testing.T) {
	for x := -5.0; x <= 5.0; x += 0.5 {
		want := math.Exp(x)
		got := Exp(x)
		relErr := math.Abs(got-want) / math.Max(math.Abs(want), 1e-300)
		if relErr > 1e-9 {
			t.Errorf("Exp(%v) = %v, math.Exp = %v, relErr = %e", x, got, want, relErr)
		}
	}
}

func TestLn_KnownValues(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{1, 0},
		{math.E, 1.0},
		{2, math.Ln2},
	}
	for _, tt := range tests {
		got := Ln(tt.input)
		if !approxTol(got, tt.expected, logTol*10) {
			t.Errorf("Ln(%v) = %v, want %v (diff=%e)", tt.input, got, tt.expected, math.Abs(got-tt.expected))
		}
	}
}

func TestLn_Zero_Negative(t *testing.T) {
	if !math.IsNaN(Ln(0)) {
		t.Error("Ln(0) should be NaN")
	}
	if !math.IsNaN(Ln(-1)) {
		t.Error("Ln(-1) should be NaN")
	}
}

func TestExpLn_Inverse(t *testing.T) {
	for x := 0.5; x <= 10.0; x += 0.5 {
		got := Exp(Ln(x))
		if !approxTol(got, x, 1e-6) {
			t.Errorf("Exp(Ln(%v)) = %v, want %v", x, got, x)
		}
	}
}

func TestLog2_Log10(t *testing.T) {
	if !approxTol(Log2(8), 3.0, logTol) {
		t.Errorf("Log2(8) = %v, want 3", Log2(8))
	}
	if !approxTol(Log2(1024), 10.0, logTol) {
		t.Errorf("Log2(1024) = %v, want 10", Log2(1024))
	}
	if !approxTol(Log10(100), 2.0, logTol) {
		t.Errorf("Log10(100) = %v, want 2", Log10(100))
	}
	if !approxTol(Log10(1000), 3.0, logTol) {
		t.Errorf("Log10(1000) = %v, want 3", Log10(1000))
	}
}

func TestPow_KnownValues(t *testing.T) {
	tests := []struct {
		x, y, exp float64
	}{
		{2, 10, 1024},
		{3, 3, 27},
		{10, 0, 1},
		{5, -1, 0.2},
	}
	for _, tt := range tests {
		got := Pow(tt.x, tt.y)
		if !approxTol(got, tt.exp, expTol) {
			t.Errorf("Pow(%v,%v) = %v, want %v", tt.x, tt.y, got, tt.exp)
		}
	}
}

// ━━━ Hyperbolic Tests ━━━

func TestSinhCosh_Identity(t *testing.T) {
	for x := -3.0; x <= 3.0; x += 0.5 {
		s := Sinh(x)
		c := Cosh(x)
		if !approxTol(c*c-s*s, 1.0, 1e-6) {
			t.Errorf("cosh²(%v)-sinh²(%v) = %v, want 1.0", x, x, c*c-s*s)
		}
	}
}

func TestTanh_Range(t *testing.T) {
	for x := -10.0; x <= 10.0; x += 1.0 {
		got := Tanh(x)
		if got < -1.0 || got > 1.0 {
			t.Errorf("Tanh(%v) = %v, out of [-1,1]", x, got)
		}
	}
	if !approxTol(Tanh(0), 0, 1e-15) {
		t.Errorf("Tanh(0) = %v, want 0", Tanh(0))
	}
}

// ━━━ Special Functions ━━━

func TestSqrt_KnownValues(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 0},
		{1, 1},
		{4, 2},
		{9, 3},
		{2, math.Sqrt(2)},
		{0.25, 0.5},
	}
	for _, tt := range tests {
		got := Sqrt(tt.input)
		if !approxTol(got, tt.expected, 1e-9) {
			t.Errorf("Sqrt(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestSqrt_Negative(t *testing.T) {
	if !math.IsNaN(Sqrt(-1)) {
		t.Error("Sqrt(-1) should be NaN")
	}
}

func TestClamp(t *testing.T) {
	if Clamp(5, 0, 10) != 5 {
		t.Error("Clamp(5,0,10) should be 5")
	}
	if Clamp(-1, 0, 10) != 0 {
		t.Error("Clamp(-1,0,10) should be 0")
	}
	if Clamp(15, 0, 10) != 10 {
		t.Error("Clamp(15,0,10) should be 10")
	}
}

func TestLerp(t *testing.T) {
	if !approxTol(Lerp(0, 10, 0.5), 5, 1e-15) {
		t.Error("Lerp(0,10,0.5) should be 5")
	}
	if !approxTol(Lerp(0, 100, 0.25), 25, 1e-15) {
		t.Error("Lerp(0,100,0.25) should be 25")
	}
}

func TestSmoothStep(t *testing.T) {
	if !approxTol(SmoothStep(0, 1, 0), 0, 1e-15) {
		t.Error("SmoothStep(0,1,0) should be 0")
	}
	if !approxTol(SmoothStep(0, 1, 1), 1, 1e-15) {
		t.Error("SmoothStep(0,1,1) should be 1")
	}
	if !approxTol(SmoothStep(0, 1, 0.5), 0.5, 1e-15) {
		t.Errorf("SmoothStep(0,1,0.5) = %v, want 0.5", SmoothStep(0, 1, 0.5))
	}
}

func TestDistance2D(t *testing.T) {
	d := Distance2D(0, 0, 3, 4)
	if !approxTol(d, 5, 1e-15) {
		t.Errorf("Distance2D(0,0,3,4) = %v, want 5", d)
	}
}

// ━━━ Bezier Tests ━━━

func TestBezierPoint_Endpoints(t *testing.T) {
	p := [4]float64{100, 200, 300, 400}
	got := BezierPoint(0, p[0], p[1], p[2], p[3])
	if !approxTol(got, p[0], 1e-15) {
		t.Errorf("BezierPoint(0,...) = %v, want %v", got, p[0])
	}
	got = BezierPoint(1, p[0], p[1], p[2], p[3])
	if !approxTol(got, p[3], 1e-15) {
		t.Errorf("BezierPoint(1,...) = %v, want %v", got, p[3])
	}
}

func TestGenerateBezierPath_Length(t *testing.T) {
	start := [2]float64{0, 0}
	end := [2]float64{100, 100}
	path := GenerateBezierPath(start, end, 10)
	if len(path) != 11 {
		t.Errorf("GenerateBezierPath returned %d points, want 11", len(path))
	}
}

func TestGenerateFastPath(t *testing.T) {
	start := [2]float64{50, 50}
	end := [2]float64{300, 400}
	path := GenerateFastPath(start, end, 5)
	if len(path) != 6 {
		t.Errorf("GenerateFastPath returned %d points, want 6", len(path))
	}
}

// ━━━ Benchmarks ━━━

func BenchmarkSin_EML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Sin(1.234)
	}
}

func BenchmarkSin_Math(b *testing.B) {
	for i := 0; i < b.N; i++ {
		math.Sin(1.234)
	}
}

func BenchmarkCos_EML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Cos(0.567)
	}
}

func BenchmarkExp_EML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Exp(2.5)
	}
}

func BenchmarkExp_Math(b *testing.B) {
	for i := 0; i < b.N; i++ {
		math.Exp(2.5)
	}
}

func BenchmarkLn_EML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Ln(42.0)
	}
}

func BenchmarkSqrt_EML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Sqrt(144.0)
	}
}

func BenchmarkBezierPoint(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BezierPoint(0.5, 0, 100, 200, 300)
	}
}

func BenchmarkGenerateBezierPath(b *testing.B) {
	start := [2]float64{0, 0}
	end := [2]float64{500, 500}
	for i := 0; i < b.N; i++ {
		GenerateBezierPath(start, end, 15)
	}
}
