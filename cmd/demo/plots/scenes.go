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
		{ID: "scatter3", Group: g, Title: "3D scatter",
			Note:  "Two batches that overlap on the floor and part only in the third column. Each point drops a line to the floor, which is how its height reads; orbit to see the two clouds come apart.",
			Scene: scatter3Scene},
		{ID: "smith-sphere", Group: g, Title: "Smith sphere",
			Note:  "The Smith chart carried onto the Riemann sphere. A negative resistance, which runs off the page of a flat chart, crosses the equator into the southern hemisphere and comes back.",
			Scene: smithSphereScene},
		{ID: "antenna-pattern", Group: g, Title: "Radiation pattern (spherical)",
			Note:  "A four-element broadside array's power pattern as a surface on a spherical scene: the radius in every direction is the power sent that way.",
			Scene: antennaScene},
		{ID: "ribbon", Group: g, Title: "Ribbon (a line with width)",
			Note:  "A descending approach drawn as a band rather than a stroke. The band is a surface, so it is shaded and it occludes: where the path crosses itself, the ribbon says which pass is in front and a line of constant width says nothing.",
			Scene: ribbonScene},
		{ID: "joint-orientations", Group: g, Title: "Spherical histogram",
			Note:  "Which way a set of fractures point, binned into cells of equal area on the sphere and stood up at their count. A latitude–longitude grid would pile a false ring round the equator; bands of equal cos θ do not.",
			Scene: jointScene},
		{ID: "bloch", Group: g, Title: "Bloch sphere",
			Note:  "A detuned Rabi oscillation as two dozen states on the Bloch sphere, starting at |0⟩. Consecutive states are joined along the arc between them, so the coarse sweep still runs over the surface rather than cutting through the ball.",
			Scene: blochScene},
	}
}

// scatter3Scene is two production batches measured on three dimensions. Their
// length and width overlap, so the flat chart of those two sees one cloud; the
// thickness is what tells them apart.
func scatter3Scene() *three.Plot {
	const n = 160
	var l, w, t []float64
	var batch []string
	for i := range n {
		b, off := "A", 0.0
		if i%2 == 1 {
			b, off = "B", 0.9
		}
		l = append(l, 20+0.6*noise(11000+i))
		w = append(w, 12+0.5*noise(12000+i)+0.3*off)
		t = append(t, 3+0.18*noise(13000+i)+off)
		batch = append(batch, b)
	}
	src := figure.NewTable().Float64("l", l).Float64("w", w).Float64("t", t).String("batch", batch)

	sc := three.NewScene(
		three.XTitle("length (mm)"),
		three.YTitle("width (mm)"),
		three.ZTitle("thickness (mm)"),
	).
		X(scale.Linear(scale.Nice())).
		Y(scale.Linear(scale.Nice())).
		Z(scale.Linear(scale.Nice())).
		Add(three.Scatter3(src, geom.X("l"), geom.Y("w"), geom.Z("t"),
			geom.ColorBy("batch", scale.Qualitative(palette.OkabeIto)), geom.Size(6)))

	return three.New(
		three.Size(720, 560),
		three.Title("Two batches, one floor"),
		three.Theme(theme.Light),
	).Scene(sc)
}

// smithSphereScene sweeps the input impedance of a one-port with a negative
// resistance near resonance — the device an oscillator is built from — and
// marks where the resistance changes sign.
func smithSphereScene() *three.Plot {
	impedance := func(w float64) (r, x float64) {
		r = 1.2 - 1.8*math.Exp(-math.Pow((w-1)/0.25, 2))
		x = 2 * (w - 1/w)
		return r, x
	}
	var r, x, one, crossR, crossX, crossOne []float64
	prev := 0.0
	for k := 0; k <= 400; k++ {
		ri, xi := impedance(0.25 + 2.75*float64(k)/400)
		r, x, one = append(r, ri), append(x, xi), append(one, 1)
		if k > 0 && (prev < 0) != (ri < 0) {
			crossR, crossX, crossOne = append(crossR, ri), append(crossX, xi), append(crossOne, 1)
		}
		prev = ri
	}
	sweep := figure.NewTable().Float64("r", r).Float64("x", x).Float64("one", one)
	marks := figure.NewTable().Float64("r", crossR).Float64("x", crossX).Float64("one", crossOne)

	sc := three.NewScene(three.Spherical(three.Smith())).
		Z(scale.Linear(scale.Domain(0, 1))).
		Add(
			three.Line3(sweep, geom.X("r"), geom.Y("x"), geom.Z("one"),
				geom.Color(palette.Blue), geom.Width(2), geom.Label("z(ω)")),
			three.Scatter3(marks, geom.X("r"), geom.Y("x"), geom.Z("one"),
				geom.Color(palette.Vermilion), geom.Size(8), geom.Droplines(false),
				geom.Label("r = 0")),
		)
	return three.New(
		three.Size(560, 560),
		three.Title("A negative resistance, on the Smith sphere"),
		three.Theme(theme.Light),
	).Scene(sc)
}

