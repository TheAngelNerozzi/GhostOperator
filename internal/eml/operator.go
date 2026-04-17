package eml

import "math"

// maxIterations controls precision.  32 iterations yield ~15 decimal
// digits of accuracy for float64 (the full precision of the type).
const maxIterations = 32

// cordicMode selects the coordinate system for the rotation.
type cordicMode int

const (
	circular  cordicMode = 0
	linear    cordicMode = 1
	hyperbolic cordicMode = -1
)

// precomputedTables holds the pre-calculated angle tables and gain for
// each CORDIC mode.  Initialized once by init().
var precomputedTables struct {
	angleTableCircular  [maxIterations]float64
	angleTableHyperbolic [maxIterations]float64
	gainCircular        float64
	gainHyperbolic      float64
	// Max angle the circular CORDIC can converge to (sum of all arctan(2^-i))
	maxCircularAngle float64
	// Max angle the hyperbolic CORDIC can converge to
	maxHyperbolicAngle float64
}

func init() {
	var sumCirc, sumHyp float64 = 0.0, 0.0
	var gc, gh float64 = 1.0, 1.0
	for i := 0; i < maxIterations; i++ {
		pow := math.Pow(2, -float64(i))
		precomputedTables.angleTableCircular[i] = math.Atan(pow)
		precomputedTables.angleTableHyperbolic[i] = math.Atanh(pow)
		sumCirc += math.Atan(pow)
		gc *= math.Sqrt(1.0 + pow*pow)
		// Hyperbolic: skip i where (i+1)%4==0 for i>=4 (convergence fix)
		skip := (i >= 4) && ((i+1)%4 == 0)
		if !skip {
			sumHyp += math.Atanh(pow)
			gh *= math.Sqrt(1.0 - pow*pow)
		}
	}
	precomputedTables.gainCircular = gc
	precomputedTables.gainHyperbolic = gh
	precomputedTables.maxCircularAngle = sumCirc  // ~1.7433 rad
	precomputedTables.maxHyperbolicAngle = sumHyp // ~1.1182 rad
}

// EMLOp is the single fundamental binary operator of the EML framework.
//
// It performs one CORDIC micro-rotation:
//
//	EMLOp(x, y, d, m) = (x - d*y*m, y + d*x*m)
//
// where d ∈ {-1, +1} is the rotation direction and m = 2^{-i}
// is the per-step multiplier.  Every elementary function in the EML
// framework is ultimately a composition of EMLOp calls across 32
// iterations with pre-computed tables.
func EMLOp(x, y float64, d int, m float64) (float64, float64) {
	return x - float64(d)*y*m, y + float64(d)*x*m
}

// cordicRotateCircular performs CORDIC rotation in circular mode.
// Input angle must be in [-π/2, π/2] for convergence.
// With x0=1/K, y0=0: returns (cos(θ), sin(θ)).
func cordicRotateCircular(x, y, theta float64) (rx, ry float64) {
	accumAngle := 0.0
	cx, cy := x, y
	for i := 0; i < maxIterations; i++ {
		pow := math.Pow(2, -float64(i))
		angle := precomputedTables.angleTableCircular[i]
		var d int
		if theta > accumAngle {
			d = 1
		} else {
			d = -1
		}
		cx, cy = EMLOp(cx, cy, d, pow)
		accumAngle += float64(d) * angle
	}
	return cx, cy
}

// cordicVectorCircular performs CORDIC vectoring in circular mode.
// Drives y→0 while accumulating the angle.
// Returns (magnitude * K, arctan(y/x)).
func cordicVectorCircular(x, y float64) (mag, angle float64) {
	accumAngle := 0.0
	cx, cy := x, y
	for i := 0; i < maxIterations; i++ {
		pow := math.Pow(2, -float64(i))
		angle := precomputedTables.angleTableCircular[i]
		var d int
		if cy < 0 {
			d = 1
		} else {
			d = -1
		}
		cx, cy = EMLOp(cx, cy, d, pow)
		accumAngle += float64(d) * angle
	}
	return cx, accumAngle
}

// cordicVectorHyperbolic performs CORDIC vectoring in hyperbolic mode.
func cordicVectorHyperbolic(x, y float64) (mag, angle float64) {
	accumAngle := 0.0
	cx, cy := x, y
	for i := 0; i < maxIterations; i++ {
		skip := (i >= 4) && ((i+1)%4 == 0)
		if skip {
			continue
		}
		pow := math.Pow(2, -float64(i))
		angle := precomputedTables.angleTableHyperbolic[i]
		var d int
		if cy < 0 {
			d = 1
		} else {
			d = -1
		}
		cx, cy = EMLOp(cx, cy, d, pow)
		accumAngle += float64(d) * angle
	}
	return cx, accumAngle
}

// cordicRotateHyperbolic performs CORDIC rotation in hyperbolic mode.
func cordicRotateHyperbolic(x, y, theta float64) (rx, ry float64) {
	accumAngle := 0.0
	cx, cy := x, y
	for i := 0; i < maxIterations; i++ {
		skip := (i >= 4) && ((i+1)%4 == 0)
		if skip {
			continue
		}
		pow := math.Pow(2, -float64(i))
		angle := precomputedTables.angleTableHyperbolic[i]
		var d int
		if theta > accumAngle {
			d = 1
		} else {
			d = -1
		}
		cx, cy = EMLOp(cx, cy, d, pow)
		accumAngle += float64(d) * angle
	}
	return cx, ry
}
