// Package bsplines provides a CPU implementation of [B-spline](https://en.wikipedia.org/wiki/B-spline).
//
// It also provides configuration that can be used by [github.com/gomlx/bsplines/gomlx] package to
// calculate B-splines efficiently using GoMLX.
//
// For plotting, checkout the sub-package [github.com/gomlx/bsplines/plotly] that will plot the B-spline,
// its basis functions and the same curve using GoMLX using the Plotly library and GoNB notebook.
// It's provided with the demo notebook [bsplines.ipynb].
package bsplines

import (
	"github.com/gomlx/exceptions"
	"slices"
)

// at accesses an arbitrary element of the slice. The difference from the `[]` operator is that it allows
// negative numbers: they are counted from the end of the slice: so -1 refers to the last element.
func at[E any](slice []E, idx int) E {
	if idx < 0 {
		idx = len(slice) + idx
	}
	return slice[idx]
}

//go:generate stringer -type=ExtrapolationType

// ExtrapolationType defines how a B-spline should behave outside the knots (for x < knots[0] or x > knots[-1]).
type ExtrapolationType int

const (
	// ExtrapolateZero configures a B-spline to take value of 0 outside the knots.
	ExtrapolateZero ExtrapolationType = iota

	// ExtrapolateConstant configures a B-spline to take the constant value of the first/last control point outside the knots.
	ExtrapolateConstant

	// ExtrapolateLinear configures a B-spline to extrapolate linearly outside the first/last control point outside the knots.
	ExtrapolateLinear
)

// BSpline contains the basic configuration of a B-spline.
// Notice the control points are not part of the configuration: they are given during evaluation.
//
// It can be used to evaluate points, or with the package [github.com/gomlx/bsplines/plotly] to plot a 1D B-spline
// or, with the package [github.com/gomlx/bsplines/gomlx] to create an evaluator for batch of inputs.
type BSpline struct {
	degree                       int
	expandedKnots, controlPoints []float64
	extrapolation                ExtrapolationType

	// knot(x-coordinate) value for controlPoints[1] and controlPoints[-1], used for
	// linear extrapolation.
	knotValueForControlPoint1, knotValueForControlPointM2 float64
}

// New create a new B-spline with the given [degree] (`order == degree+1`).
// To use it for evaluation, the control points must be given with [WithControlPoints].
//
// The [knots] must be sorted and not be repeated.
// Internally, [degree] extra values are inserted on the start and end of the knots vector, to clamp the endings.
func New(degree int, knots []float64) *BSpline {
	if len(knots) < 2 {
		exceptions.Panicf("bsplines.New requires at least 2 knots, got %d instead", len(knots))
	}
	if !slices.IsSortedFunc(knots, func(a, b float64) int {
		if a < b {
			return -1
		} else {
			// We want repeated numbers to fail.
			return 1
		}
	}) {
		exceptions.Panicf("bsplines.New requires knots to be strictly increasing (no repeats), got %v instead", knots)
	}
	b := &BSpline{
		degree:        degree,
		expandedKnots: make([]float64, len(knots)+2*degree),
		extrapolation: ExtrapolateConstant,
	}
	for ii := range degree {
		// Set clamping points.
		b.expandedKnots[ii] = knots[0]
		b.expandedKnots[len(b.expandedKnots)-ii-1] = at(knots, -1)
	}
	copy(b.expandedKnots[degree:len(b.expandedKnots)-degree], knots)

	// Find control points x-coordinate values:
	controlX := b.ControlPointsX()
	b.knotValueForControlPoint1, b.knotValueForControlPointM2 = controlX[1], at(controlX, -2)
	return b
}

// NewRegular creates a new B-spline that is defined with enough knots for [numControlPoints].
// The knots are created evenly spaced from 0.0 to 1.0.
//
// [numControlPoints] must be at least `degree + 1`.
func NewRegular(degree, numControlPoints int) *BSpline {
	if numControlPoints < degree+1 {
		exceptions.Panicf("bsplines.NewRegular requires numControlPoints=%d >= 2", numControlPoints)
	}
	numKnots := numControlPoints - degree + 1
	knots := make([]float64, numKnots)
	for ii := range knots {
		knots[ii] = float64(ii) / float64(numKnots-1)
	}
	return New(degree, knots)
}

// WithControlPoints associate the given control points to this B-spline.
// There must be exactly `len(knots)+degree-1` control points.
//
// It must be set before evaluation. It can also be switched each time before an evaluation, it's a very cheap operation.
// Notice the knots themselves cannot change -- create another B-spline if different knots are needed.
//
// It returns itself so configuration calls can be cascaded.
func (b *BSpline) WithControlPoints(controlPoints []float64) *BSpline {
	numKnots := len(b.expandedKnots) - 2*b.degree
	if len(controlPoints) != numKnots+b.degree-1 {
		exceptions.Panicf("BSpline.WithControlPoints() with %d knots, expected %d control points (== `len(knots)+degree-1`), but got %d instead", numKnots, numKnots+b.degree-1, len(controlPoints))
	}
	b.controlPoints = controlPoints
	return b
}

// WithExtrapolation defines how the evaluation should extrapolate for values before the first knot or after the
// last knot.
//
// The default value is [ExtrapolateConstant].
//
// It returns itself so configuration calls can be cascaded.
func (b *BSpline) WithExtrapolation(e ExtrapolationType) *BSpline {
	b.extrapolation = e
	return b
}

// Degree of the B-spline.
func (b *BSpline) Degree() int { return b.degree }

// Knots of the B-spline. Values must not be changed -- if one needs to change the knots, create a new B-Spline.
func (b *BSpline) Knots() []float64 {
	return b.expandedKnots[b.degree : len(b.expandedKnots)-b.degree]
}

