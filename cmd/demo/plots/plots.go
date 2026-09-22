// Package plots is the catalogue of the kitchensink demo: every plot type
// figure can draw, as a builder the app calls whenever it needs a fresh chart.
//
// The builders are ports of the figure gallery (backend/gg/cmd/gallery) and of
// the figure examples, plus a few charts written for the demo. Each builder
// makes its data and its plot anew on every call, because the app rebuilds a
// chart when it switches renderer and a plot must not be shared between two
// widgets.
package plots

import (
	"github.com/timzifer/figure"
	"github.com/timzifer/figure/theme"
	"github.com/timzifer/figure/three"
)

// Entry is one leaf of the demo's tree. Exactly one of Plot, Scene, Grid is set.
type Entry struct {
	ID    string              // stable kebab-case id, unique, used by a -plot flag
	Group string              // tree group, one of the Group* constants below
	Title string              // tree label
	Note  string              // one sentence: what to try / what it shows
	Plot  func() *figure.Plot // flat chart (shown in an interactive widget)
	Scene func() *three.Plot  // 3D chart (shown in an orbit widget)
	Grid  func() *figure.Grid // subplot grid (rendered as a static image)
}

// The tree's groups.
const (
	GroupBasics        = "Basics"
	GroupAxes          = "Axes & scales"
	GroupStacking      = "Groups & stacking"
	GroupDistributions = "Distributions"
	GroupFields        = "Fields"
	GroupPolar         = "Polar"
	GroupSmith         = "Smith"
	GroupRelational    = "Relational"
	GroupAnnotations   = "Annotations"
	GroupLayout        = "Layout"
	Group3D            = "3D"
)

// Groups returns the group names in tree order.
func Groups() []string {
	return []string{
		GroupBasics, GroupAxes, GroupStacking, GroupDistributions, GroupFields,
		GroupPolar, GroupSmith, GroupRelational, GroupAnnotations, GroupLayout, Group3D,
	}
}

// All returns every entry, in tree order (grouped).
func All() []Entry {
	var out []Entry
	for _, g := range [][]Entry{
		basics(), axes(), stacking(), distributions(), fields(),
		polar(), smith(), relational(), annotations(), layout(), scenes(),
	} {
		out = append(out, g...)
	}
	return out
}

// Newest lists entry ids, most recently added first. The demo's tree opens on
// the head of it, so a chart type figure has just grown is the first thing on
// screen: add an id at the front when you add an entry for one.
func Newest() []string {
	return []string{
		"horizon",
		"nichols", "nichols-peak", "smith-vswr",
		"contour", "contour-heatmap", "contour-floor",
		"surface", "cascade", "bar3", "line3",
	}
}

// flat makes a builder for a flat chart: a fresh responsive plot at its design
// size, with a theme and a title, configured by build.
func flat(title string, w, h int, th theme.Theme, build func(*figure.Plot), opts ...figure.Option) func() *figure.Plot {
	return func() *figure.Plot {
		p := figure.New(append([]figure.Option{
			figure.Theme(th),
			figure.Size(w, h),
			figure.Title(title),
			figure.Responsive(true),
		}, opts...)...)
		build(p)
		return p
	}
}