// antennaScene is four isotropic elements along x, half a wavelength apart,
// each with a cosine element pattern over a ground plane.
func antennaScene() *three.Plot {
	const (
		elements = 4
		spacing  = 0.5 // wavelengths
	)
	gain := func(phi, theta float64) float64 {
		th, ph := theta*math.Pi/180, phi*math.Pi/180
		psi := 2 * math.Pi * spacing * math.Sin(th) * math.Cos(ph)
		af := 1.0
		if s := math.Sin(psi / 2); math.Abs(s) > 1e-9 {
			af = math.Abs(math.Sin(elements*psi/2) / (elements * s))
		}
		el := math.Max(math.Cos(th), 0)
		return (af * el) * (af * el)
	}
	var phi, theta, g []float64
	for t := 0; t <= 180; t += 5 {
		for p := 0; p < 360; p += 5 {
			phi, theta = append(phi, float64(p)), append(theta, float64(t))
			g = append(g, gain(float64(p), float64(t)))
		}
	}
	src := figure.NewTable().Float64("phi", phi).Float64("theta", theta).Float64("gain", g)

	sc := three.NewScene(three.Spherical(three.AxisEnds("x", "", "y", "", "z", ""))).
		Z(scale.Linear(scale.Domain(0, 1))).
		Add(three.Surface(src, geom.X("phi"), geom.Y("theta"), geom.Z("gain"),
			geom.ColorBy("gain", scale.Sequential(palette.Viridis))))
	return three.New(
		three.Size(620, 560),
		three.Title("A four-element broadside array"),
		three.Theme(theme.Light),
	).Scene(sc)
}

// blochScene drives a qubit off resonance: from |0⟩ it rotates about an axis
// tilted from x towards z by the detuning, and closes the loop after one
// period.
func blochScene() *three.Plot {
	const (
		detuning = 0.6 // of the Rabi frequency
		// Two dozen states is a coarse sweep, and it stays on the ball: two
		// rows a step apart are joined along the great circle between them
		// rather than by a chord through the inside of the sphere.
		steps = 24
	)
	omega := math.Hypot(1, detuning)
	ax, az := 1/omega, detuning/omega

	var phi, theta, r []float64
	for k := 0; k <= steps; k++ {
		t := 2 * math.Pi * float64(k) / steps
		// Rodrigues' rotation of (0, 0, 1) about (ax, 0, az) by t.
		c, s := math.Cos(t), math.Sin(t)
		x := ax * az * (1 - c)
		y := -ax * s
		z := c + az*az*(1-c)
		phi = append(phi, math.Mod(math.Atan2(y, x)*180/math.Pi+360, 360))
		theta = append(theta, math.Acos(math.Max(-1, math.Min(1, z)))*180/math.Pi)
		r = append(r, 1)
	}
	path := figure.NewTable().Float64("phi", phi).Float64("theta", theta).Float64("r", r)
	ends := figure.NewTable().
		Float64("phi", []float64{phi[0], phi[steps/2]}).
		Float64("theta", []float64{theta[0], theta[steps/2]}).
		Float64("r", []float64{1, 1})

	// ASCII brackets: the rasterizer's embedded fonts have no ⟨ or ⟩, and a
	// rune no face can draw is written as a question mark, so the ket would
	// read |0? here. A chart that wants the printed spelling supplies a font
	// that has them through fynefigure.FallbackFont.
	sc := three.NewScene(three.Spherical(three.AxisEnds("|+>", "|->", "|+i>", "|-i>", "|0>", "|1>"))).
		Z(scale.Linear(scale.Domain(0, 1))).
		Add(
			three.Line3(path, geom.X("phi"), geom.Y("theta"), geom.Z("r"),
				geom.Color(palette.Blue), geom.Width(2), geom.Label("state")),
			three.Scatter3(ends, geom.X("phi"), geom.Y("theta"), geom.Z("r"),
				geom.Color(palette.Vermilion), geom.Size(9), geom.Label("start, half period")),
		)
	return three.New(
		three.Size(560, 560),
		three.Title("A detuned Rabi oscillation"),
		three.Theme(theme.Light),
	).Scene(sc)
}

