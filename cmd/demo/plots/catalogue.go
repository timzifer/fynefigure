package plots

import (
	"fmt"
	"math"
	"time"

	"github.com/timzifer/figure"
	"github.com/timzifer/figure/coord"
	"github.com/timzifer/figure/facet"
	"github.com/timzifer/figure/geom"
	"github.com/timzifer/figure/ir"
	"github.com/timzifer/figure/mathtext"
	"github.com/timzifer/figure/palette"
	"github.com/timzifer/figure/scale"
	"github.com/timzifer/figure/stat"
	"github.com/timzifer/figure/theme"
)

// --- Basics -----------------------------------------------------------------

func basics() []Entry {
	g := GroupBasics
	return []Entry{
		{ID: "signal", Group: g, Title: "Signal (time axis)",
			Note: "A damped oscillation on a time axis, smoothed with a curve tension; zoom in to see the tick labels change unit.",
			Plot: flat("Signal", 800, 400, theme.Dark, func(p *figure.Plot) {
				times, values := damped()
				src := figure.NewTable().Time("t", times).Float64("y", values)
				p.X(scale.Time())
				p.Y(scale.Linear(scale.Nice()))
				p.Add(geom.Line(src, geom.X("t"), geom.Y("y"),
					geom.Color(palette.SkyBlue), geom.Tension(0.4)))
			})},
		{ID: "series", Group: g, Title: "Three series",
			Note: "Three lines told apart by colour and dash; click legend entries or hover to compare values.",
			Plot: flat("Three series", 800, 400, theme.Light, func(p *figure.Plot) {
				xs := ramp(0, 10, 120)
				src := figure.Float64Columns(map[string][]float64{
					"x":      xs,
					"linear": apply(xs, func(x float64) float64 { return x }),
					"square": apply(xs, func(x float64) float64 { return x * x / 10 }),
					"root":   apply(xs, func(x float64) float64 { return 3 * math.Sqrt(x) }),
				})
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(
					geom.Line(src, geom.X("x"), geom.Y("linear"), geom.Label("linear")),
					geom.Line(src, geom.X("x"), geom.Y("square"), geom.Label("square"), geom.Dash(6, 4)),
					geom.Line(src, geom.X("x"), geom.Y("root"), geom.Label("root"), geom.Dash(2, 3)),
				)
			})},
		{ID: "scatter", Group: g, Title: "Scatter",
			Note: "Two point clouds with different marker shapes; hover a marker to read its row.",
			Plot: flat("Scatter", 700, 420, theme.Light, func(p *figure.Plot) {
				xs, a, b := clusters()
				src := figure.Float64Columns(map[string][]float64{"x": xs, "a": a, "b": b})
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice()))
				p.Add(
					geom.Scatter(src, geom.X("x"), geom.Y("a"), geom.Label("group A"),
						geom.Shape(ir.MarkerCircle), geom.Size(7)),
					geom.Scatter(src, geom.X("x"), geom.Y("b"), geom.Label("group B"),
						geom.Shape(ir.MarkerDiamond), geom.Size(7)),
				)
			})},
		{ID: "bars", Group: g, Title: "Bars (numeric X)",
			Note: "Bars at bin centres on a continuous axis, where the gaps between bins carry meaning.",
			Plot: flat("Response time distribution", 700, 400, theme.Light, func(p *figure.Plot) {
				bins := ramp(5, 95, 10)
				counts := []float64{18, 47, 82, 96, 71, 44, 25, 13, 6, 2}
				src := figure.Float64Columns(map[string][]float64{"ms": bins, "count": counts})
				p.X(scale.Linear())
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(geom.Bar(src, geom.X("ms"), geom.Y("count"), geom.Color(palette.Green)))
			})},
		{ID: "area", Group: g, Title: "Area band",
			Note: "An interval drawn as an area between two columns, with the estimate as a line over it.",
			Plot: flat("Estimate and interval", 700, 400, theme.Light, func(p *figure.Plot) {
				xs := ramp(0, 12, 120)
				src := figure.Float64Columns(map[string][]float64{
					"x":  xs,
					"y":  apply(xs, func(x float64) float64 { return math.Sin(x) + x/6 }),
					"lo": apply(xs, func(x float64) float64 { return math.Sin(x) + x/6 - 0.3 - x/20 }),
					"hi": apply(xs, func(x float64) float64 { return math.Sin(x) + x/6 + 0.3 + x/20 }),
				})
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice()))
				p.Add(
					geom.Area(src, geom.X("x"), geom.Y("hi"), geom.Y2("lo"),
						geom.Label("interval"), geom.Color(palette.SkyBlue), geom.Width(1)),
					geom.Line(src, geom.X("x"), geom.Y("y"),
						geom.Label("estimate"), geom.Color(palette.Blue)),
				)
			})},
		{ID: "steps", Group: g, Title: "Steps",
			Note: "A value held until it changes, drawn as a step line.",
			Plot: flat("Replicas over the day", 700, 380, theme.Dark, func(p *figure.Plot) {
				hours := ramp(0, 23, 24)
				replicas := []float64{2, 2, 2, 2, 2, 3, 5, 8, 12, 14, 14, 13, 13, 14, 15, 15, 13, 10, 7, 5, 4, 3, 2, 2}
				src := figure.Float64Columns(map[string][]float64{"hour": hours, "n": replicas})
				p.X(scale.Linear())
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(geom.Step(src, geom.X("hour"), geom.Y("n"),
					geom.Color(palette.Yellow), geom.Width(2)))
			})},
		{ID: "categories", Group: g, Title: "Categories",
			Note: "Bars on an ordinal axis, coloured from their own value through a sequential ramp.",
			Plot: flat("Sales by region", 700, 400, theme.Light, func(p *figure.Plot) {
				src := figure.NewTable().
					String("region", []string{"north", "south", "east", "west", "central", "overseas"}).
					Float64("sales", []float64{18, 42, 31, 25, 37, 12})
				p.X(scale.Ordinal())
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(geom.Bar(src, geom.X("region"), geom.Y("sales"),
					geom.ColorBy("sales", scale.Sequential(palette.Viridis))))
			})},
		{ID: "errorbars", Group: g, Title: "Error bars",
			Note: "Faded bars with solid 95 % intervals over them, and a number format on the axis.",
			Plot: flat("Mean latency, with its 95 % interval", 700, 400, theme.Light, func(p *figure.Plot) {
				src := figure.NewTable().
					String("service", []string{"auth", "search", "cart", "checkout", "media"}).
					Float64("mean", []float64{42, 118, 63, 91, 210}).
					Float64("ci", []float64{6, 22, 9, 14, 38})
				p.X(scale.Ordinal())
				p.Y(scale.Linear(scale.Nice(), scale.Zero(), scale.NumberFormat("# ms")))
				ink := palette.OkabeIto.At(0)
				p.Add(
					geom.Bar(src, geom.X("service"), geom.Y("mean"),
						geom.Fill(palette.Lerp(ir.RGB(255, 255, 255), ink, 0.3)),
						geom.Label("mean")),
					geom.ErrorBar(src, geom.X("service"), geom.Y("mean"), geom.ErrorBy("ci"),
						geom.Color(ink), geom.Width(1.5)),
				)
			})},
		{ID: "label-placement", Group: g, Title: "Label placement",
			Note: "Four labels on nearly coincident points, placed so they do not overlap; zoom in to watch them settle.",
			Plot: flat("Nearby labels, placed in source order", 640, 400, theme.Light, func(p *figure.Plot) {
				src := figure.NewTable().
					Float64("x", []float64{5, 5.05, 5.1, 5.15}).
					Float64("y", []float64{5, 5.02, 5.04, 5.06}).
					String("name", []string{"Alpha", "Beta", "Gamma", "Delta"})
				p.X(scale.Linear(scale.Domain(0, 10)))
				p.Y(scale.Linear(scale.Domain(0, 10)))
				p.Add(
					geom.Scatter(src, geom.X("x"), geom.Y("y")),
					geom.Text(src, geom.X("x"), geom.Y("y"), geom.TextBy("name"), geom.AvoidOverlap(true)),
				)
			}, figure.Legend(false))},
		{ID: "status-threshold", Group: g, Title: "Threshold-coloured line",
			Note: "A line coloured by a threshold scale: the colour changes exactly where the reading crosses 200 and 300 ms.",
			Plot: flat("Checkout latency against its budget", 760, 420, theme.Light, func(p *figure.Plot) {
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice()))
				limits := scale.Threshold(
					palette.Ramp{palette.Blue, palette.Orange, palette.Red},
					[]float64{200, 300},
				)
				p.Add(
					geom.HBand(200, 300, geom.Extend(false)),
					geom.Line(latencyTrace(), geom.X("minute"), geom.Y("ms"), geom.ColorBy("ms", limits)),
				)
			}, figure.XTitle("minute"), figure.YTitle("ms"))},
		{ID: "status-state", Group: g, Title: "Named-state step",
			Note: "A step coloured by machine state through a named scale, so a fault is always red.",
			Plot: flat("Line 3, by machine state", 760, 420, theme.Light, func(p *figure.Plot) {
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(geom.Step(stateRows(), geom.X("minute"), geom.Y("rate"),
					geom.ColorBy("state", scale.Named(map[string]ir.Color{
						"running":     palette.Green,
						"fault":       palette.Red,
						"maintenance": palette.Orange,
					}))))
			}, figure.XTitle("minute"), figure.YTitle("parts/min"))},
	}
}

