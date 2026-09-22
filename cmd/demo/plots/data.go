package plots

import (
	"fmt"
	"math"
	"math/cmplx"
	"sync"
	"time"

	"github.com/timzifer/figure"
	"github.com/timzifer/figure/data"
	"github.com/timzifer/figure/theme"
)

// Sample data for the catalogue. Everything is a fixed formula or recurrence
// rather than math/rand, so a chart looks the same every time it is rebuilt.

// --- generic ------------------------------------------------------------------

func ramp(lo, hi float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		if n == 1 {
			out[i] = lo
			continue
		}
		out[i] = lo + (hi-lo)*float64(i)/float64(n-1)
	}
	return out
}

func apply(xs []float64, fn func(float64) float64) []float64 {
	out := make([]float64, len(xs))
	for i, x := range xs {
		out[i] = fn(x)
	}
	return out
}

// noise is a deterministic pseudo-normal deviate: the Irwin–Hall sum of twelve
// values from a fixed linear congruential sequence, minus six.
func noise(i int) float64 {
	v := uint64(i)*2862933555777941757 + 3037000493
	sum := 0.0
	for range 12 {
		v = v*6364136223846793005 + 1442695040888963407
		sum += float64(v>>11) / float64(uint64(1)<<53)
	}
	return sum - 6
}

func lognormalAt(i int, mu, sigma float64) float64 {
	return math.Exp(mu + sigma*noise(i))
}

// bareLayout is the theme for a chart whose axes mean nothing to the reader:
// a pie, a gauge, a layout in the unit square.
func bareLayout(t theme.Theme) theme.Theme {
	return t.With(
		theme.Grid(false, false),
		theme.AxisLines(false, false),
		theme.Ticks(false, false),
	)
}

// --- basics -------------------------------------------------------------------

func damped() ([]time.Time, []float64) {
	const n = 240
	start := time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)
	times := make([]time.Time, n)
	values := make([]float64, n)
	for i := range n {
		times[i] = start.Add(time.Duration(i) * 15 * time.Second)
		x := float64(i) / n
		values[i] = math.Exp(-2*x) * math.Sin(12*math.Pi*x)
	}
	return times, values
}

func clusters() (xs, a, b []float64) {
	const n = 45
	xs = make([]float64, n)
	a = make([]float64, n)
	b = make([]float64, n)
	for i := range n {
		t := float64(i) / n
		xs[i] = t * 10
		a[i] = 4 + 3*math.Sin(7*t) + 0.8*math.Sin(31*t)
		b[i] = 9 - 2.5*math.Cos(5*t) + 0.6*math.Cos(23*t)
	}
	return xs, a, b
}

// latencyTrace is an hour of latency with two excursions, one well past the
// second boundary.
func latencyTrace() *data.Table {
	const n = 120
	minutes := make([]float64, n)
	ms := make([]float64, n)
	for i := range n {
		t := float64(i)
		minutes[i] = t / 2
		ms[i] = 170 + 25*math.Sin(t/7)
		switch {
		case i >= 30 && i < 44:
			ms[i] += 70
		case i >= 78 && i < 88:
			ms[i] += 160
		}
	}
	return figure.NewTable().Float64("minute", minutes).Float64("ms", ms)
}

// stateRows is what a machine was doing, one row per change.
func stateRows() *data.Table {
	return figure.NewTable().
		Float64("minute", []float64{0, 12, 18, 27, 34, 48, 60}).
		Float64("rate", []float64{82, 82, 0, 0, 46, 79, 79}).
		String("state", []string{
			"running", "running", "fault", "fault", "maintenance", "running", "running",
		})
}

// --- axes & scales ------------------------------------------------------------

// bigSignal is n samples of a noisy trace with a one-sample spike and a NaN
// dropout, both of which a min/max decimation must keep.
func bigSignal(n int) figure.Source {
	t := make([]float64, n)
	v := make([]float64, n)
	for i := range n {
		x := float64(i) / float64(n)
		t[i] = float64(i)
		v[i] = math.Sin(20*math.Pi*x) + 0.3*math.Sin(701*math.Pi*x)
	}
	v[n/3] = 4.2
	for i := range min(5000, n/4) {
		v[n/2+i] = math.NaN()
	}
	return figure.Float64Columns(map[string][]float64{"i": t, "v": v})
}

// --- groups & stacking --------------------------------------------------------