// NumControlPoints returns the expected number of control points for the current knots.
func (b *BSpline) NumControlPoints() int {
	return len(b.Knots()) + b.degree - 1
}

// ControlPoints returns the control points. at creation time it may be nil.
// To change them, use WithControlPoints instead.
func (b *BSpline) ControlPoints() []float64 {
	return b.controlPoints
}

// ControlPointsX calculates the x values for each one of the control points.
// These values are not something used in the evaluation, but are handy to plot the control points,
// since they are at the center of its area of influence.
func (b *BSpline) ControlPointsX() []float64 {
	numControlPoints := b.NumControlPoints()
	xs := make([]float64, numControlPoints)
	for ii := range numControlPoints {
		if ii == 0 {
			xs[ii] = b.expandedKnots[0]
		} else if ii == len(b.expandedKnots)-1 {
			xs[ii] = b.expandedKnots[len(b.expandedKnots)-1]
		} else {
			for jj := range b.degree {
				xs[ii] += b.expandedKnots[ii+jj+1]
			}
			xs[ii] /= float64(b.degree)
		}
	}
	return xs
}

// Evaluate 1D B-spline on the value of x (some text call this the parameter value, also referred as `t`).
// This function is the simplest version, but not very fast, and run on CPU.
//
// One must set the control points using WithControlPoints before calling this function.
func (b *BSpline) Evaluate(x float64) float64 {
	if len(b.controlPoints) == 0 {
		exceptions.Panicf("BSpline.Evaluate() require control points to be set using BSpline.WithControlPoints()")
	}
	if x < b.expandedKnots[0] || x >= b.expandedKnots[len(b.expandedKnots)-1] {
		return b.extrapolate(x)
	}
	var result float64
	for controlPointIdx, controlPoint := range b.controlPoints {
		basis := b.BasisFunction(controlPointIdx, b.degree, x)
		result += controlPoint * basis
	}
	return result
}

// extrapolate calculates the extrapolation of the b-spline for x -- x is expected to be outside the knots.
func (b *BSpline) extrapolate(x float64) float64 {
	switch b.extrapolation {
	case ExtrapolateZero:
		return 0.0
	case ExtrapolateConstant:
		if x < b.expandedKnots[0] {
			return b.controlPoints[0]
		} else {
			return b.controlPoints[len(b.controlPoints)-1]
		}
	case ExtrapolateLinear:
		if x < b.expandedKnots[0] {
			linearCoef := (b.controlPoints[1] - b.controlPoints[0]) /
				(b.knotValueForControlPoint1 - b.expandedKnots[0])
			return b.controlPoints[0] + (x-b.expandedKnots[0])*linearCoef
		} else {
			linearCoef := (at(b.controlPoints, -1) - at(b.controlPoints, -2)) /
				(at(b.expandedKnots, -1) - b.knotValueForControlPointM2)
			return at(b.controlPoints, -1) + (x-at(b.expandedKnots, -1))*linearCoef
		}
	}
	return 0.0
}

// BasisFunction calculates the B-spline basis function arbitrary degree at parameter x.
// This usually is not used directly, but can be interesting to plot to understand how it is calculated.
func (b *BSpline) BasisFunction(controlPointIdx, degree int, x float64) float64 {
	if degree == 0 {
		// 1 if in the knot interval, 0 otherwise
		if x >= b.expandedKnots[controlPointIdx] && x < b.expandedKnots[controlPointIdx+1] {
			return 1.0
		} else {
			return 0.0
		}
	}
	left := 0.0
	if b.expandedKnots[controlPointIdx+degree] != b.expandedKnots[controlPointIdx] {
		left = (x - b.expandedKnots[controlPointIdx]) / (b.expandedKnots[controlPointIdx+degree] - b.expandedKnots[controlPointIdx]) * b.BasisFunction(controlPointIdx, degree-1, x)
	}

	right := 0.0
	if b.expandedKnots[controlPointIdx+degree+1] != b.expandedKnots[controlPointIdx+1] {
		right = (b.expandedKnots[controlPointIdx+degree+1] - x) / (b.expandedKnots[controlPointIdx+degree+1] - b.expandedKnots[controlPointIdx+1]) * b.BasisFunction(controlPointIdx+1, degree-1, x)
	}
	return left + right
}

// Derivative creates the derivative BSpline of the given BSpline.
// Notice the control points must have been set with WithControlPoints.
//
// The returned BSpline have the same knots, and the degree will be one less than the original.
// The control points are updated.
func (b *BSpline) Derivative() *BSpline {
	knots := b.Knots()
	degree := b.degree - 1
	control := b.controlPoints
	newControl := make([]float64, b.NumControlPoints()-1)
	for ii := range newControl {
		// q_i = p * (c_{i+1} - c_i) / (knot_{i+p+1} - knot_{i+1})
		newControl[ii] = float64(b.degree) *
			(control[ii+1] - control[ii]) /
			(b.expandedKnots[ii+1+b.degree] - b.expandedKnots[ii+1])
	}
	// Extrapolation of the derivative is zero, except if original extrapolation was linear, in which case it is
	// constant.
	var extrapolation ExtrapolationType
	switch b.extrapolation {
	case ExtrapolateZero, ExtrapolateConstant:
		extrapolation = ExtrapolateZero
	case ExtrapolateLinear:
		extrapolation = ExtrapolateConstant
	}
	//fmt.Printf("derivative(p=%d): new control points are %v\n", p, newControl)
	return New(degree, knots).WithExtrapolation(extrapolation).WithControlPoints(newControl)
}