// --- Axes & scales ------------------------------------------------------------

func axes() []Entry {
	g := GroupAxes
	return []Entry{
		{ID: "logscale", Group: g, Title: "Log scale",
			Note: "Two exponential curves on a logarithmic Y axis, where they become straight lines.",
			Plot: flat("Requests per second", 700, 400, theme.Dark, func(p *figure.Plot) {
				xs := ramp(0, 14, 140)
				src := figure.Float64Columns(map[string][]float64{
					"week":  xs,
					"rps":   apply(xs, func(x float64) float64 { return 5 * math.Exp(0.72*x) }),
					"floor": apply(xs, func(x float64) float64 { return 5 * math.Exp(0.45*x) }),
				})
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Log(scale.LogNice()))
				p.Add(
					geom.Line(src, geom.X("week"), geom.Y("rps"),
						geom.Label("actual"), geom.Color(palette.Green)),
					geom.Line(src, geom.X("week"), geom.Y("floor"),
						geom.Label("plan"), geom.Color(palette.Orange), geom.Dash(6, 4)),
				)
			})},
		{ID: "twoaxes", Group: g, Title: "Two Y axes",
			Note: "Revenue bars on the left axis and a margin line on a second, percentage-formatted right axis.",
			Plot: flat("Revenue and margin", 760, 420, theme.Light, func(p *figure.Plot) {
				src := figure.NewTable().
					String("month", []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}).
					Float64("revenue", []float64{820, 910, 870, 1040, 1180, 1120}).
					Float64("margin", []float64{0.11, 0.13, 0.09, 0.15, 0.18, 0.16})
				p.X(scale.Ordinal())
				p.Y(scale.Linear(scale.Nice(), scale.Zero(), scale.NumberFormat("#,")))
				p.Y2(scale.Linear(scale.Nice(), scale.Zero(), scale.NumberFormat("#.0%")))
				p.Add(
					geom.Bar(src, geom.X("month"), geom.Y("revenue"),
						geom.Color(palette.OkabeIto.At(0)), geom.Label("revenue")),
					geom.Line(src, geom.X("month"), geom.Y("margin"), geom.OnY2(),
						geom.Color(palette.OkabeIto.At(1)), geom.Width(2), geom.Label("margin")),
				)
			}, figure.YTitle("revenue (k€)"), figure.Y2Title("margin"))},
		{ID: "twoextents", Group: g, Title: "Two X extents",
			Note: "One reading with two rulers: minutes along the bottom and cycles along a pinned top axis.",
			Plot: flat("Oven temperature through a run", 720, 400, theme.Light, func(p *figure.Plot) {
				mins := ramp(0, 120, 240)
				temp := apply(mins, func(t float64) float64 {
					return 20 + 160*(1-math.Exp(-t/18)) - 12*math.Sin(t/4)
				})
				src := figure.Float64Columns(map[string][]float64{"t": mins, "c": temp})
				p.X(scale.Linear(scale.Domain(0, 120)))
				p.X2(scale.Linear(scale.Domain(0, 24)))
				p.Y(scale.Linear(scale.Nice(), scale.Zero(), scale.NumberFormat("# °C")))
				p.Add(
					geom.Area(src, geom.X("t"), geom.Y("c"),
						geom.Fill(palette.Lerp(ir.RGB(255, 255, 255), palette.OkabeIto.At(1), 0.3))),
					geom.Line(src, geom.X("t"), geom.Y("c"),
						geom.Color(palette.OkabeIto.At(1)), geom.Width(1.5)),
				)
			}, figure.XTitle("elapsed (min)"), figure.X2Title("cycle"), figure.Legend(false))},
		{ID: "notation", Group: g, Title: "Math notation",
			Note: "TeX in the title, axis titles and legend, with redundant encoding (dash and shape as well as colour).",
			Plot: flat(`Standard error of $\bar{x}$`, 760, 420, theme.Light.With(theme.Redundant(true)), func(p *figure.Plot) {
				ns := ramp(4, 64, 31)
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				src := figure.Float64Columns(map[string][]float64{
					"n":       ns,
					"sigma-1": apply(ns, func(n float64) float64 { return 1 / math.Sqrt(n) }),
					"sigma-2": apply(ns, func(n float64) float64 { return 2 / math.Sqrt(n) }),
					"sigma-3": apply(ns, func(n float64) float64 { return 3 / math.Sqrt(n) }),
				})
				for i, col := range []string{"sigma-1", "sigma-2", "sigma-3"} {
					p.Add(geom.Line(src, geom.X("n"), geom.Y(col),
						geom.Label(fmt.Sprintf(`$\sigma = %d$`, i+1))))
				}
			},
				figure.Math(mathtext.TeX()),
				figure.XTitle(`sample size $n$`),
				figure.YTitle(`$\frac{\sigma}{\sqrt{n}}$ (mV)`),
				figure.Legend(true),
			)},
		{ID: "decimation", Group: g, Title: "Big data (decimation)",
			Note: "Two million samples reduced min/max per pixel column, so the one-sample spike and the NaN dropout survive; pan and zoom to feel the GPU path.",
			Plot: flat("Two million samples", 900, 420, theme.Light, func(p *figure.Plot) {
				src := bigSignal(2_000_000)
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice()))
				p.Add(geom.Line(src, geom.X("i"), geom.Y("v"),
					geom.Color(palette.Blue), geom.Decimate(geom.MinMax)))
			}, figure.XTitle("sample"), figure.YTitle("volts"))},
	}
}

