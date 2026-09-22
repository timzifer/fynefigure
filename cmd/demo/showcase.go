package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/timzifer/figure"
	"github.com/timzifer/figure/data"
	"github.com/timzifer/figure/geom"
	"github.com/timzifer/figure/interact"
	"github.com/timzifer/figure/ir"
	"github.com/timzifer/figure/palette"
	"github.com/timzifer/figure/scale"
	"github.com/timzifer/figure/three"
	"github.com/timzifer/fynefigure"
	"github.com/timzifer/fynefigure/chart"
	"github.com/timzifer/fynefigure/cmd/demo/plots"
	"github.com/timzifer/fynefigure/orbit"
)

// groupInteraction is the tree's last group: what the widgets add to figure
// rather than what figure draws.
const groupInteraction = "Interaction"

// showcase is the part of the catalogue that is more than one chart in one
// widget: charts linked to each other, a clock driving one, several cameras.
func showcase() []*entry {
	return []*entry{
		{
			id: "contour-linked", group: plots.GroupFields, title: "Plan and shape, linked",
			note:  "One field read two ways off one colour scale. Click a cell in either chart to ring it in both.",
			build: contourShow,
		},
		{
			id: "four-cameras", group: plots.Group3D, title: "Four cameras, one widget",
			note:  "One scene from four cameras. Drag a view to turn it, click a point to mark it in all four, double click to go home.",
			build: scenesShow,
		},
		{
			id: "four-widgets", group: plots.Group3D, title: "Four cameras, four widgets",
			note:  "The same cameras as four widgets, glued together by the program: turns, picks and a coarser draw for the ones not under the pointer.",
			build: camerasShow,
		},
		{
			id: "linked", group: groupInteraction, title: "Linked, selecting, crosshair",
			note:  "Drag a rectangle over the top chart to select rows; pan or zoom either and the other follows. Click a legend row to hide a series.",
			build: interactShow,
		},
		{
			id: "live", group: groupInteraction, title: "Live stream",
			note:  "A producer appends on its own goroutine and the chart repaints on a timer. The axis follows the window, so a drag has nothing behind the tip to pan to.",
			build: liveShow,
		},
		{
			id: "transition", group: groupInteraction, title: "Transition",
			note:  "Keyed rows moving between states: a bar that stays moves, a new one grows out of the baseline, a leaving one sinks back into it, and the labels count.",
			build: transitionShow,
		},
	}
}

// interactShow is two charts of the same table, linked, with everything a
// pointer can be made to mean switched on. None of it is on by default — a
// legend that always toggled would be wrong for one that selects rather than
// filters — so this is also the list of the switches.
func interactShow(e env) (*view, error) {
	src := signal(2000)
	top := chart.New(twoSeries(src, "Signal — drag to select"),
		e.chartOpts(chart.LegendToggle(true), chart.DragMode(figure.DragSelects))...)
	bottom := chart.New(twoSeries(src, "The same rows, linked"),
		e.chartOpts(chart.LegendToggle(true))...)

	// The link, one line each way. It does not loop: SetView is not a reader
	// moving anything, and reports nothing back.
	top.OnViewChange(func(v figure.View) { _ = bottom.SetView(v) })
	bottom.OnViewChange(func(v figure.View) { _ = top.SetView(v) })

	// A crosshair is installed once and then moved: the overlay is a pointer
	// whose fields the handler writes.
	cross := &figure.Crosshair{}
	top.Overlay(cross)
	top.Plot().On(figure.Hover, func(ev figure.Event) {
		cross.At, cross.Show = ev.Hit.At, ev.Found && !ev.Hit.Kind.Guides()
		e.readout(ev)
	})
	bottom.Plot().On(figure.Hover, e.readout)
	// A selection is one event per layer under the rectangle. What it means is
	// the caller's — figure counts the rows and stops there.
	top.Plot().On(figure.Select, func(ev figure.Event) {
		e.status(fmt.Sprintf("%s: %d rows selected", ev.Hit.Series, len(ev.Rows)))
	})

	mode := widget.NewSelect([]string{"Select", "Zoom to band", "Pan"}, func(s string) {
		switch s {
		case "Select":
			top.SetDragMode(figure.DragSelects)
		case "Zoom to band":
			top.SetDragMode(figure.DragZooms)
		default:
			top.SetDragMode(figure.DragPans)
		}
	})
	mode.SetSelected("Select")
	showAll := widget.NewButton("Show every series", func() {
		if err := top.ShowAllLayers(); err != nil {
			e.status("showing: " + err.Error())
			return
		}
		_ = bottom.ShowAllLayers()
	})

	controls := container.NewHBox(widget.NewLabel("Drag on top:"), mode, showAll)
	charts := container.NewGridWithRows(2, top, bottom)
	return &view{
		obj:   container.NewBorder(nil, controls, nil, nil, charts),
		flats: []*chart.Chart{top, bottom},
	}, nil
}

