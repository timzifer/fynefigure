package plots

import (
	"fmt"
	"math"

	"github.com/timzifer/figure"
	"github.com/timzifer/figure/geom"
	"github.com/timzifer/figure/palette"
	"github.com/timzifer/figure/scale"
	"github.com/timzifer/figure/theme"
	"github.com/timzifer/figure/three"
)

func scenes() []Entry {
	g := Group3D
	return []Entry{
		{ID: "surface", Group: g, Title: "Surface (three views)",
			Note:  "One response surface seen from three cameras at once; drag to orbit.",
			Scene: surfaceScene},
		{ID: "cascade", Group: g, Title: "Cascade (waterfall)",
			Note:  "Thirty spectrum sweeps as grouped 3D lines, with a carrier drifting through them.",
			Scene: cascadeScene},
		{ID: "bar3", Group: g, Title: "3D bars",
			Note:  "One box per service and region over two ordinal floor axes; orbit to see what the back row hides.",
			Scene: bar3Scene},
		{ID: "line3", Group: g, Title: "3D trajectory (Lorenz)",
			Note:  "The Lorenz attractor as one 3D path, which crosses itself in every flat projection but not in space.",
			Scene: line3Scene},
	}
}

func surfaceScene() *three.Plot {
	const n = 26
	xs := make([]float64, 0, n*n)
	ys := make([]float64, 0, n*n)
	zs := make([]float64, 0, n*n)
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			x := -3 + 6*float64(i)/float64(n-1)
			y := -3 + 6*float64(j)/float64(n-1)
			xs = append(xs, x)
			ys = append(ys, y)
			zs = append(zs, math.Sin(x)*math.Cos(y)*1.4+0.25*x)
		}
	}
	src := figure.NewTable().Float64("x", xs).Float64("y", ys).Float64("z", zs)
	sc := three.NewScene(three.XTitle("x"), three.YTitle("y"), three.ZTitle("z")).
		Z(scale.Linear(scale.Nice())).
		Add(three.Surface(src, geom.X("x"), geom.Y("y"), geom.Z("z"),
			geom.Fill(palette.Blue)))

	return three.New(
		three.Size(900, 340),
		three.Title("A response surface, from three angles"),
		three.Theme(theme.Light),
		three.Columns(3),
	).Scene(sc).Add(
		three.View{Camera: three.Home(), Label: "three-quarter"},
		three.View{Camera: three.LookAt(three.Elevation(1.45)), Label: "plan"},
		three.View{Camera: three.LookAt(three.Azimuth(0), three.Elevation(0.02)), Label: "front"},
	)
}

func cascadeScene() *three.Plot {
	const (
		traces = 30
		bins   = 101
	)
	var freq, sweep, power []float64
	var trace []string
	for n := 0; n < traces; n++ {
		t := float64(n) / (traces - 1)
		carrier := 900 + 24*t
		harmonic := 960 + 6*t
		for i := 0; i < bins; i++ {
			fr := 860 + 140*float64(i)/(bins-1)
			p := -92 + 3*math.Sin(float64(i)*0.7+float64(n)*0.3)
			p = math.Max(p, lorentz(fr, carrier, -28, 1.6))
			p = math.Max(p, lorentz(fr, harmonic, -58+22*t, 1.1))
			freq = append(freq, fr)
			sweep = append(sweep, float64(n))
			power = append(power, p)
			trace = append(trace, fmt.Sprintf("sweep %02d", n))
		}
	}
	src := figure.NewTable().
		Float64("f", freq).Float64("n", sweep).Float64("p", power).
		String("trace", trace)

	sc := three.NewScene(
		three.XTitle("frequency (MHz)"),
		three.YTitle("sweep"),
		three.ZTitle("power (dBm)"),
	).
		Z(scale.Linear(scale.Nice())).
		Add(three.Line3(src,
			geom.X("f"), geom.Y("n"), geom.Z("p"),
			geom.GroupBy("trace"),
			geom.Color(palette.SkyBlue),
			geom.Width(1),
		))

	return three.New(
		three.Size(760, 520),
		three.Title("A drifting carrier, thirty sweeps"),
		three.Theme(theme.Dark),
	).Scene(sc).Add(
		three.View{Camera: three.LookAt(three.Azimuth(-0.9), three.Elevation(0.42))},
	)
}