// --- Groups & stacking --------------------------------------------------------

func stacking() []Entry {
	g := GroupStacking
	return []Entry{
		{ID: "stacked", Group: g, Title: "Stacked bars",
			Note: "One bar layer over a long table: the product column makes the stack and the legend.",
			Plot: flat("Revenue by product", 700, 420, theme.Light, func(p *figure.Plot) {
				quarters, products, revenue := ledger()
				src := figure.NewTable().
					String("quarter", quarters).
					String("product", products).
					Float64("revenue", revenue)
				p.X(scale.Ordinal())
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(geom.Bar(src, geom.X("quarter"), geom.Y("revenue"),
					geom.GroupBy("product"),
					geom.ColorBy("product", scale.Qualitative(palette.OkabeIto))))
			})},
		{ID: "stream", Group: g, Title: "Streamgraph",
			Note: "Stacked areas about a wiggling baseline, ordered inside-out.",
			Plot: flat("Traffic by channel", 760, 400, theme.Light, func(p *figure.Plot) {
				days, names, visits := channels()
				src := figure.NewTable().
					Float64("day", days).
					String("channel", names).
					Float64("visits", visits)
				p.X(scale.Linear())
				p.Y(scale.Linear(scale.Nice()))
				p.Add(geom.Area(src, geom.X("day"), geom.Y("visits"),
					geom.GroupBy("channel"),
					geom.Stack(geom.StackWiggle),
					geom.Order(geom.OrderInsideOut),
					geom.ColorBy("channel", scale.Qualitative(palette.OkabeIto))))
			})},
		{ID: "heatmap", Group: g, Title: "Heatmap",
			Note: "A rect over two ordinal axes coloured by a sequential ramp: a heatmap is nothing more.",
			Plot: flat("Calls per hour", 740, 400, theme.Light, func(p *figure.Plot) {
				days, hours, calls := switchboard()
				src := figure.NewTable().
					String("day", days).
					String("hour", hours).
					Float64("calls", calls)
				p.X(scale.Ordinal(scale.OrdinalPadding(0)))
				p.Y(scale.Ordinal(scale.OrdinalPadding(0)))
				p.Add(geom.Rect(src, geom.X("day"), geom.Y("hour"),
					geom.ColorBy("calls", scale.Sequential(palette.Viridis))))
			})},
		{ID: "gantt", Group: g, Title: "Gantt",
			Note: "Rects naming both horizontal edges on a time axis against an ordinal task axis.",
			Plot: flat("Plan", 760, 300, theme.Light, func(p *figure.Plot) {
				start := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
				day := func(n int) time.Time { return start.AddDate(0, 0, n) }
				src := figure.NewTable().
					String("task", []string{"design", "build", "review", "ship"}).
					Time("from", []time.Time{day(0), day(3), day(9), day(12)}).
					Time("to", []time.Time{day(4), day(10), day(12), day(14)})
				p.X(scale.Time())
				p.Y(scale.Ordinal())
				p.Add(geom.Rect(src, geom.X("from"), geom.X2("to"), geom.Y("task"),
					geom.Fill(palette.SkyBlue), geom.Color(palette.Blue)))
			})},
		{ID: "machine-tracks", Group: g, Title: "Machine tracks",
			Note: "A speed trace with state and work-order gantt tracks under it and a speed-range band beside it, all on one time axis.",
			Plot: flat("Line 3 — last hour", 900, 520, theme.Light, func(p *figure.Plot) {
				times, speed := machineTrace()
				measured := figure.NewTable().Time("t", times).Float64("speed", speed)
				p.X(scale.Time())
				p.Y(scale.Linear(scale.Zero(), scale.Nice()))
				p.Add(
					geom.Line(measured, geom.X("t"), geom.Y("speed"),
						geom.Color(palette.Blue), geom.Label("measured")),
					geom.HLine(120, geom.Label("target")),
				)
				p.Track(figure.Bottom, figure.TrackSize(40)).
					Add(geom.Rect(machineStates(times[0]),
						geom.X("start"), geom.X2("end"), geom.Y("state"),
						geom.ColorBy("state", scale.Qualitative(palette.Default))))
				p.Track(figure.Bottom, figure.TrackSize(28)).
					Add(geom.Rect(workOrders(times[0]),
						geom.X("start"), geom.X2("end"), geom.Y("lane"),
						geom.ColorBy("order", scale.Qualitative(palette.Default))))
				p.Track(figure.Left, figure.TrackSize(18), figure.TrackAxis(false)).
					Add(geom.Rect(speedRanges(),
						geom.Y("lo"), geom.Y2("hi"), geom.X("lane"),
						geom.ColorBy("range", scale.Qualitative(palette.Default))))
			}, figure.YTitle("m/min"))},
		{ID: "horizon", Group: g, Title: "Horizon",
			Note: "A series folded into bands of equal height, each drawn at the panel's full height and told apart by colour: the strip stays readable at a height where a line chart stops being one.",
			Plot: flat("Feeder load — six hours", 820, 220, theme.Light, func(p *figure.Plot) {
				p.X(scale.Time())
				p.Y(scale.Linear())
				// The fold gives the vertical ladder up, so the colourbar is
				// what names the bands — in kilowatts, about the idle draw the
				// fold is measured from.
				p.Add(geom.Horizon(meterLoad(),
					geom.X("t"), geom.Y("kw"),
					geom.BandHeight(meterBand),
					geom.Baseline(meterBase),
					geom.Label("load"),
				))
			}, figure.YTitle("kW from idle"))},
	}
}

