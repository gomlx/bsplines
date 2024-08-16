# B-Spline function support for Go 

[![GoDev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/gomlx/bsplines?tab=doc)
[![GitHub](https://img.shields.io/github/license/gomlx/bsplines)](https://github.com/Kwynto/gosession/blob/master/LICENSE)

# [Example With Plots](https://gomlx.github.io/bsplines/)

This library provides 2 implementations of [B-Spline](https://en.wikipedia.org/wiki/B-spline) using the same API: one that evaluates fully in Go (CPU, slower)
and one using [`GoMLX`](https://github.com/gomlx/gomlx) for ML and/or accelerators.

## Highlights:

* Support for zero, constant or linear extrapolation beyond the region defined by the knots.
* Derivative B-spline.
* GoMLX "vector" version in [github.com/gomlx/gomlx/ml/layers/kan](https://github.com/gomlx/gomlx/tree/main/ml/layers/kan)
  * Building block to build [KAN: Kolmogorov–Arnold Networks](https://arxiv.org/pdf/2404.19756)
  * Batch evaluation.
  * Multiple control points -- for various different B-splines to be applied to the same input points.
    They share the same basis function calculation for improved efficiency.
* Plotting using [`GoNB`](https://github.com/janpfeifer/gonb) Jupyter Notebook.
* See [demo notebook with some plot samples](https://gomlx.github.io/bsplines/). 

## Changes Log

### v0.2.0

* Moved GoMLX code to github.com/gomlx/gomlx/ml/layers/kan, to avoid circular dependency across repositories.