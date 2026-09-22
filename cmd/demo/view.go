package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/timzifer/figure"
	ggbackend "github.com/timzifer/figure/backend/gg"
	"github.com/timzifer/figure/interact"
	"github.com/timzifer/figure/three"
	"github.com/timzifer/fynefigure"
	"github.com/timzifer/fynefigure/chart"
	"github.com/timzifer/fynefigure/orbit"
)

// env is what a chart is built with: the panel's switches as they stand, and
// the line under the stage to report what is under the pointer.
type env struct {
	interactive bool
	tooltip     bool
	detail      bool
	status      func(string)
}

// chartOpts are the options every flat chart gets, then the caller's.
func (e env) chartOpts(extra ...chart.Option) []chart.Option {
	opts := []chart.Option{chart.Interactive(e.interactive), chart.Tooltip(e.tooltip), chart.TrackRows(true)}
	if e.detail {
		// Coarse while it is dragged, sharp when it stops: worth about three
		// times the frame rate on the CPU rasterizer.
		opts = append(opts, chart.Detail(0.5))
	}
	return append(opts, extra...)
}

// orbitOpts are the options every scene gets, then the caller's.
func (e env) orbitOpts(extra ...orbit.Option) []orbit.Option {
	return append([]orbit.Option{orbit.Interactive(e.interactive)}, extra...)
}

// readout is the hover handler of a flat chart: what is under the pointer.
func (e env) readout(ev figure.Event) {
	if !ev.Found {
		e.status("")
		return
	}
	e.status(fmt.Sprintf("%s   x %.4g   y %.4g   row %d", ev.Series(), ev.Hit.X, ev.Hit.Y, ev.Hit.Row))
}

// A projected mark has no x and y to read back, so a hover over a scene says
// which view and which row — which is the whole answer a pointer has there.
func (e env) sceneReadout(h interact.Hit, found bool) {
	if !found {
		e.status("")
		return
	}
	e.status(fmt.Sprintf("view %d   %s   row %d", h.Panel+1, h.Series, h.Row))
}

func (e env) cameraReadout(i int, cam three.Camera) {
	e.status(fmt.Sprintf("view %d   azimuth %.0f°   elevation %.0f°   zoom %.2f",
		i+1, degrees(cam.Azimuth()), degrees(cam.Elevation()), cam.Zoom()))
}

func degrees(rad float64) float64 { return rad * 180 / math.Pi }

// view is one entry on stage: the object shown, and the widgets in it, which
// is what the panel's switches and numbers reach.
type view struct {
	obj    fyne.CanvasObject
	flats  []*chart.Chart
	orbits []*orbit.Chart
	still  *still

	// start begins whatever animates the view and returns what stops it. It
	// is nil for a view that only moves when it is touched.
	start func() (stop func())
}

// flatView is a plot in a chart widget.
func flatView(e env, p *figure.Plot) *view {
	c := chart.New(p, e.chartOpts()...)
	p.On(figure.Hover, e.readout)
	return &view{obj: c, flats: []*chart.Chart{c}}
}

// orbitView is a scene in an orbit widget.
func orbitView(e env, p *three.Plot) *view {
	c := orbit.New(p, e.orbitOpts(orbit.Select(true))...)
	c.OnHover(e.sceneReadout)
	c.OnCamera(e.cameraReadout)
	return &view{obj: c, orbits: []*orbit.Chart{c}}
}

// gridView is a grid of plots, which has no widget: it is drawn once, by the
// same rasterizer, and shown as a picture.
func gridView(g *figure.Grid) *view {
	s := newStill(g)
	return &view{obj: s, still: s}
}

// attach hands every widget in the view the recorder's frame callback, and
// draws a still, which has no layout to wait for.
func (v *view) attach(fn func(fynefigure.Frame)) {
	for _, c := range v.flats {
		c.OnFrame(fn)
	}
	for _, c := range v.orbits {
		c.OnFrame(fn)
	}
	if v.still != nil {
		v.still.sink = fn
		v.still.render()
	}
}

// close releases every widget's rasterizer. The view must be off the stage by
// then: a chart laid out once more after Close would open a new one.
func (v *view) close() {
	for _, c := range v.flats {
		_ = c.Close()
	}
	for _, c := range v.orbits {
		_ = c.Close()
	}
}