// --- Distributions ------------------------------------------------------------

func distributions() []Entry {
	g := GroupDistributions
	return []Entry{
		{ID: "histogram", Group: g, Title: "Histogram",
			Note: "Two thousand lognormal latencies binned by Freedman–Diaconis; the Y axis is trained on computed counts.",
			Plot: flat("Request latency", 700, 400, theme.Light, func(p *figure.Plot) {
				src := figure.Float64Columns(map[string][]float64{"ms": latencies()})
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(geom.Histogram(src, geom.X("ms"), geom.Color(palette.Blue)))
			}, figure.XTitle("milliseconds"), figure.YTitle("requests"))},
		{ID: "boxplot", Group: g, Title: "Boxplot",
			Note: "Three cohorts summarised as quartiles and whiskers, each with an outlier.",
			Plot: flat("Latency by cohort", 700, 400, theme.Light, func(p *figure.Plot) {
				keys, vals := cohorts()
				src := figure.NewTable().String("cohort", keys).Float64("ms", vals)
				p.X(scale.Ordinal())
				p.Y(scale.Linear(scale.Nice()))
				p.Add(geom.Boxplot(src, geom.X("cohort"), geom.Y("ms"), geom.Color(palette.Blue)))
			})},
		{ID: "violin", Group: g, Title: "Violin",
			Note: "Densities split by region with a pinned bandwidth; checkout's second hump is what a boxplot hides.",
			Plot: flat("Latency by service", 760, 420, theme.Light, func(p *figure.Plot) {
				p.X(scale.Ordinal())
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(geom.Violin(serviceLatencies(),
					geom.X("service"), geom.Y("ms"),
					geom.GroupBy("region"), geom.Bandwidth(3),
					geom.ColorBy("region", scale.Qualitative(palette.OkabeIto))))
			}, figure.YTitle("milliseconds"), figure.Legend(true))},
		{ID: "ridgeline", Group: g, Title: "Ridgeline",
			Note: "Twelve overlapping monthly densities down an ordinal axis pinned in calendar order.",
			Plot: flat("Daily maximum, by month", 700, 520, theme.Light, func(p *figure.Plot) {
				months, src := monthlyTemperatures()
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Ordinal(scale.Categories(months...)))
				p.Add(geom.Ridgeline(src, geom.X("degrees"), geom.Y("month"),
					geom.Overlap(2.2), geom.Color(palette.Blue)))
			}, figure.XTitle("degrees"))},
		{ID: "beeswarm", Group: g, Title: "Beeswarm",
			Note: "Every observation placed deterministically without overlap, including a cohort of only nine.",
			Plot: flat("Scores by cohort", 700, 400, theme.Light, func(p *figure.Plot) {
				p.X(scale.Ordinal())
				p.Y(scale.Linear(scale.Domain(30, 85)))
				p.Add(geom.Beeswarm(cohortScores(), geom.X("cohort"), geom.Y("score"),
					geom.Size(7), geom.Color(palette.Blue)))
			}, figure.YTitle("score"))},
		{ID: "ecdf", Group: g, Title: "ECDF",
			Note: "Cumulative distributions per cohort: no bins, no bandwidth, nothing hidden.",
			Plot: flat("Scores by cohort, cumulative", 700, 400, theme.Light, func(p *figure.Plot) {
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear())
				p.Add(geom.ECDF(cohortScores(), geom.X("score"), geom.GroupBy("cohort"),
					geom.ColorBy("cohort", scale.Qualitative(palette.OkabeIto))))
			}, figure.XTitle("score"), figure.YTitle("fraction of cohort"))},
		{ID: "qq", Group: g, Title: "QQ plot",
			Note: "Nine observations against normal quantiles with a reference line; the tail point leaves it.",
			Plot: flat("Normal QQ plot", 640, 400, theme.Light, func(p *figure.Plot) {
				src := figure.NewTable().Float64("value", []float64{8, 9, 9.5, 10, 10.2, 11, 12, 14, 19})
				p.Y(scale.Linear(scale.Domain(6, 20)))
				p.Add(geom.QQ(src, geom.X("value"), geom.Size(7)))
			}, figure.XTitle("Standard normal quantile"), figure.YTitle("Observed value"))},
		{ID: "hexbin", Group: g, Title: "Hexbin + trend",
			Note: "Fifty thousand rows binned into regular hexagons, with a loess trend through them.",
			Plot: flat("Fifty thousand observations", 760, 460, theme.Light, func(p *figure.Plot) {
				src := hexcloud(50000)
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice()))
				p.Add(
					geom.Hexbin(src, geom.X("x"), geom.Y("y"),
						geom.DensityCells(7), geom.Color(palette.Blue)),
					geom.Trend(src, geom.X("x"), geom.Y("y"),
						geom.Span(0.15), geom.Color(palette.Orange), geom.Width(2.5)),
				)
			}, figure.Legend(false))},
		{ID: "bubbles", Group: g, Title: "Bubbles (size channel)",
			Note: "Area-scaled bubbles on a log X axis, with a size key beside the colour legend.",
			Plot: flat("Income and life expectancy", 760, 460, theme.Light, func(p *figure.Plot) {
				p.X(scale.Log(scale.LogNice()))
				p.Y(scale.Linear(scale.Domain(55, 85)))
				p.Add(geom.Scatter(nations(),
					geom.X("income"), geom.Y("years"),
					geom.SizeBy("people", scale.Size()),
					geom.ColorBy("region", scale.Qualitative(palette.OkabeIto)),
					geom.Label("population (millions)")))
			}, figure.XTitle("income per person"), figure.YTitle("years"), figure.Legend(true))},
		{ID: "density", Group: g, Title: "Density (a million points)",
			Note: "A million-point scatter the layer turns into a density raster by itself; zoom in to re-bin.",
			Plot: flat("A million points", 760, 440, theme.Light, func(p *figure.Plot) {
				xs, ys := cloud(1_000_000)
				src := figure.Float64Columns(map[string][]float64{"x": xs, "y": ys})
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice()))
				p.Add(geom.Scatter(src, geom.X("x"), geom.Y("y"), geom.Color(palette.Blue)))
			})},
		{ID: "trend", Group: g, Title: "Trend (loess and linear fit)",
			Note: "A scatter with a loess curve that follows the data and a least-squares line that claims it is linear.",
			Plot: flat("Two fits through one cloud", 760, 440, theme.Light, func(p *figure.Plot) {
				src := trendCloud(300)
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice()))
				p.Add(
					geom.Scatter(src, geom.X("x"), geom.Y("y"),
						geom.Color(palette.SkyBlue), geom.Size(5), geom.Label("observations")),
					geom.Trend(src, geom.X("x"), geom.Y("y"),
						geom.Span(0.3), geom.Color(palette.Orange), geom.Width(2.5), geom.Label("loess")),
					geom.Trend(src, geom.X("x"), geom.Y("y"),
						geom.Smooth(geom.LinearFit), geom.Color(palette.Green),
						geom.Width(2), geom.Dash(6, 4), geom.Label("linear fit")),
				)
			}, figure.Legend(true))},
	}
}