// liveShow is a stream and a chart. The producer and the repaint both start
// when the view goes on stage and stop when it leaves, so a chart nobody is
// looking at costs nothing.
func liveShow(e env) (*view, error) {
	const window = 600

	st := data.NewStream("t", "y").Window(window)
	for i := range window {
		t := float64(i)
		if err := st.Append(t, throughput(t)); err != nil {
			return nil, fmt.Errorf("seeding the stream: %w", err)
		}
	}

	p := figure.New(
		figure.Responsive(true),
		figure.Size(900, 480),
		figure.Title("Live throughput"),
		figure.YTitle("rows/s"),
	)
	// A pinned Y axis is what makes a live chart readable: one that rescales
	// itself every frame turns every change into a redraw of everything.
	p.Y(scale.Linear(scale.Domain(0, 120)))
	p.Add(
		geom.Line(st.Source(), geom.X("t"), geom.Y("y"),
			geom.Color(palette.SkyBlue), geom.Label("throughput")),
		geom.HLine(100, geom.Label("capacity"), geom.Dash(6, 4)),
	)
	p.On(figure.Hover, e.readout)

	// The x axis follows the window — chart.Follow, the default — and a drag is
	// ignored, because the rows behind the tip have been dropped.
	c := chart.New(p, e.chartOpts()...)
	c.Stream(st)

	last := float64(window)
	start := func() func() {
		done := make(chan struct{})
		// The producer appends and never reads. The chart freezes a snapshot
		// between frames, so it never sees the stream half-written.
		go func() {
			t := time.NewTicker(40 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-done:
					return
				case <-t.C:
					last++
					if err := st.Append(last, throughput(last)); err != nil {
						log.Println("append:", err)
						return
					}
				}
			}
		}()
		stopDrawing := c.Animate(50 * time.Millisecond)
		return func() {
			stopDrawing()
			close(done)
		}
	}
	return &view{obj: c, flats: []*chart.Chart{c}, start: start}, nil
}

