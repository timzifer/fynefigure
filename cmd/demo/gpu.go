//go:build !linux

// The demo draws through the GPU tier where a binary can hold both it and
// Fyne's desktop driver. On Linux it cannot, which is why this file is not
// built there and tier_linux.go answers for it instead.
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
//
// Keeping that promise means no package of this binary may name the tier on
// Linux — not even to ask whether it is available. So every call goes through
// the six functions below, and the whole import lives in this one file.

package main

import (
	// Importing the tier is the whole opt-in: it registers gg's accelerator
	// from its init, and every rasterizer made afterwards uses it.
	"github.com/timzifer/fynefigure/gpu"
)

// tierLinked reports whether this binary carries the GPU tier at all.
func tierLinked() bool { return true }

// tierEnabled reports whether charts are being drawn on the tier.
func tierEnabled() bool { return gpu.Enabled() }

// tierAvailable reports whether a device answered.
func tierAvailable() bool { return gpu.Available() }

// tierEnable puts the rasterizer back on the tier, and reports whether it went.
func tierEnable() bool { return gpu.Enable() }

// tierDisable takes the rasterizer off the tier, keeping the device.
func tierDisable() { gpu.Disable() }

// tierClose gives the device back, which is gg's advice on the way out.
func tierClose() { gpu.Close() }
