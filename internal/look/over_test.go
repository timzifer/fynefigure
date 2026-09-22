package look

import (
	"image/color"
	"testing"

	figuretheme "github.com/timzifer/figure/theme"
)

// A chart restyled to follow its application keeps what its author chose to
// show. A pie built with its grid, axis lines and ticks off stays a pie, and
// a chart that asked for redundant encoding keeps it — while the page colour
// is still the application's.
func TestOverKeepsWhatTheAuthorChoseToShow(t *testing.T) {
	authored := figuretheme.Light.With(
		figuretheme.Grid(false, false),
		figuretheme.AxisLines(false, false),
		figuretheme.Ticks(false, false),
		figuretheme.Redundant(true),
	)
	page := State{Background: color.RGBA{R: 0x10, G: 0x12, B: 0x16, A: 0xff}, Size: 14}
	th := page.Over(authored)

	if th.ShowGridX || th.ShowGridY || th.ShowAxisLineX || th.ShowAxisLineY || th.ShowTicksX || th.ShowTicksY {
		t.Errorf("the restyled theme turned furniture back on: grid %v/%v, axes %v/%v, ticks %v/%v",
			th.ShowGridX, th.ShowGridY, th.ShowAxisLineX, th.ShowAxisLineY, th.ShowTicksX, th.ShowTicksY)
	}
	if len(th.SeriesDashes) != len(authored.SeriesDashes) || len(th.SeriesMarkers) != len(authored.SeriesMarkers) {
		t.Error("the restyled theme dropped the redundant encoding")
	}
	want := page.Theme()
	if th.Background != want.Background || th.TickColor != want.TickColor || th.TickSize != want.TickSize {
		t.Errorf("the restyled theme is not the page's: background %v, ink %v, size %v; want %v, %v, %v",
			th.Background, th.TickColor, th.TickSize, want.Background, want.TickColor, want.TickSize)
	}
}
