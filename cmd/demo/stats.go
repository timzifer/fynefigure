package main

import (
	"slices"
	"sync"
	"time"

	"github.com/timzifer/fynefigure"
)

// recorder keeps what the chart on stage has cost, for the side panel.
//
// It is fed from inside the widgets' draws — [fynefigure.Target.OnFrame] runs
// with the surface held, on Fyne's goroutine — so recording is all it does
// there: a lock and a few stores. The panel reads it on a ticker of its own.
type recorder struct {
	mu sync.Mutex

	// build is what making the plot and its widgets took, and shown is when
	// they went on stage, which is what the first frame is measured from.
	build time.Duration
	shown time.Time
	first time.Duration

	// draw is what each painted frame cost; probe what each call that painted
	// nothing cost — a hover's hit test, mostly — and trip how long a pointer
	// event waited for the frame that answered it, pacing included.
	draw, probe, trip ring
	paintedAt         []time.Time

	painted, idle uint64
	w, h          int
	dpr           float64

	// waiter is told the finish of every painted frame while a benchmark
	// waits on it, and is nil otherwise.
	waiter chan time.Time
}

func newRecorder() *recorder { return &recorder{} }

// reset forgets everything, for a new chart on stage. A benchmark waiting on
// frames keeps waiting: it notices the chart changed by itself.
func (r *recorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearLocked()
	r.build, r.shown, r.first = 0, time.Time{}, 0
	r.w, r.h, r.dpr = 0, 0, 0
}

// clear forgets the frames but keeps what the chart cost to make.
func (r *recorder) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearLocked()
}

func (r *recorder) clearLocked() {
	r.draw, r.probe, r.trip = ring{}, ring{}, ring{}
	r.paintedAt = r.paintedAt[:0]
	r.painted, r.idle = 0, 0
}

// built records what making the chart took and starts the clock for its first
// frame.
func (r *recorder) built(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.build, r.shown, r.first = d, time.Now(), 0
}

// frame is the widgets' OnFrame.
func (r *recorder) frame(f fynefigure.Frame) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !f.Painted {
		r.idle++
		r.probe.add(f.Cost)
		return
	}
	r.painted++
	r.draw.add(f.Cost)
	if f.Latency > 0 {
		r.trip.add(f.Latency)
	}
	r.w, r.h, r.dpr = f.W, f.H, f.DPR
	if r.first == 0 && !r.shown.IsZero() {
		r.first = f.At.Sub(r.shown)
	}
	r.paintedAt = append(r.paintedAt, f.At)
	if len(r.paintedAt) > 2*ringSize {
		r.paintedAt = append(r.paintedAt[:0], r.paintedAt[len(r.paintedAt)-ringSize:]...)
	}
	if r.waiter != nil {
		select {
		case r.waiter <- f.At:
		default:
		}
	}
}

// wait arms the recorder for a benchmark and returns the channel painted
// frames arrive on; unwait disarms it.
func (r *recorder) wait() chan time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.waiter = make(chan time.Time, 1)
	return r.waiter
}

func (r *recorder) unwait() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.waiter = nil
}

// snapshot is the panel's copy of the recorder.
type snapshot struct {
	built             bool
	build, first      time.Duration
	draw, probe, trip summary
	fps               float64
	painted, idle     uint64
	w, h              int
	dpr               float64
}

func (r *recorder) snapshot() snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := snapshot{
		built: !r.shown.IsZero(),
		build: r.build, first: r.first,
		draw: r.draw.summary(), probe: r.probe.summary(), trip: r.trip.summary(),
		painted: r.painted, idle: r.idle,
		w: r.w, h: r.h, dpr: r.dpr,
	}
	// Frames painted in the last second. A chart nobody is touching paints
	// nothing, and zero is the honest answer for it.
	now := time.Now()
	for i := len(r.paintedAt) - 1; i >= 0 && now.Sub(r.paintedAt[i]) <= time.Second; i-- {
		s.fps++
	}
	return s
}

// ringSize is how many frames a summary is over: a few seconds of a drag.
const ringSize = 120

// ring is the last ringSize durations.
type ring struct {
	v    [ringSize]time.Duration
	n, i int
	last time.Duration
}

func (r *ring) add(d time.Duration) {
	r.v[r.i] = d
	r.i = (r.i + 1) % ringSize
	r.n = min(r.n+1, ringSize)
	r.last = d
}

// summary is what the panel shows of a ring.
type summary struct {
	n                        int
	last, min, avg, p95, max time.Duration
}

func (r *ring) summary() summary {
	if r.n == 0 {
		return summary{}
	}
	return summarize(r.v[:r.n], r.last)
}

func summarize(ds []time.Duration, last time.Duration) summary {
	s := slices.Clone(ds)
	slices.Sort(s)
	var sum time.Duration
	for _, d := range s {
		sum += d
	}
	return summary{
		n:    len(s),
		last: last,
		min:  s[0],
		avg:  sum / time.Duration(len(s)),
		p95:  s[min(len(s)-1, len(s)*95/100)],
		max:  s[len(s)-1],
	}
}