// --- Fields -------------------------------------------------------------------

// The contour charts share one field range, so a colour means the same gain in
// every one of them.
const (
	fieldLo = -12.0
	fieldHi = 6.0
)

func fieldRamp() scale.ColorScale {
	return scale.Sequential(palette.Viridis, scale.ColorDomain(fieldLo, fieldHi))
}

func fieldLevels() []float64 { return stat.Levels(fieldLo, fieldHi, 9) }

func fields() []Entry {
	g := GroupFields
	return []Entry{
		{ID: "contour", Group: g, Title: "Contour",
			Note: "Isolines of a gain field coloured off a pinned ramp; close lines are a cliff, far ones a plain.",
			Plot: flat("Where the gain crosses each level", 640, 460, theme.Light, func(p *figure.Plot) {
				p.X(scale.Linear())
				p.Y(scale.Linear())
				p.Add(geom.Contour(response(),
					geom.X("bias"), geom.Y("drive"), geom.Z("gain"),
					geom.Levels(fieldLevels()...),
					geom.ColorBy("gain", fieldRamp()),
					geom.Width(1.5)))
			}, figure.XTitle("bias (V)"), figure.YTitle("drive (dBm)"))},
		{ID: "contour-heatmap", Group: g, Title: "Contour over heatmap",
			Note: "The same field as gapless cells and white isolines: values from the fill, shape from the lines.",
			Plot: flat("The same field, read both ways", 640, 460, theme.Light, func(p *figure.Plot) {
				src := response()
				p.X(scale.Linear())
				p.Y(scale.Linear())
				p.Add(
					geom.Rect(src, geom.X("bias"), geom.Y("drive"),
						geom.ColorBy("gain", fieldRamp()), geom.BarWidth(1)),
					geom.Contour(src, geom.X("bias"), geom.Y("drive"), geom.Z("gain"),
						geom.Levels(fieldLevels()...),
						geom.Color(palette.White), geom.Width(1)),
				)
			}, figure.XTitle("bias (V)"), figure.YTitle("drive (dBm)"))},
		{ID: "contour-floor", Group: g, Title: "Surface with contour floor",
			Note:  "The field as a 3D surface with the flat chart's isolines on its floor; drag to orbit.",
			Scene: contourFloorScene},
	}
}

// --- Polar --------------------------------------------------------------------

