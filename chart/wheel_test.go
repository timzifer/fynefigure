package chart_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	"github.com/timzifer/fynefigure/chart"
)

// Through the canvas rather than the layer: what matters is where Fyne sends
// the wheel. Over a chart that may not zoom it goes to the scroll container
// the chart stands in, and over one that may, to the chart.
func TestTheWheelOverAFixedChartScrollsTheList(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fixed  bool
		scroll bool
	}{
		{"PanZoom off", true, true},
		{"PanZoom on", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := chart.New(keyedPlot(), chart.Interactive(true), chart.ThemeFont(false),
				chart.MinSize(400, 300), chart.PanZoom(!tc.fixed))
			list := container.NewVScroll(container.NewVBox(c, fixedHeight(900)))

			win := test.NewTempWindow(t, list)
			win.Resize(fyne.NewSize(500, 400))
			c.Redraw()

			test.Scroll(win.Canvas(), fyne.NewPos(250, 150), 0, -40)

			if scrolled := list.Offset.Y > 0; scrolled != tc.scroll {
				t.Errorf("list offset %v after a wheel over the chart; scrolled = %v, want %v",
					list.Offset.Y, scrolled, tc.scroll)
			}
		})
	}
}

// fixedHeight is a spacer that makes the list taller than its window.
func fixedHeight(h float32) fyne.CanvasObject {
	r := container.NewWithoutLayout()
	r.Resize(fyne.NewSize(1, h))
	return container.New(&minLayout{h}, r)
}

type minLayout struct{ h float32 }

func (m *minLayout) Layout([]fyne.CanvasObject, fyne.Size) {}
func (m *minLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(1, m.h) }
