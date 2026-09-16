module github.com/timzifer/fynefigure

go 1.25.0

// figure's raster backend is pinned the way every figure module pins what it
// adapts: this bridge is validated against exactly one revision of the core and
// one of the rasterizer, and says which.
//
// Both name a commit of figure's main branch rather than a tag, because
// gg.WithFallbackFont — what puts figure's own fonts behind the application's
// typeface, see internal/look.Fallback — is in no release yet. A pseudo-version
// is what keeps this module buildable by anyone who fetches it in the meantime;
// it goes back to naming tags when figure is released again.
require (
	fyne.io/fyne/v2 v2.7.3
	github.com/timzifer/figure v0.11.1-0.20260916055931-8b2c05ccdec0
	github.com/timzifer/figure/backend/gg v0.11.1-0.20260916055931-8b2c05ccdec0
	// The Go fonts, which stand behind the application's typeface for a glyph
	// it has not got. See internal/look.Fallback.
	golang.org/x/image v0.45.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fredbi/uri v1.1.1 // indirect
	github.com/fyne-io/gl-js v0.2.0 // indirect
	github.com/fyne-io/oksvg v0.2.0 // indirect
	github.com/go-gl/gl v0.0.0-20231021071112-07e5d0ea2e71 // indirect
	github.com/go-text/render v0.2.0 // indirect
	github.com/go-text/typesetting v0.3.3 // indirect
	github.com/gogpu/gg v0.52.5 // indirect
	github.com/gogpu/gpucontext v0.28.0 // indirect
	github.com/gogpu/gputypes v0.5.2 // indirect
	github.com/hack-pad/go-indexeddb v0.3.2 // indirect
	github.com/hack-pad/safejs v0.1.0 // indirect
	github.com/jeandeaual/go-locale v0.0.0-20250612000132-0ef82f21eade // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/nicksnyder/go-i18n/v2 v2.5.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/yuin/goldmark v1.7.8 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gogpu/gg => github.com/timzifer/gg v0.52.6-figure.5
