package plots

import (
	"math"

	"github.com/timzifer/figure"
	"github.com/timzifer/figure/coord"
	"github.com/timzifer/figure/data"
	"github.com/timzifer/figure/geom"
	"github.com/timzifer/figure/ir"
	"github.com/timzifer/figure/palette"
	"github.com/timzifer/figure/scale"
	"github.com/timzifer/figure/theme"
	"github.com/timzifer/figure/three"
)

// --- Ternary ------------------------------------------------------------------

func ternary() []Entry {
	g := GroupTernary
	return []Entry{
		{ID: "soil-texture", Group: g, Title: "Soil texture triangle",
			Note: "Three parts of a mixture in a triangle: the chart reads sand and silt and works the clay out, because two of the three are free. Hover a point to read the composition back.",
			Plot: flat("Particle size, by field", 560, 520, theme.Light, func(p *figure.Plot) {
				// Both axes run over the whole simplex whatever the samples do —
				// coord.Ternary pins them, because a triangle drawn round a
				// cluster is not a triangle.
				p.X(scale.Linear())
				p.Y(scale.Linear())
				p.Add(geom.Scatter(soilSamples(),
					geom.X("sand"), geom.Y("silt"), geom.Size(6),
					geom.ColorBy("field", scale.Qualitative(palette.OkabeIto))))
				// The two axes name themselves. The third family of grid lines
				// is drawn and unlabelled, so the component it counts is named
				// at its own corner — which is how a texture triangle is
				// printed anyway, and what makes the derived part readable.
				p.Add(geom.Note(5, 5, "100% clay", geom.Align(ir.AlignCenter, ir.AlignTop)))
			}, figure.Coord(coord.Ternary(coord.TernarySum(100))),
				figure.XTitle("sand (%)"), figure.YTitle("silt (%)"))},
		{ID: "alloy-prism", Group: g, Title: "Ternary prism (Fe–Cr–Ni)",
			Note:  "The same triangle as the floor of a scene, with the melting point up the fourth axis: a composition whose reading depends on one more number. Orbit to see the liquidus sag away from the chromium corner.",
			Scene: alloyPrismScene},
	}
}

// soilSamples is ninety particle-size analyses from three fields, as
// percentages that add to a hundred. Only sand and silt are columns: the clay
// is what is left, which is the constraint the coord holds by construction.
func soilSamples() *data.Table {
	centres := []struct {
		field      string
		sand, silt float64
	}{
		{"river terrace", 62, 24},
		{"lower slope", 28, 46},
		{"floodplain", 14, 34},
	}
	var sand, silt []float64
	var field []string
	for c, centre := range centres {
		for i := range 30 {
			sa := centre.sand + 6*noise(7100+c*100+i)
			si := centre.silt + 6*noise(7500+c*100+i)
			sa, si = math.Max(sa, 0), math.Max(si, 0)
			if sa+si > 100 {
				// A composition outside the simplex is a rounding of one on it.
				k := 100 / (sa + si)
				sa, si = sa*k, si*k
			}
			sand, silt = append(sand, sa), append(silt, si)
			field = append(field, centre.field)
		}
	}
	return figure.NewTable().Float64("sand", sand).Float64("silt", silt).String("field", field)
}

// alloyPrismScene is a stainless-steel field: iron, chromium and nickel in
// percent on the floor, and the temperature the alloy melts at up the prism.
//
// The floor is coord.Ternary's triangle placed in a scene, so three.Scatter3
// draws in it unchanged — a layer puts its geometry through the frame and that
// is all it has to know about which space it is in.
func alloyPrismScene() *three.Plot {
	// A crude liquidus over the whole field: iron and nickel melt near 1500 °C
	// and chromium higher, and a mixture melts below the line between its
	// components, which is what makes the surface sag in the middle.
	melting := func(fe, cr, ni float64) float64 {
		ideal := (1538*fe + 1907*cr + 1455*ni) / 100
		return ideal - 0.021*(fe*cr+cr*ni+fe*ni)/100
	}
	// The composition is sampled on the simplex's own lattice, in steps of
	// four percent, so the cloud fills the prism rather than crowding one
	// corner the way a real alloy catalogue would.
	var fe, cr, ni, degC []float64
	for c := 0.0; c <= 100; c += 4 {
		for n := 0.0; c+n <= 100; n += 4 {
			f := 100 - c - n
			fe, cr, ni = append(fe, f), append(cr, c), append(ni, n)
			degC = append(degC, melting(f, c, n))
		}
	}
	src := figure.NewTable().
		Float64("fe", fe).Float64("cr", cr).Float64("ni", ni).Float64("degC", degC)

	// X is the first component and Y the second; the nickel is derived, and is
	// a column here only for a tooltip to read back.
	sc := three.NewScene(
		three.ZTitle("melting point (°C)"),
		three.Prism(three.PrismSum(100), three.PrismCorners("Fe", "Cr", "Ni")),
	).
		Z(scale.Linear(scale.Nice())).
		Add(three.Scatter3(src, geom.X("fe"), geom.Y("cr"), geom.Z("degC"),
			geom.ColorBy("degC", scale.Sequential(palette.Magma)),
			geom.Size(5), geom.Droplines(false)))

	return three.New(
		three.Size(620, 560),
		three.Title("Where an Fe–Cr–Ni alloy melts"),
		three.Theme(theme.Light),
	).Scene(sc).Add(three.View{Camera: three.LookAt(three.Azimuth(-0.7), three.Elevation(0.34), three.Zoom(1.25))})
}