// transitionShow cycles a chart through three states of one table, keyed by
// language. The clock is the widget's: Chart.Play advances the transition by
// the wall clock, so it takes as long as it says on any machine.
func transitionShow(e env) (*view, error) {
	states := []*data.Table{shares(0), shares(1), shares(2)}
	first, err := shareTween(states[0], states[1])
	if err != nil {
		return nil, err
	}

	p := figure.New(
		figure.Responsive(true),
		figure.Size(760, 420),
		figure.Title("Share, moving"),
		figure.XTitle("slot"),
		figure.YTitle("share"),
	)
	// Both axes are pinned: an axis that rescaled every frame would make the
	// two ends of a movement impossible to compare by eye.
	p.X(scale.Linear(scale.Domain(-0.6, 2.6)))
	p.Y(scale.Linear(scale.Domain(0, 50)))
	p.SetLayers(shareLayers(first)...)
	p.On(figure.Hover, e.readout)
	c := chart.New(p, e.chartOpts()...)

	// Everything below runs on Fyne's goroutine: the timers post there.
	var (
		step    int
		stopped bool
		halt    func()
		wait    *time.Timer
	)
	later := func(d time.Duration, fn func()) {
		wait = time.AfterFunc(d, func() { fyne.Do(fn) })
	}
	var play func()
	play = func() {
		if stopped {
			return
		}
		from, to := states[step%len(states)], states[(step+1)%len(states)]
		tw, err := shareTween(from, to)
		if err != nil {
			e.status("transition: " + err.Error())
			return
		}
		// A tween is a source, so the next movement is new layers over the
		// same plot, resolved again.
		p.SetLayers(shareLayers(tw)...)
		if err := c.Rebuild(); err != nil {
			e.status("transition: " + err.Error())
			return
		}
		tr, err := c.Transition(tw)
		if err != nil {
			e.status("transition: " + err.Error())
			return
		}
		if tr == nil {
			// Not laid out yet: nothing to move until there is a chart on screen.
			later(200*time.Millisecond, play)
			return
		}
		step++
		halt = c.Play(tr.Over(900*time.Millisecond).Ease(figure.EaseInOut), func() {
			later(900*time.Millisecond, play)
		})
	}
	start := func() func() {
		later(400*time.Millisecond, play)
		return func() {
			stopped = true
			if wait != nil {
				wait.Stop()
			}
			if halt != nil {
				halt()
			}
		}
	}
	return &view{obj: c, flats: []*chart.Chart{c}, start: start}, nil
}

// shares is state i of the transition. Two languages hold their place from
// one state to the next, one arrives and one leaves, which is the shape that
// makes enter, update and exit all visible in one movement.
func shares(i int) *data.Table {
	t := figure.NewTable()
	switch i {
	case 0:
		t.String("lang", []string{"go", "rust", "perl"})
		t.Float64("share", []float64{40, 25, 18})
	case 1:
		t.String("lang", []string{"go", "rust", "zig"})
		t.Float64("share", []float64{30, 45, 22})
	default:
		t.String("lang", []string{"go", "python", "zig"})
		t.Float64("share", []float64{36, 28, 31})
	}
	return t.Float64("slot", []float64{0, 1, 2})
}

// shareTween is the movement from one state to the next. EnterFrom and ExitTo
// are in data space — a bar that is not there yet has height zero — and slot
// is held, because halfway between two slots is not a slot.
func shareTween(a, b *data.Table) (*data.Tween, error) {
	return data.NewTween(a, b, "lang",
		data.EnterFrom("share", 0),
		data.ExitTo("share", 0),
		data.Round("share", 0),
		data.Hold("slot"),
	)
}

func shareLayers(tw *data.Tween) []geom.Geom {
	return []geom.Geom{
		geom.Bar(tw.Source(), geom.X("slot"), geom.Y("share"),
			geom.KeyBy("lang"), geom.Color(palette.Blue), geom.BarWidth(0.6), geom.Label("share")),
		// The label that counts: it reads the same blended column the bars do.
		geom.Text(tw.Source(), geom.X("slot"), geom.Y("share"),
			geom.TextBy("share"), geom.Align(ir.AlignCenter, ir.AlignBottom)),
		geom.Text(tw.Source(), geom.X("slot"), geom.Y("share"),
			geom.TextBy("lang"), geom.Align(ir.AlignCenter, ir.AlignTop), geom.Color(palette.White)),
	}
}