func ledger() (quarters, products []string, revenue []float64) {
	for q, quarter := range []string{"Q1", "Q2", "Q3", "Q4"} {
		for i, product := range []string{"prism", "lens", "filter"} {
			quarters = append(quarters, quarter)
			products = append(products, product)
			revenue = append(revenue, 20+float64(q)*4+float64(i)*9-float64(q*i))
		}
	}
	return quarters, products, revenue
}

func channels() (days []float64, names []string, visits []float64) {
	for d := range 56 {
		for i, name := range []string{"search", "social", "direct", "email"} {
			t := float64(d) / 8
			days = append(days, float64(d))
			names = append(names, name)
			visits = append(visits, math.Max(40+30*math.Sin(t/2+float64(i)*1.7)+8*math.Sin(t*3+float64(i)), 1))
		}
	}
	return days, names, visits
}

func switchboard() (days, hours []string, calls []float64) {
	for d, name := range []string{"mon", "tue", "wed", "thu", "fri"} {
		for h := 8; h < 18; h++ {
			days = append(days, name)
			hours = append(hours, fmt.Sprintf("%02d:00", h))
			v := 60*math.Exp(-math.Pow(float64(h)-10, 2)/6) +
				45*math.Exp(-math.Pow(float64(h)-15, 2)/8)
			calls = append(calls, v*(1-float64(d)*0.12))
		}
	}
	return days, hours, calls
}

// machineTrace is an hour of line speed with a stoppage in the middle.
func machineTrace() ([]time.Time, []float64) {
	const n = 240
	start := time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)
	times := make([]time.Time, n)
	values := make([]float64, n)
	for i := range n {
		times[i] = start.Add(time.Duration(i) * 15 * time.Second)
		switch {
		case i < 90:
			values[i] = 120 + 2*math.Sin(float64(i)/6)
		case i < 130:
			values[i] = 38 + math.Sin(float64(i)/3)
		default:
			values[i] = 119 + 2*math.Sin(float64(i)/6)
		}
	}
	return times, values
}

func machineStates(start time.Time) *data.Table {
	at := func(step int) time.Time { return start.Add(time.Duration(step) * 15 * time.Second) }
	return figure.NewTable().
		Time("start", []time.Time{at(0), at(90), at(130)}).
		Time("end", []time.Time{at(90), at(130), at(240)}).
		String("state", []string{"running", "fault", "running"})
}

func workOrders(start time.Time) *data.Table {
	at := func(step int) time.Time { return start.Add(time.Duration(step) * 15 * time.Second) }
	return figure.NewTable().
		Time("start", []time.Time{at(0), at(110), at(180)}).
		Time("end", []time.Time{at(110), at(180), at(240)}).
		String("lane", []string{"order", "order", "order"}).
		String("order", []string{"WO-4471", "WO-4472", "WO-4473"})
}

func speedRanges() *data.Table {
	return data.NewTable().
		Float64("lo", []float64{0, 100}).
		Float64("hi", []float64{100, 130}).
		String("lane", []string{"band", "band"}).
		String("range", []string{"below target", "at target"})
}

// meterBase is a feeder's idle draw in kilowatts, which is the origin the
// horizon chart's fold is measured from, and meterBand is one band of it.
//
// The band is pinned rather than cut from the data: 30 kW means 30 kW in this
// chart and in the next one drawn this way, where a band taken as a share of
// each day's own maximum would make two days incomparable — which is the one
// thing the form exists to prevent.
const (
	meterBase = 120.0
	meterBand = 30.0
)

// meterLoad is six hours of metered load for one feeder, about meterBase.
//
// It is the series a horizon chart is for: a slow shift swell, a faster machine
// cycle on top of it, and one short overload that a trace this height would
// flatten into the rest of the line.
func meterLoad() *data.Table {
	const n = 360
	start := time.Date(2026, time.September, 14, 6, 0, 0, 0, time.UTC)
	times := make([]time.Time, n)
	kw := make([]float64, n)
	for i := range n {
		x := float64(i) / float64(n-1)
		v := meterBase + 52*math.Sin(2*math.Pi*x) + 16*math.Sin(23*math.Pi*x)
		// The overload: a couple of bands deep and a few minutes wide.
		if d := float64(i) - 0.62*n; math.Abs(d) < 9 {
			v += 95 * (1 - math.Abs(d)/9)
		}
		times[i] = start.Add(time.Duration(i) * time.Minute)
		kw[i] = v
	}
	return figure.NewTable().Time("t", times).Float64("kw", kw)
}

