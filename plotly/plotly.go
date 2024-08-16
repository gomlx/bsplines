// Package plotly implements plotting in Jupyter Notebooks using [github.com/janpfeifer/gonb] (Notebook Kernel) and
// the Plotly [github.com/MetalBlueberry/go-plotly] library.
//
// Use New to create a new Config object, and after configuring it, use Config.Plot to draw the plot.
//
// Features:
//   - B-spline function, visible by default.
//   - Control Points, visible by default.
//   - Derivative, non-visible by default.
//   - Basis functions, non-visible by default.
package plotly

import (
	"fmt"
	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"github.com/gomlx/bsplines"
	"github.com/janpfeifer/gonb/gonbui/plotly"
)

// Config holds a plot configuration that can be changed.
// Once finished, call the method [Plot] to actually plot.
type Config struct {
	bspline       *bsplines.BSpline
	numPlotPoints int
	marginRatio   float64
}

// New returns a Config object that can be changed.
// Once finished, call Config.Plot to draw the plot in the Jupyter notebook.
func New(b *bsplines.BSpline) *Config {
	return &Config{
		bspline:       b,
		numPlotPoints: 1000,
		marginRatio:   0.1,
	}
}

// WithNumPlotPoints set the number of plot points to evaluate. Default is 1000.
func (c *Config) WithNumPlotPoints(numPlotPoints int) *Config {
	if numPlotPoints < 2 {
		numPlotPoints = 2
	}
	c.numPlotPoints = numPlotPoints
	return c
}

// WithMargin defines how much space (relative to the defined B-spline range) to plot.
// It defaults to 0.1, and it's handy to see how the curve is going to extrapolate beyond its boundaries.
func (c *Config) WithMargin(marginRatio float64) *Config {
	if marginRatio < 0 {
		marginRatio = 0
	}
	c.marginRatio = marginRatio
	return c
}

// Plot using the current configuration.
// It returns an error if plotting failed for some reason.
func (c *Config) Plot() error {
	knots := c.bspline.Knots()
	derivative := c.bspline.Derivative()

	x, bsplineY, derivativeY := make([]float64, c.numPlotPoints), make([]float64, c.numPlotPoints), make([]float64, c.numPlotPoints)
	first, last := knots[0], knots[len(knots)-1]
	delta := last - first
	first, last = first-c.marginRatio*delta, last+c.marginRatio*delta
	for ii := range c.numPlotPoints {
		x[ii] = first + (last-first)*float64(ii)/float64(c.numPlotPoints)
		bsplineY[ii] = c.bspline.Evaluate(x[ii])
		derivativeY[ii] = derivative.Evaluate(x[ii])
	}
	basisPlots := make([][]float64, c.bspline.NumControlPoints())
	for controlIdx := range len(basisPlots) {
		basisPlots[controlIdx] = make([]float64, c.numPlotPoints)
		basisPlot := basisPlots[controlIdx]
		for ii := range c.numPlotPoints {
			basisPlot[ii] = c.bspline.BasisFunction(controlIdx, c.bspline.Degree(), x[ii])
		}
	}

	controls := c.bspline.ControlPoints()
	fig := &grob.Fig{
		Data: grob.Traces{
			&grob.Bar{
				Name:       "Control Points",
				X:          c.bspline.ControlPointsX(),
				Y:          controls,
				Showlegend: grob.True,
				Marker: &grob.BarMarker{
					Line: &grob.BarMarkerLine{
						Width: 3.0,
					},
				},
			},
			&grob.Bar{
				Name:       "B-spline",
				X:          x,
				Y:          bsplineY,
				Width:      2.0,
				Showlegend: grob.True,
			},
			&grob.Bar{
				Name:       "1st derivative",
				X:          x,
				Y:          derivativeY,
				Width:      2.0,
				Showlegend: grob.True,
				Visible:    grob.BarVisibleLegendonly,
			},
		},
		Layout: &grob.Layout{
			Title: &grob.LayoutTitle{
				Text: "B-Spline",
			},
			Legend: &grob.LayoutLegend{},
		},
	}
	for controlIdx := range len(controls) {
		basisPlot := basisPlots[controlIdx]
		fig.Data = append(fig.Data,
			&grob.Bar{
				Name:       fmt.Sprintf("Basis(idx=%d, control[idx]=%f, degree=%d)", controlIdx, controls[controlIdx], c.bspline.Degree()),
				X:          x,
				Y:          basisPlot,
				Showlegend: grob.True,
				Width:      0.5,
				Visible:    grob.BarVisibleLegendonly,
			},
		)
	}

	err := plotly.DisplayFig(fig)
	if err != nil {
		err = fmt.Errorf("plotly.DisplayFig failed: %v", err)
	}
	return err
}