// bar3Scene is latency per service and region. Both floor axes are ordinal,
// which is what three.Bar3 asks for.
func bar3Scene() *three.Plot {
	services := []string{"auth", "search", "cart", "checkout", "media"}
	regions := []string{"eu", "us", "apac", "sa"}
	var svc, reg []string
	var ms []float64
	for i, s := range services {
		for j, r := range regions {
			svc = append(svc, s)
			reg = append(reg, r)
			ms = append(ms, 40+18*float64(i)+25*float64(j)+15*math.Sin(float64(i*3+j)))
		}
	}
	src := figure.NewTable().String("service", svc).String("region", reg).Float64("ms", ms)

	sc := three.NewScene(
		three.XTitle("service"),
		three.YTitle("region"),
		three.ZTitle("latency (ms)"),
	).
		X(scale.Ordinal()).
		Y(scale.Ordinal()).
		Z(scale.Linear(scale.Nice(), scale.Zero())).
		Add(three.Bar3(src,
			geom.X("service"), geom.Y("region"), geom.Z("ms"),
			geom.Fill(palette.Blue), geom.BarWidth(0.8)))

	return three.New(
		three.Size(760, 520),
		three.Title("Latency by service and region"),
		three.Theme(theme.Light),
	).Scene(sc).Add(three.View{Camera: three.Home()})
}

// line3Scene integrates the Lorenz system: a trajectory that only makes sense
// in three dimensions.
func line3Scene() *three.Plot {
	const (
		n     = 4000
		dt    = 0.008
		sigma = 10.0
		rho   = 28.0
		beta  = 8.0 / 3
	)
	xs, ys, zs := make([]float64, n), make([]float64, n), make([]float64, n)
	x, y, z := 1.0, 1.0, 1.0
	deriv := func(x, y, z float64) (float64, float64, float64) {
		return sigma * (y - x), x*(rho-z) - y, x*y - beta*z
	}
	for i := range n {
		// Classic fourth-order Runge–Kutta, so the path is smooth at this step.
		k1x, k1y, k1z := deriv(x, y, z)
		k2x, k2y, k2z := deriv(x+dt/2*k1x, y+dt/2*k1y, z+dt/2*k1z)
		k3x, k3y, k3z := deriv(x+dt/2*k2x, y+dt/2*k2y, z+dt/2*k2z)
		k4x, k4y, k4z := deriv(x+dt*k3x, y+dt*k3y, z+dt*k3z)
		x += dt / 6 * (k1x + 2*k2x + 2*k3x + k4x)
		y += dt / 6 * (k1y + 2*k2y + 2*k3y + k4y)
		z += dt / 6 * (k1z + 2*k2z + 2*k3z + k4z)
		xs[i], ys[i], zs[i] = x, y, z
	}
	src := figure.NewTable().Float64("x", xs).Float64("y", ys).Float64("z", zs)

	sc := three.NewScene(three.XTitle("x"), three.YTitle("y"), three.ZTitle("z")).
		X(scale.Linear(scale.Nice())).
		Y(scale.Linear(scale.Nice())).
		Z(scale.Linear(scale.Nice())).
		Add(three.Line3(src, geom.X("x"), geom.Y("y"), geom.Z("z"),
			geom.Color(palette.Orange), geom.Width(1)))

	return three.New(
		three.Size(720, 560),
		three.Title("The Lorenz attractor"),
		three.Theme(theme.Dark),
	).Scene(sc).Add(three.View{Camera: three.LookAt(three.Azimuth(-0.7), three.Elevation(0.3))})
}

// contourFloorScene is the contour field as a surface with its isolines on the
// floor, off the same ramp and levels as the flat contour charts.
func contourFloorScene() *three.Plot {
	src := response()
	sc := three.NewScene(
		three.XTitle("bias (V)"),
		three.YTitle("drive (dBm)"),
		three.ZTitle("gain (dB)"),
	).
		Z(scale.Linear(scale.Nice())).
		Add(
			three.Contour(src,
				geom.X("bias"), geom.Y("drive"), geom.Z("gain"),
				geom.Levels(fieldLevels()...), geom.ColorBy("gain", fieldRamp())),
			three.Surface(src,
				geom.X("bias"), geom.Y("drive"), geom.Z("gain"),
				geom.ColorBy("gain", fieldRamp())),
		)
	return three.New(
		three.Size(660, 560),
		three.Title("The shape, and its plan beneath it"),
		three.Theme(theme.Light),
	).Scene(sc)
}
