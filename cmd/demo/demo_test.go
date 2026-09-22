package main

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// open puts the kitchen in a test window at the size the demo opens at.
func open(t *testing.T) *kitchen {
	t.Helper()
	test.NewTempApp(t)
	w := test.NewTempWindow(t, nil)
	k := newKitchen(w, catalog())
	w.SetContent(k.content())
	w.Resize(fyne.NewSize(1400, 820))
	t.Cleanup(k.close)
	return k
}

// settle lays the stage out and waits for a painted frame, which is what a
// chart that went on stage owes.
func settle(k *kitchen) {
	k.stage.Refresh()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if k.rec.snapshot().painted > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Every leaf of the tree is a chart that builds and draws: the catalogue is
// the demo's whole content, and an entry that fails is a blank stage nobody
// notices until they click it.
func TestEveryEntryBuildsAndPaints(t *testing.T) {
	k := open(t)
	for _, e := range k.cat.entries {
		k.show(e)
		settle(k)
		if err := k.err(); err != nil {
			t.Errorf("%s: %v", e.id, err)
			continue
		}
		s := k.rec.snapshot()
		if s.painted == 0 {
			t.Errorf("%s: nothing was painted", e.id)
		}
		t.Logf("%-20s build %6s   first %6s   draw %6s", e.id, ms(s.build), ms(s.first), ms(s.draw.last))
	}
}

// The GPU switch closes the chart on stage, switches the tier and builds the
// chart again, and has to leave a chart that draws on whichever tier it lands.
func TestTheGPUSwitchRebuildsAChartThatPaints(t *testing.T) {
	k := open(t)
	was := gpuTier.Enabled()
	t.Cleanup(func() {
		if was {
			gpuTier.Enable()
		}
	})
	k.show(k.cat.find("decimation"))
	settle(k)

	for _, on := range []bool{false, true, false, true} {
		got := k.setGPU(on)
		settle(k)
		if on && got != gpuTier.Available() {
			t.Errorf("setGPU(true) = %v with the tier available = %v", got, gpuTier.Available())
		}
		if !on && gpuTier.Enabled() {
			t.Error("setGPU(false) left the tier on")
		}
		if err := k.err(); err != nil {
			t.Errorf("after setGPU(%v): %v", on, err)
		}
		s := k.rec.snapshot()
		if s.painted == 0 {
			t.Errorf("after setGPU(%v): nothing was painted", on)
		}
		t.Logf("GPU %-5v  enabled %-5v  draw %s", on, gpuTier.Enabled(), ms(s.draw.last))
	}
}

// A benchmark nudge must paint a frame on every kind of stage — a flat chart
// panned, a scene turned, a still drawn again — or the benchmark times out.
func TestANudgePaints(t *testing.T) {
	k := open(t)
	for _, id := range []string{"signal", "surface", "subplots"} {
		k.show(k.cat.find(id))
		settle(k)
		k.v.aim()
		before := k.rec.snapshot().painted
		for i := range 4 {
			k.v.nudge(i)
		}
		if after := k.rec.snapshot().painted; after < before+4 {
			t.Errorf("%s: four nudges painted %d frames", id, after-before)
		}
	}
}

// Switching away from an animated view stops it: nothing is painted for a
// chart that is off the stage.
func TestSwitchingAwayStopsAnimation(t *testing.T) {
	k := open(t)
	for _, id := range []string{"live", "transition"} {
		k.show(k.cat.find(id))
		settle(k)
		time.Sleep(300 * time.Millisecond)
		k.teardown()
		k.rec.reset()
		time.Sleep(300 * time.Millisecond)
		if n := k.rec.snapshot().painted; n != 0 {
			t.Errorf("%s: %d frames painted after it left the stage", id, n)
		}
	}
}

// The tree starts closed but for New, which lists the newest charts and
// selects them under its own nodes; a chart that is not new opens its group.
func TestTheTreeOpensOnNew(t *testing.T) {
	k := open(t)
	for _, g := range k.cat.children[""] {
		if open := k.tree.IsBranchOpen(g); open != (g == newNode) {
			t.Errorf("%s: open = %v", g, open)
		}
	}
	fresh := k.cat.children[newNode]
	if len(fresh) == 0 || len(fresh) > newest {
		t.Fatalf("New holds %d charts, want 1..%d", len(fresh), newest)
	}

	k.open("")
	if k.node != fresh[0] {
		t.Errorf("open(\"\") selected %q, want %q", k.node, fresh[0])
	}
	if k.cur == nil || newPrefix+k.cur.id != fresh[0] {
		t.Errorf("open(\"\") put %v on stage", k.cur)
	}

	k.open("signal")
	if k.node != "signal" || !k.tree.IsBranchOpen(groupPrefix+k.cur.group) {
		t.Errorf("open(signal): node %q, group open %v", k.node, k.tree.IsBranchOpen(groupPrefix+k.cur.group))
	}
}