func polar() []Entry {
	g := GroupPolar
	return []Entry{
		{ID: "pie", Group: g, Title: "Pie (ring)",
			Note: "A stacked bar in a polar coord taking theta from Y, with a hole: a ring that closes on the total.",
			Plot: flat("Browser share", 620, 400, bareLayout(theme.Light), func(p *figure.Plot) {
				names, share := browserShare()
				src := figure.NewTable().
					Float64("all", make([]float64, len(names))).
					Float64("share", share).
					String("browser", names)
				p.X(scale.Linear())
				p.Y(scale.Linear())
				p.Add(geom.Bar(src, geom.X("all"), geom.Y("share"),
					geom.GroupBy("browser"),
					geom.ColorBy("browser", scale.Qualitative(palette.OkabeIto))))
			}, figure.Coord(coord.Polar(coord.Theta(coord.FromY), coord.Hole(0.45))))},
		{ID: "donut", Group: g, Title: "Budget donut (explode)",
			Note: "Slices reaching out to each team's budget used, with the over-budget team broken out of the ring.",
			Plot: flat("Spend against each team's budget", 620, 440, bareLayout(theme.Light), func(p *figure.Plot) {
				teams, share, floor, used, pull := budgets()
				src := figure.NewTable().
					String("team", teams).
					Float64("share", share).
					Float64("floor", floor).
					Float64("used", used).
					Float64("pull", pull)
				p.X(scale.Linear(scale.Domain(0, 1)))
				p.Y(scale.Linear())
				p.Add(geom.Bar(src,
					geom.X("floor"), geom.X2("used"), geom.Y("share"),
					geom.GroupBy("team"), geom.ExplodeBy("pull"),
					geom.ColorBy("team", scale.Qualitative(palette.OkabeIto))))
			}, figure.Coord(coord.Pie(coord.Radius(0.95))))},
		{ID: "radar", Group: g, Title: "Radar",
			Note: "Closed, unstacked areas over an ordinal angular axis with chord edges.",
			Plot: flat("Two designs", 620, 440, theme.Dark, func(p *figure.Plot) {
				axisNames, designs, scores := designScores()
				src := figure.NewTable().
					String("axis", axisNames).
					String("design", designs).
					Float64("score", scores)
				p.X(scale.Ordinal(scale.OrdinalPadding(0)))
				p.Y(scale.Linear(scale.Domain(0, 10)))
				p.Add(geom.Area(src, geom.X("axis"), geom.Y("score"),
					geom.GroupBy("design"), geom.Stack(geom.NoStack), geom.Closed(true),
					geom.ColorBy("design", scale.Qualitative(palette.OkabeIto))))
			}, figure.Coord(coord.Polar(coord.Chord())))},
		{ID: "gauge", Group: g, Title: "Gauge",
			Note: "One bar over a half-turn sweep with a hole, and an HLine that becomes a limit needle.",
			Plot: flat("Capacity used", 520, 320, bareLayout(theme.Light), func(p *figure.Plot) {
				src := figure.Float64Columns(map[string][]float64{
					"one":  {0},
					"used": {68},
				})
				p.X(scale.Linear())
				p.Y(scale.Linear(scale.Domain(0, 100)))
				p.Add(
					geom.Bar(src, geom.X("one"), geom.Y("used"), geom.Fill(palette.Vermilion)),
					geom.HLine(90, geom.Label("limit")),
				)
			},
				figure.Legend(false),
				figure.Coord(coord.Polar(
					coord.Theta(coord.FromY),
					coord.Hole(0.6),
					coord.Sweep(math.Pi),
					coord.Start(-math.Pi/2),
				)),
			)},
		{ID: "wind-rose", Group: g, Title: "Wind rose",
			Note: "A bar chart with direction on the angle and hours on the radius: Nightingale's coxcomb.",
			Plot: flat("Hours by wind direction", 560, 520, theme.Light, func(p *figure.Plot) {
				points, hours := wind()
				src := figure.NewTable().
					String("direction", points).
					Float64("hours", hours)
				p.X(scale.Ordinal(scale.OrdinalPadding(0)))
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(geom.Bar(src, geom.X("direction"), geom.Y("hours"),
					geom.Color(palette.Blue), geom.BarWidth(0.9)))
			}, figure.Coord(coord.Polar()))},
	}
}

// --- Smith --------------------------------------------------------------------

// smithAxes gives a plot the grid a paper Smith chart is printed with: pinned
// domains and the conventional tick values.
func smithAxes(p *figure.Plot) {
	p.X(scale.Linear(scale.Domain(0, 50),
		scale.TickValues(0, 0.2, 0.5, 1, 2, 5)))
	p.Y(scale.Linear(scale.Domain(-50, 50),
		scale.TickValues(-5, -2, -1, -0.5, -0.2, 0.2, 0.5, 1, 2, 5)))
}

