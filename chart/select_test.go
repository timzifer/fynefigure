package chart_test

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/timzifer/figure"
	"github.com/timzifer/figure/data"
	"github.com/timzifer/figure/geom"
	"github.com/timzifer/figure/scale"
	"github.com/timzifer/fynefigure"
	"github.com/timzifer/fynefigure/chart"
)

// keyedPlot is a scatter whose rows have names, so that a selection made here
// says something a chart over another table can act on.
func keyedPlot() *figure.Plot {
	const n = 24
	x := make([]float64, n)
	y := make([]float64, n)
	id := make([]string, n)
	for i := range n {
		v := float64(i) / 3
		x[i], y[i] = v, math.Sin(v)
		id[i] = string(rune('a' + i%26))
	}
	src := data.NewTable().Float64("t", x).Float64("y", y).String("id", id)

	p := figure.New(figure.Size(400, 250))
	p.X(scale.Linear())
	p.Add(geom.Scatter(src, geom.X("t"), geom.Y("y"), geom.KeyBy("id"), geom.Label("signal")))
	return p
}

// markAt finds a position over a data mark by asking the chart what is under
// each of a grid of points until one of them is a mark rather than furniture.
func markAt(t *testing.T, c *chart.Chart) (fyne.Position, figure.Hit) {
	t.Helper()
	var found figure.Hit
	var ok bool
	c.Plot().On(figure.Hover, func(ev figure.Event) {
		if ev.Found && !ev.Hit.Kind.Guides() && ev.Hit.Row >= 0 {
			found, ok = ev.Hit, true
		}
	})
	for y := float32(4); y < 300; y += 2 {
		for x := float32(4); x < 500; x += 2 {
			ok = false
			chart.PointerOf(c).MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(x, y)}})
			if ok {
				return fyne.NewPos(x, y), found
			}
		}
	}
	t.Fatal("no data mark was found anywhere in the chart")
	return fyne.Position{}, figure.Hit{}
}

// A click on a mark picks the row behind it, and says so in a vocabulary
// another chart understands: the key, not the row number.
func TestClickingAMarkPicksItsRow(t *testing.T) {
	c := chart.New(keyedPlot(), chart.Interactive(true), chart.ThemeFont(false), chart.Select(true))
	shownAt(t, c)

	var told fynefigure.Selection
	calls := 0
	c.OnSelect(func(s fynefigure.Selection) { told, calls = s, calls+1 })

	pos, hit := markAt(t, c)
	click(c, pos)

	sel := c.Selection()
	if len(sel) != 1 {
		t.Fatalf("clicking a mark picked %d rows, want 1", len(sel))
	}
	if sel[0].Row != hit.Row || sel[0].Layer != hit.Layer {
		t.Errorf("picked row %d of layer %d, want row %d of layer %d",
			sel[0].Row, sel[0].Layer, hit.Row, hit.Layer)
	}
	if sel[0].Key == "" {
		t.Error("the picked row carries no key, so nothing can act on it elsewhere")
	}
	if calls != 1 || !told.Equal(sel) {
		t.Errorf("the handler was told %v after %d calls, want %v after 1", told, calls, sel)
	}
	if err := c.Err(); err != nil {
		t.Errorf("drawing the chart with a selection: %v", err)
	}
}

// A chart that was not asked to select does not, which is the same rule the
// legend follows: figure wires nothing by itself and neither does a widget
// nobody asked.
func TestAChartDoesNotSelectUnlessAsked(t *testing.T) {
	// Row tracking on and selection off, so that the test is about the wiring
	// rather than about a hit that reported no row to pick.
	c := chart.New(keyedPlot(), chart.Interactive(true), chart.ThemeFont(false), chart.TrackRows(true))
	shownAt(t, c)

	pos, _ := markAt(t, c)
	click(c, pos)
	if len(c.Selection()) != 0 {
		t.Error("a chart without Select picked a row when a mark was clicked")
	}
}

