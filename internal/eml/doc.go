// Package eml implements the EML mathematical framework:
// "All elementary functions from a single binary operator."
//
// The fundamental operation is a generalized CORDIC rotation:
//
//	EMLOp(x, y, d) = (x - d*y*m, y + d*x*m)
//
// where m = 2^(-i) is the per-iteration multiplier and d = ±1 is the
// rotation direction.  By iterating this single binary operation with
// pre-computed angle tables, every elementary function (trigonometric,
// exponential, logarithmic, hyperbolic) can be derived at O(1) speed
// with double-precision (float64) accuracy.
//
// Architecture:
//
//	operator.go   — the single binary operator EMLOp + CORDIC engine
//	trig.go       — Sin, Cos, Tan, ArcSin, ArcCos, ArcTan, ArcTan2
//	exp_log.go    — Exp, Ln, Log2, Log10, Pow
//	hyperbolic.go — Sinh, Cosh, Tanh, ArcSinh, ArcCosh, ArcTanh
//	bezier.go     — Bezier curve interpolation for cursor movement
//	special.go    — Sqrt, Abs, Clamp, Lerp
package eml
