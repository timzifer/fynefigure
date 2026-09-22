package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const hint = "Hover the chart to read it. Drag to pan, turn the wheel to zoom, double click to go back."

// kitchen is the window: the tree, the stage, the panel, and the one chart on
// stage at a time.
//
// Everything here runs on Fyne's goroutine — the tree's and the panel's
// callbacks, and the ticker, which posts there — so none of it is locked.
type kitchen struct {
	w   fyne.Window
	cat *catalogue
	rec *recorder

	// The panel's switches, which every chart is built with.
	interactive, detail bool

	cur      *entry
	node     widget.TreeNodeID // the tree node cur was selected by
	v        *view
	stop     func()
	buildErr error

	tree                *widget.Tree
	stage               *fyne.Container
	title, note, status *widget.Label
	panel               *panel
	done                chan struct{}
}

func newKitchen(w fyne.Window, cat *catalogue) *kitchen {
	k := &kitchen{w: w, cat: cat, rec: newRecorder(), interactive: true, done: make(chan struct{})}
	k.title = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	k.note = widget.NewLabel("")
	k.note.Wrapping = fyne.TextWrapWord
	k.status = widget.NewLabel(hint)
	k.status.Truncation = fyne.TextTruncateEllipsis
	k.stage = container.NewStack()
	k.tree = k.newTree()
	k.panel = newPanel(k)
	return k
}

// content is the window's layout: the tree, then the stage, then the panel.
func (k *kitchen) content() fyne.CanvasObject {
	center := container.NewBorder(container.NewVBox(k.title, k.note), k.status, nil, nil, k.stage)
	inner := container.NewHSplit(center, container.NewVScroll(k.panel.obj))
	inner.Offset = 0.76
	outer := container.NewHSplit(k.tree, inner)
	outer.Offset = 0.18
	return outer
}

func (k *kitchen) newTree() *widget.Tree {
	t := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID { return k.cat.children[id] },
		k.cat.isBranch,
		func(branch bool) fyne.CanvasObject {
			if branch {
				return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			}
			return widget.NewLabel("")
		},
		func(id widget.TreeNodeID, _ bool, o fyne.CanvasObject) { o.(*widget.Label).SetText(k.cat.label(id)) },
	)
	t.OnSelected = func(id widget.TreeNodeID) {
		if e, ok := k.cat.entry(id); ok {
			k.node = id
			if e != k.cur {
				k.show(e)
			}
			return
		}
		// A group is not a chart: open or close it, and keep the chart that is
		// on stage selected.
		t.ToggleBranch(id)
		if k.node != "" {
			t.Select(k.node)
		} else {
			t.Unselect(id)
		}
	}
	// Closed but for New: the tree starts as a short list of what changed,
	// and every group is a click away.
	t.OpenBranch(newNode)
	return t
}

// open selects an entry by id, which puts it on stage. A chart that is not
// new opens its group, since a closed branch hides the selection.
func (k *kitchen) open(id string) {
	e := k.cat.find(id)
	node := k.cat.node(e)
	if node == e.id {
		k.tree.OpenBranch(groupPrefix + e.group)
	}
	k.tree.Select(node)
	k.tree.ScrollTo(node)
}

// env is what the next chart is built with.
func (k *kitchen) env() env {
	return env{interactive: k.interactive, detail: k.detail, status: k.say}
}

// say puts a line under the stage, or the hint when there is nothing to say.
func (k *kitchen) say(s string) {
	if s == "" {
		s = hint
	}
	k.status.SetText(s)
}

// show builds an entry from scratch and puts it on stage in place of the one
// there. Building from scratch is the point: what the panel shows as the
// build and the first frame is what a program showing this chart would pay.
func (k *kitchen) show(e *entry) {
	k.teardown()
	k.rec.reset()
	k.cur, k.buildErr = e, nil
	k.title.SetText(e.group + "  ›  " + e.title)
	k.note.SetText(e.note)
	k.say("")

	start := time.Now()
	v, err := e.build(k.env())
	if err != nil {
		k.buildErr = fmt.Errorf("%s: %w", e.id, err)
		k.stage.Objects = []fyne.CanvasObject{widget.NewLabel(k.buildErr.Error())}
		k.stage.Refresh()
		k.panel.shown()
		return
	}
	k.rec.built(time.Since(start))
	v.attach(k.rec.frame)
	k.v = v
	k.stage.Objects = []fyne.CanvasObject{v.obj}
	k.stage.Refresh()
	if v.start != nil {
		k.stop = v.start()
	}
	k.panel.shown()
}

// teardown takes the chart off the stage and releases it.
func (k *kitchen) teardown() {
	if k.stop != nil {
		k.stop()
		k.stop = nil
	}
	// Off the stage first, then closed: a chart laid out once more after
	// Close would open a new rasterizer, on whatever tier is current by then.
	k.stage.Objects = nil
	k.stage.Refresh()
	if k.v != nil {
		k.v.close()
		k.v = nil
	}
}

// rebuild builds the entry on stage again, for a switch that a chart cannot
// take while it is live.
func (k *kitchen) rebuild() {
	if k.cur != nil {
		k.show(k.cur)
	}
}

// setGPU switches the tier and reports whether it ended up where it was
// asked to. A rasterizer holds GPU state from its first frame, so the chart
// on stage is closed before the switch and built again after it — see
// gpu.Disable.
func (k *kitchen) setGPU(on bool) bool {
	k.teardown()
	ok := true
	if on {
		ok = gpuTier.Enable()
	} else {
		gpuTier.Disable()
	}
	k.rebuild()
	return ok
}

// setInteractive is the one switch a live chart takes as it is.
func (k *kitchen) setInteractive(on bool) {
	k.interactive = on
	if k.v != nil {
		k.v.setInteractive(on)
	}
}

func (k *kitchen) setDetail(on bool) {
	k.detail = on
	k.rebuild()
}

// err is what went wrong building or drawing the chart on stage.
func (k *kitchen) err() error {
	if k.buildErr != nil {
		return k.buildErr
	}
	if k.v != nil {
		return k.v.err()
	}
	return nil
}

// tick refreshes the panel four times a second, which is often enough to
// read a number while it changes and rarely enough to cost nothing.
func (k *kitchen) tick() {
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-k.done:
			return
		case <-t.C:
			fyne.Do(k.panel.refresh)
		}
	}
}

func (k *kitchen) close() {
	close(k.done)
	k.teardown()
}
