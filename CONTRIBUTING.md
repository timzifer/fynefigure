# Contributing

## Layout

Two modules, one repository.

| Path | Module | Depends on |
|---|---|---|
| `.` | `github.com/timzifer/fynefigure` | Fyne, figure, figure's raster backend (`backend/stilus`) |
| `cmd/demo` | `github.com/timzifer/fynefigure/cmd/demo` | the above |

The demo is a module of its own so that what it pulls in to show every chart
stays out of the widget's module graph. Its `replace` directive points at the
directory above it, and nothing imports the demo, so it costs no one anything.

There used to be a third, `gpu`, which switched on gg's GPU tier. figure
replaced gg with stilus
([ADR 0094](https://github.com/timzifer/figure/blob/main/docs/adr/0094-the-raster-backend-is-stilus.md)),
which leaves the tier nothing to accelerate, and stilus on the CPU draws a
frame faster than the tier did — so it was removed rather than ported.

Both modules declare Go 1.27.1, because `backend/stilus` does: the font
library it draws text with requires it.

Inside the main module the split is figure's own, between `backend/window` and
`backend/window/show`: `fynefigure` draws, and `fynefigure/chart` and
`fynefigure/orbit` steer. A backend must not know what a scale or a panel is,
and everything in `chart` is about scales and panels — and everything in
`orbit` about cameras. A change that needs to cross that line is a sign the
seam is in the wrong place — say so rather than routing around it.

`chart` and `orbit` are two widgets rather than one because figure has two
plots: a `figure.Plot` is steered by figure's own `Input`, and a `three.Plot`
has no scales to pan and leaves its loop to the host. What the two share —
reading Fyne's theme — is in `internal/look`.

## Everyday commands

```sh
# what CI runs
gofmt -l .                                   # must print nothing
go vet ./... && (cd cmd/demo && go vet ./...)
go build ./... && (cd cmd/demo && go build ./...)
go test ./... && (cd cmd/demo && go test ./...)
go test -race ./...

# the library must not need cgo; only the demo does, because Fyne's desktop
# driver does
CGO_ENABLED=0 go build . ./chart ./orbit

# a chart in a window, by hand — the one thing CI cannot check. The demo is a
# module of its own: see cmd/demo/go.mod.
(cd cmd/demo && go run .)
```

Developing against a figure checkout next door wants a workspace. It is not
committed, and the `require` directives always name published tags:

```sh
go work init . ./cmd/demo ../figure ../figure/backend/stilus
```

## The measurements

Several defaults in this repository are numbers rather than opinions, and the
benchmarks are where they come from. Re-run them before changing one.

```sh
go test -run='^$' -bench=Frame -benchtime=25x .        # what a frame costs
go test -run='^$' -bench=Drag  -benchtime=20x ./chart  # what the pacing saves
```

`fynefigure.DamageBudget`, `chart.Detail`, `chart.FrameInterval` and the
choice to draw whole frames rather than partial ones each have a table in their
doc comment. A change that moves one should move its table too.

## Tests

They need no display: Fyne's test driver paints in software. What is asserted
is what figure reports — the size the chart was laid out at, the domain after
a zoom, the frames it painted — rather than how it looks. A test that scraped
pixels to check an interaction would be testing the rasterizer.

Where a test does read pixels it is because pixels are the subject: that the
widget draws what the same plot exports, that a followed axis is drawn at all,
that a font has the glyphs a chart uses.

Write the test that would have caught the bug.
`TestAFollowedAxisMovesWithEveryFrameRatherThanInSteps` exists because a
sliding chart's axis once held still for half a tick and then jumped.

## Things worth knowing before changing them

**Drawing happens under `Target.Render`.** The rasterizer is single-goroutine
and Fyne reads its buffer from the painter, which is the drawing goroutine on a
desktop driver and is not under the test one. Resting on the two being the same
is what the lock exists to avoid. `Render` is not reentrant, and `Plot.Live`
and `Live.Close` both reach back into the target — so neither goes inside one.

**Input is paced, and dropping an event loses nothing.** figure pans by the
distance from the last position it was told about, and wheel deltas are summed,
which is exact. Anything added to the gesture path has to keep that true.

**A followed axis is released and un-niced.** figure retrains every unpinned
axis on each render, but a zoom, a pin and a high-water mark survive that, and
a linear axis rounds its domain outward — right for a still chart and wrong
for a sliding window. See `chart.Follow`.

**Partial repaints are off on purpose.** On stilus they save a fifth at best,
on a frame that is already well under a millisecond. The number is in
`fynefigure.DamageBudget`.
