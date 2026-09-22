package main

// tier is what the demo asks of figure's GPU tier. Each platform fills it in:
// gpu.go with the tier itself, gpu_linux.go with a machine that has no device.
type tier struct {
	// Enabled reports whether rasterizers made now draw on the GPU.
	Enabled func() bool
	// Available reports whether a device answered, on or off.
	Available func() bool
	// Enable turns the tier on and reports whether it is on.
	Enable func() bool
	// Disable turns the tier off and keeps its device.
	Disable func()
	// Close gives the device back.
	Close func()
}