func smith() []Entry {
	g := GroupSmith
	return []Entry{
		{ID: "smith", Group: g, Title: "Impedance sweep",
			Note: "A patch antenna's S11 converted to normalised impedance, with band edges and the best match marked.",
			Plot: flat("A patch antenna across its band", 620, 560, theme.Light, func(p *figure.Plot) {
				re, im := s11Sweep(121)
				r, x := make([]float64, len(re)), make([]float64, len(re))
				for i := range re {
					r[i], x[i] = coord.SmithZ(re[i], im[i])
				}
				smithAxes(p)
				p.Add(geom.Line(figure.NewTable().Float64("r", r).Float64("x", x),
					geom.X("r"), geom.Y("x"), geom.Color(palette.OkabeIto[1])))
				best := bestMatch(re, im)
				p.Add(geom.Scatter(figure.NewTable().
					Float64("r", []float64{r[0], r[best], r[len(r)-1]}).
					Float64("x", []float64{x[0], x[best], x[len(x)-1]}),
					geom.X("r"), geom.Y("x"), geom.Color(palette.OkabeIto[0])))
			}, figure.Coord(coord.Smith()), figure.Legend(false))},
		{ID: "smith-matching", Group: g, Title: "Matching locus",
			Note: "An L network walking 15 − j25 Ω to the centre, drawn as true arcs rather than chords.",
			Plot: flat("Matching 15 − j25 Ω to 50 Ω", 620, 560, theme.Light, func(p *figure.Plot) {
				series, shunt := matchLocus(9)
				step1 := figure.NewTable().
					Float64("r", realParts(series)).Float64("x", imagParts(series))
				step2 := figure.NewTable().
					Float64("r", realParts(shunt)).Float64("x", imagParts(shunt))
				smithAxes(p)
				p.Add(geom.Line(step1, geom.X("r"), geom.Y("x"), geom.Color(palette.OkabeIto[1])))
				p.Add(geom.Line(step2, geom.X("r"), geom.Y("x"), geom.Color(palette.OkabeIto[2])))
				p.Add(geom.Scatter(figure.NewTable().
					Float64("r", []float64{real(series[0]), real(shunt[len(shunt)-1])}).
					Float64("x", []float64{imag(series[0]), imag(shunt[len(shunt)-1])}),
					geom.X("r"), geom.Y("x"), geom.Color(palette.OkabeIto[0])))
			}, figure.Coord(coord.Smith(coord.SmithArc())), figure.Legend(false))},
		{ID: "smith-admittance", Group: g, Title: "Admittance chart",
			Note: "The same two matching steps on the Y chart, where the shunt step is a plain slide along a conductance circle.",
			Plot: flat("The same two steps, read as admittance", 620, 560, theme.Light, func(p *figure.Plot) {
				series, shunt := matchLocus(60)
				all := append(append([]complex128{}, series...), shunt...)
				for i, z := range all {
					all[i] = 1 / z
				}
				src := figure.NewTable().
					Float64("g", realParts(all)).Float64("b", imagParts(all))
				smithAxes(p)
				p.Add(geom.Line(src, geom.X("g"), geom.Y("b"), geom.Color(palette.OkabeIto[2])))
			}, figure.Coord(coord.Smith(coord.SmithAdmittance(true))), figure.Legend(false))},
		{ID: "smith-vswr", Group: g, Title: "VSWR circles and Q arcs",
			Note: "The antenna sweep read against 1.5, 2 and 3:1 VSWR circles and constant-Q arcs — two loci the coord bends into shape.",
			Plot: flat("The same antenna, read against 2:1", 620, 560, theme.Light, func(p *figure.Plot) {
				re, im := s11Sweep(121)
				r, x := make([]float64, len(re)), make([]float64, len(re))
				for i := range re {
					r[i], x[i] = coord.SmithZ(re[i], im[i])
				}
				smithAxes(p)
				p.Add(geom.Locus(stat.SmithVSWR, []float64{1.5, 2, 3}, geom.Dash(), geom.Label("VSWR")))
				p.Add(geom.Locus(stat.SmithQ, []float64{1, 2, 5}, geom.Dash(2, 3), geom.Label("Q")))
				p.Add(geom.Line(figure.NewTable().Float64("r", r).Float64("x", x),
					geom.X("r"), geom.Y("x"),
					geom.Color(palette.OkabeIto[1]), geom.Width(2), geom.Label("S₁₁")))
			}, figure.Coord(coord.Smith()))},
	}
}

// --- Relational ---------------------------------------------------------------

// layoutPlot is the setup every relational chart shares: two unniced linear
// scales over the unit square the layout runs in.
func layoutPlot(p *figure.Plot) {
	p.X(scale.Linear())
	p.Y(scale.Linear())
}

func relational() []Entry {
	g := GroupRelational
	bare := bareLayout(theme.Light)
	return []Entry{
		{ID: "treemap", Group: g, Title: "Treemap",
			Note: "A directory tree packed into squarified rectangles, each leaf's area its size.",
			Plot: flat("Disk by directory", 700, 420, bare, func(p *figure.Plot) {
				layoutPlot(p)
				p.Add(geom.Treemap(diskUsage(),
					geom.ID("path"), geom.Parent("under"), geom.Value("kb"),
					geom.Padding(0.006),
					geom.ColorBy("group", scale.Qualitative(palette.OkabeIto))))
			})},
		{ID: "icicle", Group: g, Title: "Icicle",
			Note: "The same hierarchy as a span across and a depth down, every level drawn.",
			Plot: flat("Disk by depth", 640, 380, bare, func(p *figure.Plot) {
				layoutPlot(p)
				p.Add(geom.Icicle(diskUsage(),
					geom.ID("path"), geom.Parent("under"), geom.Value("kb"),
					geom.Padding(0.004)))
			}, figure.Legend(false))},
		{ID: "sunburst", Group: g, Title: "Sunburst",
			Note: "The icicle wrapped round a circle by a polar coord: root in the middle, leaves at the rim.",
			Plot: flat("Disk by directory", 520, 440, bare, func(p *figure.Plot) {
				layoutPlot(p)
				p.Add(geom.Icicle(diskUsage(),
					geom.ID("path"), geom.Parent("under"), geom.Value("kb"),
					geom.Padding(0.004)))
			}, figure.Coord(coord.Polar(coord.Hole(0.12))))},
		{ID: "sankey", Group: g, Title: "Sankey",
			Note: "An edge list as a flow: each band as thick as the requests per second it carries.",
			Plot: flat("Requests per second", 700, 400, bare, func(p *figure.Plot) {
				layoutPlot(p)
				p.Add(geom.Sankey(requestFlow(),
					geom.From("from"), geom.To("to"), geom.Value("rps"),
					geom.Padding(0.03)))
			})},
		{ID: "arc", Group: g, Title: "Arc diagram",
			Note: "The same edges as ribbons rising off a rail, arcing as high as they reach.",
			Plot: flat("Service traffic", 620, 400, bare, func(p *figure.Plot) {
				layoutPlot(p)
				p.Add(geom.Arc(requestFlow(),
					geom.From("from"), geom.To("to"), geom.Value("rps"),
					geom.Padding(0.01)))
			})},
		{ID: "chord", Group: g, Title: "Chord diagram",
			Note: "The arc diagram with its rail moved to the rim of a polar coord.",
			Plot: flat("Service traffic", 520, 440, bare, func(p *figure.Plot) {
				layoutPlot(p)
				p.Add(geom.Arc(requestFlow(),
					geom.From("from"), geom.To("to"), geom.Value("rps"),
					geom.Baseline(1), geom.Padding(0.01)))
			}, figure.Coord(coord.Polar()))},
	}
}

// --- Annotations --------------------------------------------------------------

