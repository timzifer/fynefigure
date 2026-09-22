// Command demo is figure's kitchensink: every chart figure draws, in a Fyne
// window, with what each one costs beside it.
//
//	cd cmd/demo
//	go run .
//	go run . -plot surface -cpu
//	go run . -list
//
// It is a module of its own so that it can import the GPU tier, which is
// nested inside the widget's module and so outside its graph. See go.mod
// beside this file, and gpu.go for the tier itself, which is not every
// platform's to have.
//
// The tree on the left is the catalogue: every mark, coordinate system and
// layout figure has, grouped the way docs/chart-types.md groups them, and the
// interaction the widgets add on top — linked charts, a live stream, a
// transition, several cameras on one scene. Pick one and it is built from
// scratch and put on stage.
//
// The panel on the right is what it costs. Every call that draws into a
// chart's surface is timed through fynefigure.Target.OnFrame — resize,
// redraw, stream and pointer alike:
//
//   - Build is making the plot and its widgets, and First frame the time from
//     there to the first painted frame.
//   - Draw is what a painted frame costs to lay out and rasterize, and Frame
//     rate how many were painted in the last second.
//   - Hit test is what a call that painted nothing cost — a pointer moving
//     over a chart nobody is zooming, mostly.
//   - Input → paint is how long a pointer event waited for the frame that
//     answered it, the widget's pacing included. It is the roundtrip a reader
//     feels under a drag.
//
// Benchmark drives sixty frames through the chart and times each from the
// call to the painted frame.
//
// Above the numbers are the switches. GPU puts the rasterizer on figure's GPU
// tier or takes it off, at run time: the chart on stage is closed, the tier is
// switched with gpu.Disable or gpu.Enable, and the chart is built again,
// because a rasterizer keeps GPU state from its first frame and cannot be
// switched under a live one. Interactive turns the pointer off and on without
// a rebuild. Half resolution draws a flat chart coarse while it is dragged and
// sharpens it afterwards — the frame rate is the thing to watch.
//
// The single charts are in package plots, ported from figure's gallery and
// examples. The ones that need more than one widget, or a clock, are in
// showcase.go, cameras.go and contour.go.
package main

import (
	"flag"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	first := flag.String("plot", "", "id of the chart to show first (see -list); the newest when empty")
	cpu := flag.Bool("cpu", false, "start on the CPU rasterizer rather than the GPU tier")
	list := flag.Bool("list", false, "list the charts and exit")
	flag.Parse()

	cat := catalog()
	if *list {
		for _, e := range cat.entries {
			fmt.Printf("%-20s %s › %s\n", e.id, e.group, e.title)
		}
		return
	}

	// The tier is given back on the way out, which is gg's advice for the
	// device it holds. The panel's switch uses Disable and Enable instead,
	// which keep it.
	defer gpuTier.Close()
	if *cpu {
		gpuTier.Disable()
	}

	a := app.New()
	w := a.NewWindow("figure — kitchensink")
	w.Resize(fyne.NewSize(1400, 820))
	k := newKitchen(w, cat)
	w.SetContent(k.content())
	// The panel's numbers move on a ticker, started here rather than by
	// newKitchen: under Fyne's test driver fyne.Do runs where it is called,
	// so a ticker in a test would refresh the panel beside the test itself.
	go k.tick()
	w.SetOnClosed(k.close)
	k.open(*first)
	w.ShowAndRun()
}
