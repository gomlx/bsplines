# Package bsplines: B-Spline support for Go

This library provides 2 implementations of [B-Spline](https://en.wikipedia.org/wiki/B-spline) using the same API: one that evaluates in CPU (slower) and one using `GoMLX` for ML and/or accelerators.

## Highlights:

* Support for zero, constant or linear extrapolation beyond the region defined by the knots.
* GoMLX version:
  * Batch evaluation.
  * Multiple control points -- for various different B-splines to be applied to the same input points.
    They share the same basis function calculation for improved efficiency.
* Plotting using [GoNB](https://github.com/janpfeifer/gonb) Jupyter Notebook.
* See demo notebook with some plot samples. 