func annotations() []Entry {
	return []Entry{
		{ID: "annotations", Group: GroupAnnotations, Title: "Bands, lines and notes",
			Note: "A tolerance band, a deploy window, an SLO line and a text note around one latency trace.",
			Plot: flat("Latency against its budget", 760, 400, theme.Light, func(p *figure.Plot) {
				xs := ramp(0, 60, 180)
				src := figure.Float64Columns(map[string][]float64{
					"minute": xs,
					"p99": apply(xs, func(x float64) float64 {
						return 190 + 45*math.Sin(x/7) + 12*math.Sin(x/2.3)
					}),
				})
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(
					geom.HBand(170, 210, geom.Label("tolerance")),
					geom.VBand(22, 26, geom.Fill(palette.Orange), geom.Opacity(0.18), geom.Label("deploy")),
					geom.Line(src, geom.X("minute"), geom.Y("p99"),
						geom.Color(palette.Blue), geom.Label("p99")),
					geom.HLine(250, geom.Label("SLO")),
					geom.Note(1, 246, "budget", geom.FontSize(11), geom.Align(ir.AlignStart, ir.AlignTop)),
				)
			})},
		{ID: "nichols", Group: GroupAnnotations, Title: "Nichols diagram",
			Note: "One loop at two gains over the closed-loop M and N contours; the lower gain is tangent to 3 dB. Zoom in and the contours are recomputed, not magnified.",
			Plot: flat("An open loop, and what it does to the closed one", 640, 560, theme.Light, func(p *figure.Plot) {
				p.X(scale.Linear(scale.Domain(-270, -90),
					scale.TickValues(-270, -240, -210, -180, -150, -120, -90)))
				p.Y(scale.Linear(scale.Domain(-24, 36)))
				nicholsGrid(p)
				for i, k := range []float64{nicholsTangent(), 6} {
					phase, gain := loopSweep(k)
					p.Add(geom.Line(
						figure.NewTable().Float64("phase", phase).Float64("gain", gain),
						geom.X("phase"), geom.Y("gain"),
						geom.Color(palette.OkabeIto[1+i]), geom.Width(2),
						geom.Label(fmt.Sprintf("K = %.2f", k))))
				}
			}, figure.XTitle("open-loop phase (degrees)"), figure.YTitle("open-loop gain (dB)"))},
		{ID: "nichols-peak", Group: GroupAnnotations, Title: "Nichols resonance detail",
			Note: "A tenth of the Nichols plane around where the loop touches the 3 dB contour, with the peak computed from the loop.",
			Plot: flat("Where the loop touches 3 dB", 520, 460, theme.Light, func(p *figure.Plot) {
				k := nicholsTangent()
				phase, gain := loopSweep(k)
				at := loopPeak(k)
				p.X(scale.Linear(scale.Domain(-190, -110), scale.TickValues(-180, -160, -140, -120)))
				p.Y(scale.Linear(scale.Domain(-8, 10)))
				nicholsGrid(p)
				p.Add(geom.Line(
					figure.NewTable().Float64("phase", phase).Float64("gain", gain),
					geom.X("phase"), geom.Y("gain"), geom.Color(palette.OkabeIto[1]), geom.Width(2)))
				p.Add(geom.Scatter(
					figure.NewTable().Float64("phase", []float64{at.phase}).Float64("gain", []float64{at.gain}),
					geom.X("phase"), geom.Y("gain"), geom.Color(palette.OkabeIto[0]), geom.Size(9)))
				p.Add(geom.Note(at.phase-2.5, at.gain+1.5,
					fmt.Sprintf("peak %.1f dB at ω = %.2f rad/s", at.peak, at.w),
					geom.Align(ir.AlignEnd, ir.AlignBaseline)))
			}, figure.XTitle("open-loop phase (degrees)"), figure.YTitle("open-loop gain (dB)"), figure.Legend(false))},
	}
}

// nicholsGrid adds the closed-loop magnitude (solid) and phase (dashed)
// contours under whatever the panel holds, at the levels a chart is printed at.
func nicholsGrid(p *figure.Plot) {
	p.Add(geom.Locus(stat.NicholsM, []float64{-12, -6, -3, -1, 0, 1, 3, 6, 12},
		geom.Dash(), geom.Label("closed-loop gain")))
	p.Add(geom.Locus(stat.NicholsN, []float64{-1, -5, -10, -20, -45, -90, -150, -210, -270},
		geom.Dash(2, 3), geom.Label("closed-loop phase")))
}

// --- Layout -------------------------------------------------------------------

func layout() []Entry {
	g := GroupLayout
	return []Entry{
		{ID: "facets", Group: g, Title: "Facets",
			Note: "One layer wrapped into a panel per region, sharing axes and a target line.",
			Plot: flat("Throughput by region", 800, 460, theme.Light, func(p *figure.Plot) {
				regions, hours, rps := fleet()
				src := figure.NewTable().
					String("region", regions).
					Float64("hour", hours).
					Float64("rps", rps)
				p.X(scale.Linear(scale.Nice()))
				p.Y(scale.Linear(scale.Nice(), scale.Zero()))
				p.Add(geom.Line(src, geom.X("hour"), geom.Y("rps"),
					geom.Color(palette.Blue), geom.Label("throughput")))
				p.Add(geom.HLine(60, geom.Label("target")))
				p.Facet(facet.Wrap("region", facet.Columns(3)))
			})},
		{ID: "subplots", Group: g, Title: "Subplots (grid)",
			Note: "Four independent plots laid out in a two-column grid, rendered as a static image.",
			Grid: subplots},
	}
}

func subplots() *figure.Grid {
	g := figure.NewGrid(2,
		figure.GridTheme(theme.Dark),
		figure.GridSize(800, 480),
		figure.GridTitle("Fleet overview"),
	)
	xs := ramp(0, 12, 120)
	add := func(title string, fn func(float64) float64, c int) {
		p := figure.New(figure.Title(title))
		p.X(scale.Linear(scale.Nice()))
		p.Y(scale.Linear(scale.Nice()))
		p.Add(geom.Line(figure.Float64Columns(map[string][]float64{
			"x": xs, "y": apply(xs, fn),
		}), geom.X("x"), geom.Y("y"), geom.Color(palette.OkabeIto.At(c))))
		g.Add(p)
	}
	add("latency", func(x float64) float64 { return 50 + 20*math.Sin(x) }, 0)
	add("throughput", func(x float64) float64 { return 900 * math.Exp(-x/9) }, 1)
	add("errors", func(x float64) float64 { return 3 * math.Sin(x/2) * math.Sin(x/2) }, 2)
	add("saturation", func(x float64) float64 { return 0.45 + 0.3*math.Sin(x/3) }, 3)
	return g
}
