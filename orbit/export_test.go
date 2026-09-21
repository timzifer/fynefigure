package orbit

import (
	"testing"
	"time"
)

// Pointer is the layer that takes the pointer for a chart.
type Pointer = pointer

// PointerOf is where a test hands a chart the events Fyne's driver would,
// without a canvas deciding whether they reach it. It works whether or not the
// chart is [Interactive]; a test of that goes through the canvas.
func PointerOf(c *Chart) *Pointer { return c.ptr }

// SelectionRings reports how many rings the selection drew as of the last
// frame, and how many of them said they were behind something. It is how a test
// reaches the answer the overlay computed while it drew.
func SelectionRings(c *Chart) (total, hidden int) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.mk == nil {
		return 0, 0
	}
	for _, mk := range c.mk.ring.Marks {
		total++
		if mk.Hidden {
			hidden++
		}
	}
	return total, hidden
}

// WheelSettles sets how long a wheel has to be still to be over, for the rest
// of the test. A test that sends two notches and expects one gesture needs the
// gap between them to be shorter than this, which on a runner drawing under the
// race detector 120ms is not.
func WheelSettles(t *testing.T, d time.Duration) {
	was := wheelSettles
	wheelSettles = d
	t.Cleanup(func() { wheelSettles = was })
}
