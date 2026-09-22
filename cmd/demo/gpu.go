//go:build !linux

// The demo draws through the GPU tier where a binary can hold both it and
// Fyne's desktop driver. On Linux it cannot, which is why this file is not
// built there.
//
// The tier reaches dlopen, dlsym and __errno_location through
// //go:cgo_import_dynamic. Fyne's desktop driver is cgo, and on linux/amd64
// with cgo on, those imports do not resolve — the link ends in
//
//	dlopen_stub: unhandled relocation for goffi_dlopen (SDYNIMPORT) (R_CALL)
//
// no matter how it is driven: -linkmode=external, -extldflags=-ldl, -tags
// nofakecgo and goffi v0.6.4 were each tried on a Linux runner and each ended
// the same way. Nothing upstream has met this before, because figure builds
// the tier with CGO_ENABLED=0 throughout and this repository's gpu module is
// only ever linked into a test binary that pulls in no cgo at all. macOS and
// Windows link it beside Fyne without complaint; both are checked on a runner.
//
// A Linux reader gets the CPU rasterizer, which is what they would have got
// from a machine with no usable device anyway.

package main

// Importing the tier is the whole opt-in: it registers gg's accelerator from
// its init, and every rasterizer made afterwards uses it.
import "github.com/timzifer/fynefigure/gpu"

// gpuTier is the tier as the demo reaches it, and the only place the demo
// names the gpu package: a file anywhere else that imported it would put the
// tier back into the Linux binary. See gpu_linux.go for what Linux gets.
var gpuTier = tier{
	Enabled:   gpu.Enabled,
	Available: gpu.Available,
	Enable:    gpu.Enable,
	Disable:   gpu.Disable,
	Close:     gpu.Close,
}