func (v *view) setInteractive(on bool) {
	for _, c := range v.flats {
		c.SetInteractive(on)
	}
	for _, c := range v.orbits {
		c.SetInteractive(on)
	}
}

// setTooltip reaches the flat charts only: a scene has no tooltip.
func (v *view) setTooltip(on bool) {
	for _, c := range v.flats {
		c.SetTooltip(on)
	}
}

// fit is what a double click does: every flat chart autoscaled, every scene
// sent home.
func (v *view) fit() error {
	var errs []error
	for _, c := range v.flats {
		errs = append(errs, c.Autoscale())
	}
	for _, c := range v.orbits {
		errs = append(errs, c.Home())
	}
	return errors.Join(errs...)
}

// err is what went wrong in the view's last frame, if anything.
func (v *view) err() error {
	var errs []error
	for _, c := range v.flats {
		errs = append(errs, c.Err())
	}
	for _, c := range v.orbits {
		errs = append(errs, c.Err())
	}
	if v.still != nil {
		errs = append(errs, v.still.err)
	}
	return errors.Join(errs...)
}

// export writes the view's first chart to a PNG: the same plot through the
// same rasterizer, so what lands in the file is what is on screen.
func (v *view) export(path string) error {
	switch {
	case len(v.flats) > 0:
		return v.flats[0].Plot().Render(ggbackend.PNG(path))
	case len(v.orbits) > 0:
		return v.orbits[0].Plot().Render(ggbackend.PNG(path))
	case v.still != nil:
		return v.still.grid.Render(ggbackend.PNG(path))
	}
	return errors.New("nothing on stage")
}

// The benchmark drives the view's first widget one frame at a time: a flat
// chart panned by a pixel and back, a scene turned a little and back, a still
// drawn again.

func (v *view) benchable() bool {
	return len(v.flats) > 0 || len(v.orbits) > 0 || v.still != nil
}

// aim puts figure's idea of the pointer in the middle of a flat chart. A pan
// moves the panel under the last pointer position, and a chart nobody has
// pointed at yet has none.
func (v *view) aim() {
	if len(v.flats) == 0 {
		return
	}
	l, t := v.flats[0].Live(), v.flats[0].Target()
	if l == nil || t == nil {
		return
	}
	w, h, dpr := t.Size()
	_ = t.Render(func() error {
		l.Move(float64(w)*dpr/2, float64(h)*dpr/2)
		return nil
	})
}

// nudge draws frame i of a benchmark.
func (v *view) nudge(i int) {
	d := 1.0
	if i%2 == 1 {
		d = -1
	}
	switch {
	case len(v.flats) > 0:
		l, t := v.flats[0].Live(), v.flats[0].Target()
		if l == nil || t == nil {
			return
		}
		_ = t.Render(func() error { return l.PanBy(d, 0) })
		t.Present()
	case len(v.orbits) > 0:
		c := v.orbits[0]
		_ = c.SetCamera(0, three.Orbit(c.Camera(0), 0.02*d, 0))
	case v.still != nil:
		v.still.render()
	}
}

// still is a picture of a chart that has no widget of its own.
type still struct {
	widget.BaseWidget
	grid *figure.Grid
	img  *canvas.Image
	sink func(fynefigure.Frame)
	err  error
}

func newStill(g *figure.Grid) *still {
	s := &still{grid: g, img: canvas.NewImageFromImage(nil)}
	s.img.FillMode = canvas.ImageFillContain
	s.img.ScaleMode = canvas.ImageScaleSmooth
	s.img.SetMinSize(fyne.NewSize(240, 160))
	s.ExtendBaseWidget(s)
	return s
}

func (s *still) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(s.img) }

// render draws the grid and reports the frame the way a widget's target would.
// A grid is laid out at the size it was made for; the picture is scaled to
// the stage.
func (s *still) render() {
	start := time.Now()
	var buf bytes.Buffer
	if s.err = s.grid.Render(ggbackend.Writer(&buf, ggbackend.FormatPNG)); s.err != nil {
		return
	}
	img, err := png.Decode(&buf)
	if s.err = err; err != nil {
		return
	}
	f := fynefigure.Frame{At: time.Now(), Painted: true, W: img.Bounds().Dx(), H: img.Bounds().Dy(), DPR: 1}
	f.Cost = f.At.Sub(start)
	s.img.Image = img
	s.img.Refresh()
	if s.sink != nil {
		s.sink(f)
	}
}
