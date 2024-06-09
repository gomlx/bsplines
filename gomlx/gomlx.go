// Package gomlx implements B-splines evaluation using GoMLX.
// It returns a computation graph that can be combined/used on other computations, e.g., to implement
// calibration layers for input of neural-networks, or for "KAN - Kolmogorov-Arnold Networks" [1]
//
// It is meant to work for batches of inputs, each example with multiple inputs and outputs, pay special
// attention to the shapes of the control points, inputs and outputs. They are documented in the [Evaluator]
// function.
//
// [1] https://arxiv.org/pdf/2404.19756
package gomlx

import (
	. "github.com/gomlx/gomlx/graph"
	"github.com/gomlx/gomlx/types/shapes"
	"github.com/gomlx/gomlx/types/slices"
)

func basisFunctionGraph(p int, u, knots *Node) *Node {
	dtype := u.DType()

	if p == 0 {
		// u >= knots[i] && u < knots[i+1]
		cond := And(
			GreaterOrEqual(u, knots),
			ShiftLeft(LessThan(u, knots), 1, 0.0))
		p0 := ConvertType(cond, dtype) // true -> 1.0, false -> 0.0
		//p0.SetLogged("basis(0)")
		return p0
	}

	recursiveBasis := basisFunctionGraph(p-1, u, knots)

	// Find knotsDelta `p` steps ahead: replace zeros with ones for numeric safety.
	knotsDelta := Sub(Shift(knots, -1, ShiftLeftDir, p), knots)
	//knotsDelta.SetLogged(fmt.Sprintf("knotsDelta(%d)", p))
	zeros := ZerosLike(knotsDelta)
	knotsDeltaIsZero := Equal(knotsDelta, zeros)
	knotsDelta = Where(knotsDeltaIsZero, OnesLike(knotsDelta), knotsDelta)

	//knotsDeltaIsZero.SetLogged(fmt.Sprintf("knotsDeltaIsZero(%d)", p))

	weightsLeft := Div(Sub(u, knots), knotsDelta)
	weightsLeft = Where(knotsDeltaIsZero, zeros, weightsLeft)
	left := Mul(weightsLeft, recursiveBasis)
	//left.SetLogged(fmt.Sprintf("left(%d)", p))

	weightsRight := Sub(Shift(knots, -1, ShiftLeftDir, p+1), u)
	weightsRight = Div(weightsRight, Shift(knotsDelta, -1, ShiftLeftDir, 1))
	weightsRight = Where(Shift(knotsDeltaIsZero, -1, ShiftLeftDir, 1), zeros, weightsRight)
	right := Mul(weightsRight, Shift(recursiveBasis, -1, ShiftLeftDir, 1))
	//right.SetLogged(fmt.Sprintf("right(%d)", p))

	return Add(left, right)
}

func evaluateGraph(degree int, u, knots, controlPoints *Node) *Node {
	//u.SetLogged("x")
	basis := basisFunctionGraph(degree, u, knots)
	basis = Slice(basis, AxisRange(0, controlPoints.Shape().Dimensions[0]))
	//basis.SetLogged(fmt.Sprintf("basis(%d)", degree))
	return Dot(controlPoints, basis)
}

type BSplineGomlx struct {
	b *BSpline
}

func (b *BSpline) GraphEvaluator() *BSplineGomlx {
	if slices.Last(knots.Shape().Dimensions) != slices.Last(controlPoints.Shape().Dimensions)+degree+1 {
		exceptions.Panicf("bsplines.New(): knots last dimension (%d) should be equal to controlPoints last dimension (%d) + degree (%d) + 1",
			slices.Last(knots.Shape().Dimensions), slices.Last(controlPoints.Shape().Dimensions), degree)
	}

	bg := &BSplineGomlx{
		b: b,
	}
	bg.evalExec = NewExec(manager, func(x, knots, controlPoints *Node) *Node {
		x = ConvertType(x, controlPoints.DType())
		f := evaluateGraph(degree, x, knots, controlPoints)
		return ConvertType(f, shapes.Float64)
	})
	return b
}
