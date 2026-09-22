package chart_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"github.com/timzifer/figure"
	"github.com/timzifer/fynefigure"
	"github.com/timzifer/fynefigure/chart"
)

// renders counts the calls through the chart's surface, painted or not.
func renders(c *chart.Chart) *int {
	n := new(int)
	c.OnFrame(func(fynefigure.Frame) { *n++ })
	return n
}

// painted counts the frames the chart actually rasterized.
func painted(c *chart.Chart) *int {
	n := new(int)
	c.OnFrame(func(f fynefigure.Frame) {
		if f.Painted {
			*n++
		}
	})
	return n
}

// A frame asked for while one is waiting is the waiting one: however often a
// chart is redrawn and refreshed before its turn, it is drawn once.
func TestFramesAskedForWhileOneWaitsAreOne(t *testing.T) {
	c, _ := shown(t, fyne.NewSize(400, 250))
	n := renders(c)

	chart.HoldFrame(c)
	c.Redraw()
	c.Refresh()
	c.Redraw()
	if *n != 0 {
		t.Fatalf("%d frames drawn while one was waiting, want none", *n)
	}
	chart.DrawHeld(c)
	if *n == 0 {
		t.Fatal("the waiting frame was not drawn")
	}

	// Once drawn, the next request queues a frame of its own again.
	drawn := *n
	c.Redraw()
	if *n == drawn {
		t.Error("a redraw after the waiting frame drew nothing")
	}
}

// Fyne lays out a hidden widget too. A hidden chart takes the size and draws
// nothing, and showing it draws what it missed.
func TestAHiddenChartDrawsNothingUntilShown(t *testing.T) {
	c, _ := shown(t, fyne.NewSize(400, 250))
	n := painted(c)

	c.Hide()
	c.Resize(fyne.NewSize(600, 300))
	c.Refresh()
	c.Redraw()
	if *n != 0 {
		t.Fatalf("a hidden chart drew %d frames, want none", *n)
	}
	if w, h := c.Live().Size(); w != 600 || h != 300 {
		t.Errorf("the hidden chart is %dx%d, want the 600x300 it was laid out at", w, h)
	}

	c.Show()
	if *n == 0 {
		t.Error("showing the chart did not draw it")
	}
	if b := c.Target().Image().Bounds(); b.Dx() != 600 || b.Dy() != 300 {
		t.Errorf("the frame shown is %v, want 600x300", b)
	}
}

// A resize is one frame, not the size's frame and then another.
func TestAResizeIsOneFrame(t *testing.T) {
	c, _ := shown(t, fyne.NewSize(400, 250))
	n := painted(c)
	c.Resize(fyne.NewSize(500, 300))
	if *n != 1 {
		t.Errorf("the resize painted %d frames, want 1", *n)
	}
}

// ResetView releases the zoom as Autoscale does, but tells nobody: a caller
// that sets the view straight after is not a reader moving the chart.
func TestResetViewReleasesTheZoomAndReportsNothing(t *testing.T) {
	c, _ := shown(t, fyne.NewSize(500, 300))
	before := domainOf(t, c)

	scroll(c, fyne.NewPos(250, 150), 3)
	if got := domainOf(t, c); got == before {
		t.Fatal("the wheel did not zoom the chart")
	}
	var reported int
	c.OnViewChange(func(figure.View) { reported++ })

	c.ResetView()
	if reported != 0 {
		t.Errorf("ResetView reported %d view changes, want none", reported)
	}
	if got := domainOf(t, c); got != before {
		t.Errorf("the domain is %v after ResetView, want the data's %v", got, before)
	}
}
