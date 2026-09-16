package fynefigure

import (
	"fyne.io/fyne/v2/canvas"
	ggbackend "github.com/timzifer/figure/backend/gg"
)

// Option configures a [Target].
type Option func(*config)

type config struct {
	// font is regular, bold and italic, and fallback are the faces consulted
	// for a rune none of them has a glyph for. They are held apart from the
	// rasterizer's options because [Target.SetFont] replaces one without
	// meaning to drop the other: a chart that followed the application's
	// typeface to a new one keeps the fallback it was given.
	font     [3][]byte
	fallback [][]byte
	scale    canvas.ImageScale
	budget   float32
}

// gg is the rasterizer's own options for this configuration.
func (c config) gg() []ggbackend.Option {
	var opts []ggbackend.Option
	if len(c.font[0]) > 0 {
		opts = append(opts, ggbackend.WithFont(c.font[0], c.font[1], c.font[2]))
	}
	if len(c.fallback) > 0 {
		opts = append(opts, ggbackend.WithFallbackFont(c.fallback...))
	}
	return opts
}

// Font replaces the rasterizer's embedded Go fonts with supplied TrueType or
// OpenType files. Pass bold or italic as nil to reuse regular for that style.
//
// It is how a chart is drawn in the application's own typeface: package
// fynefigure/chart reads the faces off the Fyne theme and passes them here.
// Without it a chart uses the same fonts every other figure raster does,
// which is what makes its pixels comparable with an exported PNG.
func Font(regular, bold, italic []byte) Option {
	return func(c *config) { c.font = [3][]byte{regular, bold, italic} }
}

// FallbackFont adds fonts consulted, in order, for a rune the chart's own font
// has no glyph for.
//
// A rasterizer draws a label with the one typeface it was given, where a
// vector viewer asks the reader's system for a face that has the glyph. So a
// chart drawn in an application's own font loses the characters that font does
// not cover — Fyne's NotoSans has no ≤, and the rasterizer's embedded Go fonts
// have no ⟨ or ⟩, which is a Bloch sphere's |0⟩. This is where a face that has
// them is supplied. A rune no face can draw is written as `?` rather than
// dropped, because a label that quietly loses a character says something the
// data does not.
//
// The metrics stay the chart's own font's, so a fallback glyph in one label
// does not move the baseline of the row it is in. Package fynefigure/chart
// passes the rasterizer's own fonts here when it follows the Fyne theme's
// typeface, which is what keeps a themed chart's mathematical symbols.
func FallbackFont(ttf ...[]byte) Option {
	return func(c *config) { c.fallback = append(c.fallback, ttf...) }
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
// to repaint only that. It sounds like a saving and here it is not: the
// rasterizer clears the damaged box and clips every drawing call to it, and its
// clip is a mask it rasterizes across the surface and then samples per pixel —
// on top of the clip a panel already puts there. Measured on a 900x480 chart,
// per frame:
//
//	                          whole frame   partial
//	a stream sliding left        29 ms       180 ms
//	a stream growing at its
//	  right-hand tip             30 ms       170 ms
//	panning a static chart       50 ms        50 ms
//
// Six times slower where it does anything, and a wash where it does not. So
// the default is off, and this is here for a chart whose changes really are
// confined to a corner of it, and for the day the rasterizer's clip gets
// cheaper. There is a benchmark; measure before turning it on.
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