// scenesShow is one surface seen from four cameras: the three-quarter view an
// author designs at, and the plan and two elevations an engineering drawing
// has always had. The scene is built once and its scales trained once; each
// view is only a camera on it.
func scenesShow(e env) (*view, error) {
	labels := []string{"three-quarter", "plan", "front", "side"}
	p := three.New(three.Size(900, 480), three.Title("Response surface"), three.Columns(2)).
		Scene(saddleScene()).
		Add(
			three.View{Camera: three.Home(), Label: labels[0]},
			three.View{Camera: three.LookAt(three.Elevation(1.45)), Label: labels[1]},
			three.View{Camera: three.LookAt(three.Azimuth(0), three.Elevation(0.02)), Label: labels[2]},
			three.View{Camera: three.LookAt(three.Azimuth(-math.Pi/2), three.Elevation(0.02)), Label: labels[3]},
		)
	c := orbit.New(p, e.orbitOpts(orbit.Select(true))...)

	c.OnCamera(func(i int, cam three.Camera) {
		e.status(fmt.Sprintf("%s   azimuth %.0f°   elevation %.0f°   zoom %.2f",
			labels[i], degrees(cam.Azimuth()), degrees(cam.Elevation()), cam.Zoom()))
	})
	c.OnHover(func(h interact.Hit, found bool) {
		if !found {
			e.status("")
			return
		}
		e.status(fmt.Sprintf("%s   %s   row %d", labels[h.Panel], h.Series, h.Row))
	})
	// One click, four rings: a point picked in the plan is the same point in
	// the three-quarter view and both profiles.
	//
	// The handler runs with the chart held, so it must not call back into it —
	// c.ViewCount() here would deadlock against the click still being
	// delivered. Everything it needs is read before it is registered.
	views := len(labels)
	c.OnSelect(func(sel fynefigure.Selection) {
		if len(sel) == 0 {
			e.status("")
			return
		}
		e.status(fmt.Sprintf("row %d picked in the %s view, and marked in all %d",
			sel[0].Row, labels[sel[0].View], views))
	})

	home := widget.NewButton("Home", func() {
		if err := c.Home(); err != nil {
			e.status("home: " + err.Error())
		}
	})
	clear := widget.NewButton("Clear selection", func() { c.SetSelection(nil) })
	return &view{
		obj:    container.NewBorder(nil, container.NewHBox(home, clear), nil, nil, c),
		orbits: []*orbit.Chart{c},
	}, nil
}

// saddle is a response with a ridge one way and a trough the other: the shape
// whose point is that a heatmap of it looks symmetric and it is not.
func saddle() figure.Source {
	const n = 28
	xs := make([]float64, 0, n*n)
	ys := make([]float64, 0, n*n)
	zs := make([]float64, 0, n*n)
	for j := range n {
		for i := range n {
			x := -3 + 6*float64(i)/(n-1)
			y := -3 + 6*float64(j)/(n-1)
			xs, ys = append(xs, x), append(ys, y)
			zs = append(zs, math.Sin(x)*math.Cos(y)*1.4+0.25*x)
		}
	}
	return figure.NewTable().Float64("x", xs).Float64("y", ys).Float64("z", zs)
}

// twoSeries is two named series over one table, which is what gives a chart a
// legend to click.
func twoSeries(src figure.Source, title string) *figure.Plot {
	p := figure.New(
		figure.Responsive(true),
		figure.Size(900, 240),
		figure.Title(title),
		figure.XTitle("t"),
		figure.YTitle("amplitude"),
	)
	p.Add(
		geom.Line(src, geom.X("t"), geom.Y("signal"),
			geom.Color(palette.SkyBlue), geom.Label("signal")),
		geom.Line(src, geom.X("t"), geom.Y("carrier"),
			geom.Color(palette.Vermilion), geom.Label("carrier")),
	)
	return p
}

func signal(n int) figure.Source {
	x := make([]float64, n)
	y := make([]float64, n)
	carrier := make([]float64, n)
	for i := range n {
		t := float64(i) / 40
		x[i] = t
		y[i] = math.Sin(t) + 0.35*math.Sin(7.3*t) + 0.12*math.Sin(31*t)
		carrier[i] = 0.8 * math.Cos(t/1.7)
	}
	return figure.Float64Columns(map[string][]float64{"t": x, "signal": y, "carrier": carrier})
}

func throughput(t float64) float64 {
	return 70 + 25*math.Sin(t/50) + 8*rand.Float64()
}