// --- distributions ------------------------------------------------------------

func cohorts() ([]string, []float64) {
	names := []string{"alpha", "beta", "gamma"}
	var keys []string
	var vals []float64
	for g, name := range names {
		for i := range 40 {
			t := float64(i) / 40
			v := 24 + float64(g)*7 + 9*math.Sin(9*t+float64(g)) + 3*math.Sin(37*t) + 2*math.Cos(61*t)
			keys, vals = append(keys, name), append(vals, v)
		}
		keys, vals = append(keys, name), append(vals, 62+float64(g)*5)
	}
	return keys, vals
}

func latencies() []float64 {
	out := make([]float64, 2000)
	for i := range out {
		out[i] = lognormalAt(i*11+11, 3.6, 0.55)
	}
	return out
}

func serviceLatencies() *data.Table {
	names := []string{"auth", "search", "checkout"}
	regions := []string{"eu", "us"}
	var vals []float64
	var svc, region []string
	for i := range 900 {
		s := i % 3
		r := (i / 3) % 2
		v := lognormalAt(i, 3.2+0.35*float64(s)+0.2*float64(r), 0.4)
		if s == 2 && i%7 == 0 {
			v *= 3
		}
		vals = append(vals, v)
		svc = append(svc, names[s])
		region = append(region, regions[r])
	}
	return figure.NewTable().Float64("ms", vals).String("service", svc).String("region", region)
}

func monthlyTemperatures() (months []string, src *data.Table) {
	months = []string{
		"jan", "feb", "mar", "apr", "may", "jun",
		"jul", "aug", "sep", "oct", "nov", "dec",
	}
	var temps []float64
	var month []string
	for m, name := range months {
		mid := 11 + 9*math.Sin(2*math.Pi*(float64(m)-3)/12)
		spread := 3.5 + 1.5*math.Cos(2*math.Pi*float64(m)/12)
		for i := range 220 {
			temps = append(temps, mid+spread*noise(m*997+i))
			month = append(month, name)
		}
	}
	return months, figure.NewTable().Float64("degrees", temps).String("month", month)
}

func cohortScores() *data.Table {
	names := []string{"control", "variant A", "variant B"}
	sizes := []int{60, 60, 9}
	var scores []float64
	var cohort []string
	for c, name := range names {
		for i := range sizes[c] {
			scores = append(scores, 50+float64(c)*6+9*noise(c*613+i))
			cohort = append(cohort, name)
		}
	}
	return figure.NewTable().Float64("score", scores).String("cohort", cohort)
}

func hexcloud(n int) data.Source {
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range n {
		x := 10 * float64(i) / float64(n-1)
		xs[i] = x
		ys[i] = math.Sin(x)*3 + x/3 + 1.2*noise(i)
	}
	return figure.Float64Columns(map[string][]float64{"x": xs, "y": ys})
}

func nations() *data.Table {
	return figure.NewTable().
		Float64("income", []float64{1500, 4200, 12800, 31000, 46000, 58000, 9800, 24000}).
		Float64("years", []float64{58, 66, 72, 78, 81, 82, 70, 76}).
		Float64("people", []float64{212, 1420, 274, 51, 68, 335, 84, 38}).
		String("region", []string{
			"Africa", "Asia", "Asia", "Europe",
			"Europe", "Americas", "Africa", "Americas",
		})
}

// cloud is a correlated point cloud, drawn from splitmix64.
func cloud(n int) (xs, ys []float64) {
	xs = make([]float64, n)
	ys = make([]float64, n)
	var seed uint64 = 0x9e3779b97f4a7c15
	next := func() float64 {
		seed += 0x9e3779b97f4a7c15
		z := seed
		z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
		z = (z ^ (z >> 27)) * 0x94d049bb133111eb
		z ^= z >> 31
		return float64(z>>11) / (1 << 53)
	}
	for i := range n {
		u, v := next(), next()
		r := math.Sqrt(-2 * math.Log(u+1e-15))
		a, b := r*math.Cos(2*math.Pi*v), r*math.Sin(2*math.Pi*v)
		xs[i] = a
		ys[i] = 0.65*a + 0.76*b
	}
	return xs, ys
}

