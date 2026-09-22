package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// benchFrames is how many frames a benchmark draws: enough for a p95 to mean
// something, few enough to finish in a second on the GPU.
const benchFrames = 60

const benchHint = "Pans the chart a pixel at a time — turns a scene, redraws a still — and times each call to its painted frame."

// The rows of the timings card, in order.
const (
	rowBuild = iota
	rowFirst
	rowDraw
	rowDrawAvg
	rowDrawMax
	rowFPS
	rowProbe
	rowTrip
	rowTripAvg
	rowFrames
	rowSurface
)

var rowNames = [...]string{
	rowBuild:   "Build",
	rowFirst:   "First frame",
	rowDraw:    "Draw",
	rowDrawAvg: "   avg · p95",
	rowDrawMax: "   max",
	rowFPS:     "Frame rate",
	rowProbe:   "Hit test",
	rowTrip:    "Input → paint",
	rowTripAvg: "   avg · p95",
	rowFrames:  "Frames",
	rowSurface: "Surface",
}

// panel is the right-hand side: the switches, what the chart on stage costs,
// and what can be done to it.
type panel struct {
	k *kitchen

	gpuCheck, pointer, coarse *widget.Check
	backend                   *widget.Label

	values   []*widget.Label
	errLabel *widget.Label

	bench, fit, export, reset *widget.Button
	benchOut                  *widget.Label
	benching                  bool

	// quiet is set while a check is written from code, so that its callback
	// does not take it for the reader.
	quiet bool

	obj fyne.CanvasObject
}

func newPanel(k *kitchen) *panel {
	p := &panel{k: k}

	p.gpuCheck = widget.NewCheck("GPU", func(on bool) {
		if p.quiet {
			return
		}
		if !k.setGPU(on) {
			p.set(p.gpuCheck, false)
			k.say("The GPU tier could not be brought back, so the chart is drawn on the CPU.")
		}
		p.refresh()
	})
	p.set(p.gpuCheck, gpuTier.Enabled())
	if !gpuTier.Available() {
		p.gpuCheck.Disable()
	}
	p.pointer = widget.NewCheck("Interactive", func(on bool) {
		if !p.quiet {
			k.setInteractive(on)
		}
	})
	p.set(p.pointer, k.interactive)
	p.coarse = widget.NewCheck("Half resolution while dragging", func(on bool) {
		if !p.quiet {
			k.setDetail(on)
		}
	})
	p.set(p.coarse, k.detail)
	p.backend = widget.NewLabel("")
	p.backend.Wrapping = fyne.TextWrapWord

	form := container.New(layout.NewFormLayout())
	for _, name := range rowNames {
		v := widget.NewLabelWithStyle("—", fyne.TextAlignTrailing, fyne.TextStyle{Monospace: true})
		p.values = append(p.values, v)
		form.Add(widget.NewLabel(name))
		form.Add(v)
	}
	p.errLabel = widget.NewLabel("")
	p.errLabel.Wrapping = fyne.TextWrapWord
	p.errLabel.Importance = widget.DangerImportance
	p.errLabel.Hide()

	p.bench = widget.NewButtonWithIcon(fmt.Sprintf("Benchmark %d frames", benchFrames),
		theme.MediaPlayIcon(), p.runBench)
	p.benchOut = widget.NewLabel(benchHint)
	p.benchOut.Wrapping = fyne.TextWrapWord
	p.fit = widget.NewButtonWithIcon("Autoscale", theme.ZoomFitIcon(), func() {
		if k.v != nil {
			if err := k.v.fit(); err != nil {
				k.say(err.Error())
			}
		}
	})
	p.export = widget.NewButtonWithIcon("Export PNG", theme.DocumentSaveIcon(), p.exportPNG)
	p.reset = widget.NewButtonWithIcon("Reset timings", theme.ViewRefreshIcon(), func() {
		k.rec.clear()
		p.refresh()
	})

	p.obj = container.NewVBox(
		widget.NewCard("Rendering", "", container.NewVBox(p.gpuCheck, p.pointer, p.coarse, p.backend)),
		widget.NewCard("Timings", "", container.NewVBox(form, p.errLabel)),
		widget.NewCard("Benchmark", "", container.NewVBox(p.bench, p.benchOut)),
		widget.NewCard("Chart", "", container.NewVBox(p.fit, p.export, p.reset)),
	)
	return p
}

// set writes a check without running its callback.
func (p *panel) set(c *widget.Check, on bool) {
	p.quiet = true
	c.SetChecked(on)
	p.quiet = false
}

// shown is told a new entry is on stage.
func (p *panel) shown() {
	v := p.k.v
	enable(p.export, v != nil)
	enable(p.fit, v != nil && (len(v.flats) > 0 || len(v.orbits) > 0))
	if v != nil && len(v.flats) == 0 && len(v.orbits) > 0 {
		p.fit.SetText("Home")
	} else {
		p.fit.SetText("Autoscale")
	}
	enable(p.bench, v != nil && v.benchable() && !p.benching)
	if !p.benching {
		p.benchOut.SetText(benchHint)
	}
	p.refresh()
}

