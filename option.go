package fynefigure

import (
	"fyne.io/fyne/v2/canvas"
	stilusbackend "github.com/timzifer/figure/backend/stilus"
)

// Option configures a [Target].
type Option func(*config)

type config struct {
	raster []stilusbackend.Option
	scale  canvas.ImageScale
	budget float32
}

// Font replaces the rasterizer's embedded Go fonts with supplied TrueType or
// OpenType files. Pass bold or italic as nil to reuse regular for that style.
//
// It is how a chart is drawn in the application's own typeface: package
// fynefigure/chart reads the faces off the Fyne theme and passes them here.
// Without it a chart uses the same fonts every other figure raster does,
// which is what makes its pixels comparable with an exported PNG.
func Font(regular, bold, italic []byte) Option {
	return func(c *config) { c.raster = append(c.raster, stilusbackend.WithFont(regular, bold, italic)) }
}

// ScaleMode sets how Fyne resamples the chart if it ever has to.
//
// It normally does not: the raster is generated at exactly the pixel size the
// painter asks for. The exception is the one frame after a display's device
// pixel ratio changes, where the previous frame is stretched while the next is
// rasterized at the new ratio. The default is [canvas.ImageScaleSmooth].
func ScaleMode(m canvas.ImageScale) Option {
	return func(c *config) { c.scale = m }
}

// DamageBudget turns partial repaints back on, for frames whose damaged region
// covers no more than f of the surface. Zero, the default, repaints the whole
// frame every time.
//
// figure works out where a frame changed and offers the rasterizer the chance
// to repaint only that: it clears the damaged box and clips every drawing call
// to it. Under gg, figure's earlier rasterizer, that clip was a mask sampled
// per pixel and a partial frame cost six times a whole one. stilus clips a
// rectangle for nothing, so the penalty is gone — and so is most of the point,
// because a whole frame is already cheap. Measured on a 900x480 chart on an
// Apple M4, per frame:
//
//	                          whole frame   partial
//	a stream sliding left       0.23 ms     0.18 ms
//	a stream growing at its
//	  right-hand tip            0.41 ms     0.34 ms
//	panning a static chart      0.78 ms     0.80 ms
//
// A fifth saved where it does anything, and a wash where it does not. The
// default stays off, because a partial repaint is one more thing that can
// leave a stale pixel behind, and this is here for a chart whose changes
// really are confined to a corner of it. There is a benchmark; measure before
// turning it on.
func DamageBudget(f float32) Option {
	return func(c *config) {
		if f < 0 {
			f = 0
		}
		c.budget = f
	}
}

func build(opts []Option) config {
	c := config{scale: canvas.ImageScaleSmooth}
	for _, o := range opts {
		o(&c)
	}
	return c
}