// trendCloud is n noisy points about a curve that is mostly, but not quite,
// a straight line — so the loess and the linear fit visibly disagree.
func trendCloud(n int) data.Source {
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range n {
		// Spread the abscissae unevenly, the way real measurements fall.
		x := 10 * math.Pow(float64(i)/float64(n-1), 1.2)
		xs[i] = x
		ys[i] = 2 + 0.6*x + 1.6*math.Sin(x/1.4) + 0.9*noise(i*7+3)
	}
	return figure.Float64Columns(map[string][]float64{"x": xs, "y": ys})
}

// --- fields -------------------------------------------------------------------

// response is a device's small-signal gain over bias and drive: a diagonal
// ridge that rolls off at both ends.
func response() figure.Source {
	const n = 40
	bias := make([]float64, 0, n*n)
	drive := make([]float64, 0, n*n)
	gain := make([]float64, 0, n*n)
	for j := range n {
		for i := range n {
			b := -1 + 2*float64(i)/(n-1)
			d := -20 + 30*float64(j)/(n-1)
			bias, drive = append(bias, b), append(drive, d)
			gain = append(gain, 4-0.02*(d+6)*(d+6)-6*(b-0.1)*(b-0.1)+1.5*b*math.Cos(d/6))
		}
	}
	return figure.NewTable().
		Float64("bias", bias).Float64("drive", drive).Float64("gain", gain)
}

// --- polar --------------------------------------------------------------------

func browserShare() (names []string, share []float64) {
	return []string{"chrome", "safari", "firefox", "edge", "other"},
		[]float64{46, 24, 14, 11, 5}
}

func budgets() (teams []string, share, floor, used, pull []float64) {
	return []string{"platform", "data", "growth", "support", "design"},
		[]float64{32, 24, 18, 14, 12},
		[]float64{0.35, 0.35, 0.35, 0.35, 0.35},
		[]float64{0.92, 0.78, 1, 0.55, 0.66},
		[]float64{0, 0, 0.12, 0, 0}
}

func designScores() (axisNames, designs []string, scores []float64) {
	byDesign := map[string][]float64{
		"prism": {8, 6, 9, 4, 7},
		"lens":  {5, 9, 6, 8, 5},
	}
	for _, design := range []string{"prism", "lens"} {
		for i, axis := range []string{"speed", "clarity", "range", "cost", "weight"} {
			axisNames = append(axisNames, axis)
			designs = append(designs, design)
			scores = append(scores, byDesign[design][i])
		}
	}
	return axisNames, designs, scores
}

func wind() (points []string, hours []float64) {
	points = []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	for i := range points {
		t := float64(i) / float64(len(points)) * 2 * math.Pi
		hours = append(hours, math.Round(600+420*math.Cos(t-0.9)))
	}
	return points, hours
}

// --- smith --------------------------------------------------------------------

// s11Sweep is a synthesised reflection measurement of a patch antenna: a
// parallel RLC resonance behind its feed inductance, swept across its band.
func s11Sweep(n int) (re, im []float64) {
	const (
		z0 = 50.0
		r  = 50.0
		l  = 0.663e-9
		c  = 6.63e-12
		lf = 1.0e-9
	)
	re, im = make([]float64, n), make([]float64, n)
	for i := range n {
		f := 2.0e9 + 0.8e9*float64(i)/float64(n-1)
		w := 2 * math.Pi * f
		y := complex(1/r, w*c-1/(w*l))
		z := (complex(0, w*lf) + 1/y) / z0
		g := (z - 1) / (z + 1)
		re[i], im[i] = real(g), imag(g)
	}
	return re, im
}

// bestMatch is the index of the sample closest to the middle of the chart.
func bestMatch(re, im []float64) int {
	best, at := math.Inf(1), 0
	for i := range re {
		if d := math.Hypot(re[i], im[i]); d < best {
			best, at = d, i
		}
	}
	return at
}

// matchLocus is the two-element L network that takes 15 − j25 Ω to the centre:
// the impedances a series inductance walks through, then a shunt capacitance.
func matchLocus(n int) (series, shunt []complex128) {
	load := complex(15, -25) / 50
	r := real(load)
	end := complex(r, math.Sqrt(r*(1-r)))
	series = make([]complex128, n)
	for i := range n {
		t := float64(i) / float64(n-1)
		series[i] = complex(r, imag(load)+t*(imag(end)-imag(load)))
	}
	y0, y1 := 1/end, complex(1, 0)
	shunt = make([]complex128, n)
	for i := range n {
		t := float64(i) / float64(n-1)
		y := complex(real(y0), imag(y0)+t*(imag(y1)-imag(y0)))
		shunt[i] = 1 / y
	}
	return series, shunt
}