// A click on nothing clears the selection. It is the gesture every reader
// already knows, and the only way to unpick the last row without finding it
// again.
func TestClickingNothingClearsTheSelection(t *testing.T) {
	c := chart.New(keyedPlot(), chart.Interactive(true), chart.ThemeFont(false), chart.Select(true))
	shownAt(t, c)

	pos, _ := markAt(t, c)
	click(c, pos)
	if len(c.Selection()) != 1 {
		t.Fatal("the row was not picked to begin with")
	}
	click(c, fyne.NewPos(2, 2))
	if got := c.Selection(); len(got) != 0 {
		t.Errorf("clicking outside every mark left %v picked", got)
	}
}

// Without MultiSelect a second click replaces the first, and clicking the row
// that is already picked unpicks it.
func TestOneClickPicksOneRow(t *testing.T) {
	c := chart.New(keyedPlot(), chart.Interactive(true), chart.ThemeFont(false), chart.Select(true))
	shownAt(t, c)

	pos, _ := markAt(t, c)
	click(c, pos)
	first := c.Selection()
	click(c, pos)
	if got := c.Selection(); len(got) != 0 {
		t.Errorf("clicking the picked row again left %v picked", got)
	}
	click(c, pos)
	if got := c.Selection(); !got.Equal(first) {
		t.Errorf("clicking it a third time picked %v, want %v", got, first)
	}
}

// A selection put into a chart is not a selection the reader made, so it
// reports nothing back. It is what stops two linked charts telling each other
// about one click for ever.
func TestSettingASelectionDoesNotReportIt(t *testing.T) {
	c := chart.New(keyedPlot(), chart.Interactive(true), chart.ThemeFont(false), chart.Select(true))
	shownAt(t, c)

	calls := 0
	c.OnSelect(func(fynefigure.Selection) { calls++ })

	c.SetSelection(fynefigure.Selection{{Key: "b", Layer: -1, Row: -1}})
	if calls != 0 {
		t.Errorf("SetSelection called the handler %d times", calls)
	}
	if got := c.Selection(); len(got) != 1 || got[0].Key != "b" {
		t.Errorf("the chart holds %v, want the row it was given", got)
	}
}

// Two charts are linked by one line each way, and the link settles rather than
// running away.
func TestTwoChartsShareOneSelection(t *testing.T) {
	left := chart.New(keyedPlot(), chart.Interactive(true), chart.ThemeFont(false), chart.Select(true))
	right := chart.New(keyedPlot(), chart.Interactive(true), chart.ThemeFont(false), chart.Select(true))
	shownAt(t, left)
	shownAt(t, right)

	left.OnSelect(func(s fynefigure.Selection) { right.SetSelection(s) })
	right.OnSelect(func(s fynefigure.Selection) { left.SetSelection(s) })

	pos, _ := markAt(t, left)
	click(left, pos)

	got := right.Selection()
	if !got.Equal(left.Selection()) {
		t.Fatalf("the right chart holds %v, the left %v", got, left.Selection())
	}
	if len(got) != 1 || got[0].Key == "" {
		t.Errorf("the row crossed as %v, which names no key", got)
	}
	if err := right.Err(); err != nil {
		t.Errorf("drawing the linked chart: %v", err)
	}
}

// A selection draws. The rings are figure's own overlay, over the finished
// chart, so the picture changes and the chart underneath does not.
func TestASelectionIsDrawn(t *testing.T) {
	c := chart.New(keyedPlot(), chart.Interactive(true), chart.ThemeFont(false), chart.Select(true))
	shownAt(t, c)

	pos, _ := markAt(t, c)
	before := pixels(t, c)
	click(c, pos)
	after := pixels(t, c)

	if before == after {
		t.Error("picking a row changed nothing on screen")
	}
	c.SetSelection(nil)
	if got := pixels(t, c); got != before {
		t.Error("clearing the selection did not put the chart back")
	}
}

