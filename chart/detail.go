package chart

import "time"

// Detail is the level-of-detail trade an interactive raster can make: while a
// reader is moving the chart, draw it coarser than the screen and let the GPU
// stretch it; when they stop, draw it properly. It is off unless [Detail] asks
// for it, because it is a visible trade.
//
// It works because the rasterizer's cost is per pixel and the stretch is not.
// Fyne hands a canvas.Raster the pixel size it is about to paint, but takes
// whatever size it is given and uploads that as the texture — the quad is drawn
// at the widget's size either way, so a smaller frame is scaled by the graphics
// card, for free, rather than resampled on the processor. Measured on a 900x480
// chart on an Apple M4, per frame, with figure's stilus rasterizer:
//
//	                   full     0.5
//	panning 4000 pts   0.81 ms  0.52 ms
//	a sliding stream   0.23 ms  0.17 ms
//
// Half the linear scale is a quarter of the pixels and far less than a
// quarter saved, because on stilus most of a frame is spent per mark rather
// than per pixel. Under gg, figure's earlier rasterizer, the same trade took a
// pan from 45 ms to 21 ms; on stilus it buys a fraction of a millisecond, so
// it is worth having only on a very large chart or a slow machine.

// howSharpToSharpen is how long after the last wheel notch the chart is drawn
// properly again. A drag says when it ends; a wheel does not, so it is timed.
const howSharpToSharpen = 120 * time.Millisecond

// coarsen drops the rasterizer to the interactive resolution for the gesture
// that is starting, if it is not there already.
// setCoarse records the resolution the chart is drawn at. The painter reads it,
// so it lives under the small lock rather than the chart's own.
func (c *Chart) setCoarse(on bool) {
	c.mu.Lock()
	c.coarse = on
	c.mu.Unlock()
}

// isCoarse reports whether the chart is drawn below the screen's resolution.
func (c *Chart) isCoarse() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.coarse
}

func (c *Chart) coarsen() {
	if c.isCoarse() || c.live == nil || c.cfg.detail >= 1 || c.cfg.detail <= 0 {
		return
	}
	c.setCoarse(true)
	rescale := func() error { return c.live.Rescale(c.dpr * float64(c.cfg.detail)) }
	if err := c.target.Render(rescale); err != nil {
		c.renderr = err
	}
}

// sharpen puts the rasterizer back to the screen's own resolution and redraws.
func (c *Chart) sharpen() {
	if c.sharpTimer != nil {
		c.sharpTimer.Stop()
		c.sharpTimer = nil
	}
	if !c.isCoarse() || c.live == nil {
		return
	}
	c.setCoarse(false)
	if err := c.target.Render(func() error { return c.live.Rescale(c.dpr) }); err != nil {
		c.renderr = err
		return
	}
	c.present()
}

// armSharpen asks for a sharp frame once the wheel has been still for a while.
func (c *Chart) armSharpen() {
	if !c.isCoarse() {
		return
	}
	if c.sharpTimer != nil {
		c.sharpTimer.Reset(howSharpToSharpen)
		return
	}
	c.sharpTimer = time.AfterFunc(howSharpToSharpen, func() { doOnFyne(c.locked(c.sharpen)) })
}