// refresh copies the recorder into the labels.
func (p *panel) refresh() {
	s := p.k.rec.snapshot()
	switch {
	case s.built && s.build == 0:
		// Go's clock on Windows ticks every half millisecond or so, and a
		// plot built between two ticks measures as nothing.
		p.show(rowBuild, "< 0.5 ms")
	default:
		p.show(rowBuild, ms(s.build))
	}
	p.show(rowFirst, ms(s.first))
	p.show(rowDraw, ms(s.draw.last))
	p.show(rowDrawAvg, ms2(s.draw.avg, s.draw.p95))
	p.show(rowDrawMax, ms(s.draw.max))
	p.show(rowFPS, fmt.Sprintf("%.0f fps", s.fps))
	p.show(rowProbe, ms(s.probe.avg))
	p.show(rowTrip, ms(s.trip.last))
	p.show(rowTripAvg, ms2(s.trip.avg, s.trip.p95))
	p.show(rowFrames, fmt.Sprintf("%d · %d idle", s.painted, s.idle))
	if s.w > 0 {
		p.show(rowSurface, fmt.Sprintf("%d×%d @%.3g", s.w, s.h, s.dpr))
	} else {
		p.show(rowSurface, "—")
	}

	var tier string
	switch {
	case gpuTier.Enabled():
		tier = "Drawing on the GPU tier."
	case gpuTier.Available():
		tier = "Drawing on the CPU rasterizer; the GPU tier is off."
	default:
		tier = "Drawing on the CPU rasterizer: no GPU device answered."
	}
	if p.backend.Text != tier {
		p.backend.SetText(tier)
	}

	if err := p.k.err(); err != nil {
		if p.errLabel.Text != err.Error() {
			p.errLabel.SetText(err.Error())
		}
		p.errLabel.Show()
	} else {
		p.errLabel.Hide()
	}
}

func (p *panel) show(row int, s string) {
	if p.values[row].Text != s {
		p.values[row].SetText(s)
	}
}

// runBench draws benchFrames frames through the chart on stage, each one from
// Fyne's goroutine as an event would be, and times each from the call to the
// frame it painted. The wait happens on a goroutine of its own, so the window
// keeps drawing while it runs.
func (p *panel) runBench() {
	k := p.k
	v := k.v
	if v == nil || !v.benchable() || p.benching {
		return
	}
	p.benching = true
	p.bench.Disable()
	p.benchOut.SetText("Running…")
	painted := k.rec.wait()
	v.aim()

	go func() {
		defer k.rec.unwait()
		var ds []time.Duration
		missed := 0
		for i := range benchFrames {
			alive := true
			var t0 time.Time
			select { // a frame something else painted is not this one's
			case <-painted:
			default:
			}
			fyne.DoAndWait(func() {
				if k.v != v {
					alive = false
					return
				}
				t0 = time.Now()
				v.nudge(i)
			})
			if !alive {
				break
			}
			if at, ok := next(painted, t0); ok {
				ds = append(ds, at.Sub(t0))
			} else if missed++; missed > 3 {
				break
			}
		}
		fyne.Do(func() { p.benchDone(ds, missed) })
	}()
}

// next waits for the first frame painted at or after t0.
func next(painted <-chan time.Time, t0 time.Time) (time.Time, bool) {
	timeout := time.After(2 * time.Second)
	for {
		select {
		case at := <-painted:
			if !at.Before(t0) {
				return at, true
			}
		case <-timeout:
			return time.Time{}, false
		}
	}
}

func (p *panel) benchDone(ds []time.Duration, missed int) {
	p.benching = false
	enable(p.bench, p.k.v != nil && p.k.v.benchable())
	if len(ds) == 0 {
		p.benchOut.SetText("No frame was painted. A chart following a stream does not pan, " +
			"and switching charts stops the run.")
		return
	}
	s := summarize(ds, ds[len(ds)-1])
	text := fmt.Sprintf("%d frames\nmin %s · avg %s\np95 %s · max %s\n≈ %.0f frames a second",
		s.n, ms(s.min), ms(s.avg), ms(s.p95), ms(s.max), float64(time.Second)/float64(s.avg))
	if missed > 0 {
		text += fmt.Sprintf("\n%d calls painted nothing within 2 s", missed)
	}
	p.benchOut.SetText(text)
}

func (p *panel) exportPNG() {
	k := p.k
	if k.v == nil || k.cur == nil {
		return
	}
	path := filepath.Join(os.TempDir(), "figure-"+k.cur.id+".png")
	if err := k.v.export(path); err != nil {
		k.say("export: " + err.Error())
		return
	}
	k.say("wrote " + path)
}

func enable(w fyne.Disableable, on bool) {
	if on {
		w.Enable()
	} else {
		w.Disable()
	}
}

func ms(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.2f ms", float64(d)/float64(time.Millisecond))
}

func ms2(a, b time.Duration) string {
	if a <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f · %.1f ms", float64(a)/float64(time.Millisecond), float64(b)/float64(time.Millisecond))
}
