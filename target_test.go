package fynefigure_test

import (
	"bytes"
	"image"
	"image/png"

	"math"
	"testing"
	"time"

	"fyne.io/fyne/v2/canvas"

	"github.com/timzifer/figure"
	ggbackend "github.com/timzifer/figure/backend/gg"
	"github.com/timzifer/figure/geom"
	"github.com/timzifer/fynefigure"
)

// The claim this package rests on is that a chart in a Fyne widget is the
// chart figure would have written to a file. It holds because there is one
// rasterizer: the target here is backend/gg's surface with a Fyne front end,
// so anything that made the two differ would be the presentation layer having
// got into the drawing path.

func TestAFrameIsWhatTheRasterBackendWouldHaveWritten(t *testing.T) {
	var buf bytes.Buffer
	if err := plot().Render(ggbackend.Writer(&buf, ggbackend.FormatPNG)); err != nil {
		t.Fatalf("rendering the reference PNG: %v", err)
	}
	want, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("decoding the reference PNG: %v", err)
	}

	target := fynefigure.New()
	live, err := plot().Live(target)
	if err != nil {
		t.Fatalf("opening the chart: %v", err)
	}
	defer live.Close()
	if err := live.Draw(); err != nil {
		t.Fatalf("drawing into the Fyne target: %v", err)
	}
	got := target.Image()
	if got == nil {
		t.Fatal("the target has no pixels after a frame")
	}

	if got.Bounds() != want.Bounds() {
		t.Fatalf("the frame is %v, want %v", got.Bounds(), want.Bounds())
	}
	if n := differingPixels(got, want); n != 0 {
		t.Errorf("%d pixels differ from what the raster backend writes", n)
	}
}

func TestTheFrameFollowsTheDevicePixelRatio(t *testing.T) {
	target := fynefigure.New()
	p := figure.New(figure.Size(400, 250), figure.DPR(2))
	p.Add(geom.Line(source(), geom.X("t"), geom.Y("y")))
	live, err := p.Live(target)
	if err != nil {
		t.Fatalf("opening the chart: %v", err)
	}
	defer live.Close()
	if err := live.Draw(); err != nil {
		t.Fatalf("drawing: %v", err)
	}

	w, h, dpr := target.Size()
	if w != 400 || h != 250 || dpr != 2 {
		t.Errorf("the target reports %dx%d at dpr %v, want 400x250 at 2", w, h, dpr)
	}
	if b := target.Image().Bounds(); b.Dx() != 800 || b.Dy() != 500 {
		t.Errorf("the buffer is %v, want 800x500 device pixels", b)
	}
}

// OnFrame is how a program measures a chart without wrapping every path that
// draws it, so it has to hear about every call through Render and tell a frame
// that painted from one figure found identical to the last.
func TestOnFrameReportsWhatEachDrawCostAndWhetherItPainted(t *testing.T) {
	target := fynefigure.New()
	var frames []fynefigure.Frame
	target.OnFrame(func(f fynefigure.Frame) { frames = append(frames, f) })

	live, err := plot().Live(target)
	if err != nil {
		t.Fatalf("opening the chart: %v", err)
	}
	defer live.Close()

	for range 2 {
		if err := target.Render(live.Draw); err != nil {
			t.Fatalf("drawing: %v", err)
		}
	}
	if len(frames) != 2 {
		t.Fatalf("OnFrame was called %d times for two Renders", len(frames))
	}
	first, second := frames[0], frames[1]
	if !first.Painted || first.Cost <= 0 || first.At.IsZero() {
		t.Errorf("the first frame reports %+v, want painted with a cost and a time", first)
	}
	if w, h, _ := target.Size(); first.W != w || first.H != h {
		t.Errorf("the first frame reports %dx%d, the target is %dx%d", first.W, first.H, w, h)
	}
	if second.Painted {
		t.Error("a frame identical to the last is reported as painted")
	}

	target.OnFrame(nil)
	if err := target.Render(live.Draw); err != nil {
		t.Fatalf("drawing: %v", err)
	}
	if len(frames) != 2 {
		t.Error("OnFrame(nil) did not stop the reports")
	}
}

