//go:build linux

// On Linux the demo is built without the GPU tier — see gpu.go for why — and
// these are what it is built with instead.
//
// They are stubs rather than a thin wrapper that returns false, because the
// point is the import: a Linux binary that named the tier anywhere, even to
// ask whether a device answered, would pull its foreign-function layer in and
// fail to link. So the answer here is the answer a machine with no usable
// device gives, and nothing in this file imports anything.

package main

func tierLinked() bool { return false }

func tierEnabled() bool { return false }

func tierAvailable() bool { return false }

func tierEnable() bool { return false }

func tierDisable() {}

func tierClose() {}
