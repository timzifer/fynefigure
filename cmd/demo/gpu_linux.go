// On Linux the demo is built without the GPU tier; gpu.go says why. It
// answers as a machine with no usable device would, so the panel greys out
// its GPU switch and says the chart is drawn on the CPU rasterizer.

package main

var gpuTier = tier{
	Enabled:   func() bool { return false },
	Available: func() bool { return false },
	Enable:    func() bool { return false },
	Disable:   func() {},
	Close:     func() {},
}