// Input is what turns a frame's cost into the roundtrip a reader feels: the
// first painted frame after an event reports how long after it it finished,
// and an event answered without a frame is not charged to the next one.
func TestAFrameReportsHowLongTheInputItAnswersWaited(t *testing.T) {
	target := fynefigure.New()
	var frames []fynefigure.Frame
	target.OnFrame(func(f fynefigure.Frame) { frames = append(frames, f) })

	live, err := plot().Live(target)
	if err != nil {
		t.Fatalf("opening the chart: %v", err)
	}
	defer live.Close()
	if err := target.Render(live.Draw); err != nil {
		t.Fatalf("drawing: %v", err)
	}
	if frames[0].Latency != 0 {
		t.Errorf("a frame no input asked for reports a latency of %v", frames[0].Latency)
	}

	// An event, a wait the way a paced widget would impose one, then a frame.
	target.Input()
	time.Sleep(5 * time.Millisecond)
	if err := target.Render(func() error { return live.Resize(300, 200) }); err != nil {
		t.Fatalf("resizing: %v", err)
	}
	if f := frames[len(frames)-1]; !f.Painted || f.Latency < 5*time.Millisecond {
		t.Errorf("the frame answering the input reports %+v, want painted with at least 5ms latency", f)
	}

	// An event answered by a call that painted nothing: the next frame, drawn
	// for some other reason, must not be charged with it.
	target.Input()
	if err := target.Render(live.Draw); err != nil {
		t.Fatalf("drawing: %v", err)
	}
	if err := target.Render(func() error { return live.Resize(320, 200) }); err != nil {
		t.Fatalf("resizing: %v", err)
	}
	if f := frames[len(frames)-1]; f.Latency != 0 {
		t.Errorf("a frame drawn after the input was answered reports a latency of %v", f.Latency)
	}
}

func TestAFrameThatChangedNothingIsNotPresentedAgain(t *testing.T) {
	target := fynefigure.New()
	live, err := plot().Live(target)
	if err != nil {
		t.Fatalf("opening the chart: %v", err)
	}
	defer live.Close()

	if err := live.Draw(); err != nil {
		t.Fatalf("first frame: %v", err)
	}
	first := target.Frames()
	if first == 0 {
		t.Fatal("the first frame painted nothing")
	}
	if err := live.Draw(); err != nil {
		t.Fatalf("second frame: %v", err)
	}
	if got := target.Frames(); got != first {
		t.Errorf("redrawing an unchanged chart painted %d frames, want the %d already painted", got, first)
	}
}

func TestTheRasterReportsAGeometryItWasNotDrawnAt(t *testing.T) {
	target := fynefigure.New()
	live, err := plot().Live(target)
	if err != nil {
		t.Fatalf("opening the chart: %v", err)
	}
	defer live.Close()
	if err := live.Draw(); err != nil {
		t.Fatalf("drawing: %v", err)
	}

	var gotW, gotH, calls int
	target.OnGeometry(func(w, h int) { gotW, gotH, calls = w, h, calls+1 })

	raster := target.Object().(*canvas.Raster)

	// The size it was drawn at is not news.
	if img := raster.Generator(400, 250); img == nil {
		t.Fatal("the generator produced no image")
	}
	if calls != 0 {
		t.Errorf("the geometry it was drawn at was reported %d times, want 0", calls)
	}

	// A different one is: this is a display whose device pixel ratio changed.
	if img := raster.Generator(800, 500); img == nil {
		t.Fatal("the generator produced no image")
	}
	if calls != 1 || gotW != 800 || gotH != 500 {
		t.Errorf("the painter's geometry was reported as %dx%d %d times, want 800x500 once", gotW, gotH, calls)
	}
}

func TestTheRasterShowsSomethingBeforeThereIsAChart(t *testing.T) {
	raster := fynefigure.New().Object().(*canvas.Raster)
	img := raster.Generator(120, 80)
	if img == nil {
		t.Fatal("a target with no chart generated no image")
	}
	if b := img.Bounds(); b.Dx() != 120 || b.Dy() != 80 {
		t.Errorf("the placeholder is %v, want 120x80", b)
	}
}

// differingPixels counts how many pixels of a and b are not identical.
func differingPixels(a, b image.Image) int {
	n := 0
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ar, ag, ab, aa := a.At(x, y).RGBA()
			br, bg, bb, ba := b.At(x, y).RGBA()
			if ar != br || ag != bg || ab != bb || aa != ba {
				n++
			}
		}
	}
	return n
}

func plot() *figure.Plot {
	p := figure.New(figure.Size(400, 250), figure.Title("Signal"))
	p.Add(geom.Line(source(), geom.X("t"), geom.Y("y")))
	return p
}

func source() figure.Source {
	const n = 200
	x := make([]float64, n)
	y := make([]float64, n)
	for i := range n {
		t := float64(i) / 20
		x[i], y[i] = t, math.Sin(t)
	}
	return figure.Float64Columns(map[string][]float64{"t": x, "y": y})
}