// pixels is a cheap digest of what the chart is showing, for a test that only
// needs to know whether it changed.
func pixels(t *testing.T, c *chart.Chart) uint64 {
	t.Helper()
	img := c.Target().Image()
	if img == nil {
		t.Fatal("the chart has no pixels")
	}
	var sum uint64
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 3 {
		for x := b.Min.X; x < b.Max.X; x += 3 {
			r, g, bl, a := img.At(x, y).RGBA()
			sum = sum*31 + uint64(r) + uint64(g)<<8 + uint64(bl)<<16 + uint64(a)<<24
		}
	}
	return sum
}

// trackedPlot is a panel and a band under it, each drawing one layer of three
// rectangles over the same stretches of time.
//
// It is the shape a gantt strip under a curve has, and the shape that shows a
// selection resolved against the wrong panel: figure numbers layers per panel,
// so both hold a layer 0 with a row 1, and they are different rows of
// different tables.
func trackedPlot() *figure.Plot {
	spans := func() *data.Table {
		return data.NewTable().
			Float64("start", []float64{0, 4, 8}).
			Float64("end", []float64{3, 7, 11}).
			Float64("low", []float64{0, 0, 0}).
			Float64("high", []float64{1, 1, 1})
	}
	rect := func(src *data.Table) geom.Geom {
		return geom.Rect(src, geom.X("start"), geom.X2("end"), geom.Y("low"), geom.Y2("high"))
	}

	p := figure.New(figure.Size(400, 250))
	p.X(scale.Linear())
	p.Add(rect(spans()))
	p.Track(figure.Bottom, figure.TrackSize(60),
		figure.TrackScale(scale.Linear(scale.Domain(0, 1)))).
		Add(rect(spans()))
	return p
}

// A row picked in a band is ringed in that band and nowhere else. The panel
// above has a layer and a row with the same numbers, and ringing it too marks
// something the reader did not pick.
func TestASelectionInABandStaysInTheBand(t *testing.T) {
	c := chart.New(trackedPlot(), chart.Interactive(true), chart.ThemeFont(false), chart.Select(true))
	shownAt(t, c)

	var hit figure.Hit
	found := false
	c.Plot().On(figure.Hover, func(ev figure.Event) {
		if ev.Found && !ev.Hit.Kind.Guides() && ev.Hit.Panel == 1 && ev.Hit.Row >= 0 {
			hit, found = ev.Hit, true
		}
	})
	var pos fyne.Position
	for y := float32(296); y > 4 && !found; y -= 2 {
		for x := float32(4); x < 500 && !found; x += 2 {
			pos = fyne.NewPos(x, y)
			chart.PointerOf(c).MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: pos}})
		}
	}
	if !found {
		t.Fatal("no mark was found in the band")
	}

	img := c.Target().Image()
	if img == nil {
		t.Fatal("the chart has no pixels")
	}
	ratio := float32(img.Bounds().Dy()) / c.Size().Height
	// The panel is everything above the band. The pointer may be anywhere in
	// the band, which is 60 units tall, and the ring round the band's row
	// reaches a few units past its middle — so a band's height and a ring
	// above the pointer is clear of both.
	above := int((pos.Y - 70) * ratio)
	before := rowsDigest(t, c, above)

	click(c, pos)
	sel := c.Selection()
	if len(sel) != 1 || sel[0].View != 1 || sel[0].Row != hit.Row {
		t.Fatalf("clicking the band picked %+v, want row %d of panel 1", sel, hit.Row)
	}
	if rowsDigest(t, c, above) != before {
		t.Error("a row picked in the band was ringed in the panel above it as well")
	}
}

// rowsDigest is [pixels] over the rows above y alone.
func rowsDigest(t *testing.T, c *chart.Chart, y int) uint64 {
	t.Helper()
	img := c.Target().Image()
	if img == nil {
		t.Fatal("the chart has no pixels")
	}
	var sum uint64
	b := img.Bounds()
	for py := b.Min.Y; py < y && py < b.Max.Y; py++ {
		for px := b.Min.X; px < b.Max.X; px++ {
			r, g, bl, a := img.At(px, py).RGBA()
			sum = sum*31 + uint64(r) + uint64(g)<<8 + uint64(bl)<<16 + uint64(a)<<24
		}
	}
	return sum
}