// ribbonScene is an aircraft's approach: a descending spiral that passes over
// its own track twice, which is what a band can say and a stroke cannot.
func ribbonScene() *three.Plot {
	const steps = 420
	var east, north, alt []float64
	for k := range steps + 1 {
		t := float64(k) / steps
		turns := 2.4 * 2 * math.Pi * t
		radius := 9 - 5.5*t
		east = append(east, radius*math.Cos(turns))
		north = append(north, radius*math.Sin(turns))
		alt = append(alt, 3200-2900*t)
	}
	src := figure.NewTable().Float64("east", east).Float64("north", north).Float64("alt", alt)

	sc := three.NewScene(
		three.XTitle("east (km)"),
		three.YTitle("north (km)"),
		three.ZTitle("altitude (ft)"),
	).
		X(scale.Linear(scale.Nice())).
		Y(scale.Linear(scale.Nice())).
		Z(scale.Linear(scale.Nice())).
		// The width is in scene units and carries no reading: a band whose
		// width meant an interval would be a flat line with a band under it.
		Add(three.Ribbon(src, geom.X("east"), geom.Y("north"), geom.Z("alt"),
			geom.Fill(palette.SkyBlue), geom.Thickness(0.045)))

	return three.New(
		three.Size(680, 560),
		three.Title("A holding pattern, down to the runway"),
		three.Theme(theme.Light),
	).Scene(sc).Add(three.View{Camera: three.LookAt(three.Azimuth(-0.8), three.Elevation(0.35))})
}

// jointScene is the orientation of two fracture sets in a rock mass: a strike
// azimuth and a plunge, which together are a direction and nothing else.
//
// There is no Z column. The count is the value, and the ball's own ladder is
// the key — the same shape a radiation pattern is read with.
func jointScene() *three.Plot {
	var strike, plunge []float64
	// A scattered background, sampled so that equal areas of the ball get
	// equal numbers: the azimuth is uniform and the sine of the latitude is,
	// which is the same substitution the binning is built on. Drawn against a
	// lat/long grid it would pile a false ring round the equator.
	for i := range 2000 {
		strike = append(strike, 360*frac(9000+i))
		plunge = append(plunge, math.Asin(2*frac(9700+i)-1)*180/math.Pi)
	}
	// Two fracture sets on top of it, each a spread of directions about a mean.
	sets := []struct{ strike, plunge, spread float64 }{
		{35, 58, 20},
		{128, 12, 24},
	}
	for s, set := range sets {
		for i := range 300 {
			strike = append(strike, math.Mod(set.strike+set.spread*noise(11100+s*900+i)+360, 360))
			p := set.plunge + 0.7*set.spread*noise(12100+s*900+i)
			plunge = append(plunge, math.Max(-90, math.Min(90, p)))
		}
	}
	src := figure.NewTable().Float64("strike", strike).Float64("plunge", plunge)

	// Latitude reads the second angle as the angle up from the equator, which
	// is what a plunge is measured as.
	sc := three.NewScene(three.Spherical(three.Latitude(),
		three.AxisEnds("E", "W", "N", "S", "up", "down"))).
		// Zero keeps the radius proportional to the count, so a cell twice as
		// full stands twice as far out.
		Z(scale.Linear(scale.Zero())).
		// The ramp is painted with the count the layer worked out, so the
		// column named here is a label rather than a column of the table.
		Add(three.Histogram3(src, geom.X("strike"), geom.Y("plunge"),
			geom.Bins(5), geom.ColorBy("count", scale.Sequential(palette.Viridis))))

	return three.New(
		three.Size(580, 560),
		three.Title("Two fracture sets, by orientation"),
		three.Theme(theme.Light),
	).Scene(sc)
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