func realParts(vs []complex128) []float64 {
	out := make([]float64, len(vs))
	for i, v := range vs {
		out[i] = real(v)
	}
	return out
}

func imagParts(vs []complex128) []float64 {
	out := make([]float64, len(vs))
	for i, v := range vs {
		out[i] = imag(v)
	}
	return out
}

// --- nichols ------------------------------------------------------------------

// openLoop is the plant the Nichols charts draw: an integrator and two lags.
func openLoop(w, k float64) complex128 {
	s := complex(0, w)
	return complex(k, 0) / (s * (1 + s/2) * (1 + s/10))
}

func loopOmega(t float64) float64 { return math.Pow(10, -1+4*t) }

// loopSweep is the open loop over four decades as phase in degrees and gain in
// dB, the phase unwrapped so the response does not leap across −180°.
func loopSweep(k float64) (phase, gain []float64) {
	const n = 400
	phase, gain = make([]float64, n), make([]float64, n)
	for i := range n {
		l := openLoop(loopOmega(float64(i)/float64(n-1)), k)
		phase[i] = cmplx.Phase(l) * 180 / math.Pi
		if i > 0 {
			phase[i] -= 360 * math.Round((phase[i]-phase[i-1])/360)
		}
		gain[i] = 20 * math.Log10(cmplx.Abs(l))
	}
	return phase, gain
}

type loopReading struct{ phase, gain, peak, w float64 }

// loopPeak is where the closed loop resonates, and what the open loop was
// doing there.
func loopPeak(k float64) loopReading {
	best := loopReading{peak: math.Inf(-1)}
	for i := range 4001 {
		w := loopOmega(float64(i) / 4000)
		l := openLoop(w, k)
		if db := 20 * math.Log10(cmplx.Abs(l/(1+l))); db > best.peak {
			phase := cmplx.Phase(l) * 180 / math.Pi
			if phase > 0 {
				phase -= 360
			}
			best = loopReading{phase: phase, gain: 20 * math.Log10(cmplx.Abs(l)), peak: db, w: w}
		}
	}
	return best
}

// nicholsTangent is the loop gain whose closed loop peaks at exactly 3 dB, by
// bisection, so the chart cannot drift from its note. It is solved once: the
// plots are built fresh on every call, the number they share is not.
var nicholsTangent = sync.OnceValue(func() float64 {
	lo, hi := 0.01, 100.0
	for range 64 {
		k := (lo + hi) / 2
		if loopPeak(k).peak < 3 {
			lo = k
		} else {
			hi = k
		}
	}
	return (lo + hi) / 2
})

// --- relational ---------------------------------------------------------------

func diskUsage() figure.Source {
	return figure.NewTable().
		String("path", []string{
			"/", "src", "docs", "test",
			"geom", "scale", "coord", "render",
			"guide", "adr",
			"unit", "golden",
		}).
		String("under", []string{
			"", "/", "/", "/",
			"src", "src", "src", "src",
			"docs", "docs",
			"test", "test",
		}).
		String("group", []string{
			"", "src", "docs", "test",
			"src", "src", "src", "src",
			"docs", "docs",
			"test", "test",
		}).
		Float64("kb", []float64{
			0, 0, 0, 0,
			420, 180, 260, 310,
			150, 90,
			200, 110,
		})
}

func requestFlow() figure.Source {
	return figure.NewTable().
		String("from", []string{"web", "web", "mobile", "mobile", "api", "api", "api"}).
		String("to", []string{"api", "cdn", "api", "cdn", "cache", "db", "search"}).
		Float64("rps", []float64{620, 180, 340, 120, 500, 300, 160})
}

// --- layout -------------------------------------------------------------------

func fleet() (regions []string, hours, rps []float64) {
	for ri, name := range []string{"north", "south", "east", "west", "central"} {
		for h := range 24 {
			regions = append(regions, name)
			hours = append(hours, float64(h))
			base := 40 + 12*float64(ri)
			rps = append(rps, base+18*math.Sin(float64(h)/3.8+float64(ri)))
		}
	}
	return regions, hours, rps
}

// --- 3D -----------------------------------------------------------------------

// lorentz is one resonance line in a spectrum.
func lorentz(f, at, top, width float64) float64 {
	d := (f - at) / width
	return top - 10*math.Log10(1+d*d)
}
