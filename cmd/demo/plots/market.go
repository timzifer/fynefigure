package plots

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/timzifer/figure"
	"github.com/timzifer/figure/data"
	"github.com/timzifer/figure/geom"
	"github.com/timzifer/figure/ir"
	"github.com/timzifer/figure/palette"
	"github.com/timzifer/figure/scale"
	"github.com/timzifer/figure/stat"
	"github.com/timzifer/figure/theme"
)

// --- Market -----------------------------------------------------------------
//
// The charts a trading screen draws, from one simulated market: the daily chart
// with everything on it, and the same prices as OHLC bars and as hollow
// candles. They are ports of the gallery's market.go, which is figure's copy of
// examples/market; the live chart is in the demo's showcase, because it needs
// a clock.

func market() []Entry {
	g := GroupMarket
	return []Entry{
		{ID: "market", Group: g, Title: "Daily candles",
			Note: "Candles on a trading calendar with weekends and a holiday folded out, a Bollinger band, two averages and their crossings, volume and RSI tracks under the price, the last close tagged, and volume at price beside it.",
			Plot: flat("FIG — daily, weekends and holidays folded", 1000, 660, theme.Light, func(p *figure.Plot) {
				m := simulateMarket()
				days := m.days()
				candles := dailyTable(days)
				p.X(foldedTime(marketStart))
				p.Y(scale.Linear(scale.Nice()))

				p.Add(geom.Area(candles, geom.X("t"), geom.Y("lower"), geom.Y2("upper"),
					geom.Color(palette.Gray), geom.Opacity(0.18), geom.Label("Bollinger 20, 2σ")))
				p.Add(geom.Candle(candles, geom.X("t"), geom.OHLC("open", "high", "low", "close"),
					geom.Rising(MarketUp, "up"), geom.Falling(MarketDown, "down")))
				p.Add(geom.Line(candles, geom.X("t"), geom.Y("ema"), geom.Color(palette.Orange), geom.Label("EMA 9")))
				p.Add(geom.Line(candles, geom.X("t"), geom.Y("mid"), geom.Color(palette.Purple), geom.Label("SMA 20")))

				buys, sells := crossings(days)
				p.Add(geom.Scatter(buys, geom.X("t"), geom.Y("at"),
					geom.Shape(ir.MarkerTriangle), geom.Size(10), geom.Color(MarketUp), geom.Label("buy")))
				p.Add(geom.Scatter(sells, geom.X("t"), geom.Y("at"),
					geom.Shape(ir.MarkerTriangleDown), geom.Size(10), geom.Color(MarketDown), geom.Label("sell")))

				p.Track(figure.Bottom, figure.TrackFraction(0.2), figure.TrackScale(scale.Linear(scale.Zero())), figure.TrackAxis(true)).
					Add(geom.Bar(candles, geom.X("t"), geom.Y("volume"), geom.BarWidth(0.7),
						geom.DirectionBy("open", "close"), geom.Rising(MarketUp, "up"), geom.Falling(MarketDown, "down"),
						geom.Opacity(0.6)))
				p.Track(figure.Bottom, figure.TrackFraction(0.16), figure.TrackScale(scale.Linear(scale.Domain(0, 100))), figure.TrackAxis(true)).
					Add(geom.HBand(30, 70, geom.Label("RSI 30–70")),
						geom.Line(candles, geom.X("t"), geom.Y("rsi"), geom.Color(palette.Purple), geom.Label("RSI 14")))

				p.Add(geom.LastValue(candles, geom.X("t"), geom.Y("close"), geom.Dash(3, 3),
					geom.DirectionBy("open", "close"), geom.Rising(MarketUp, "up"), geom.Falling(MarketDown, "down")))

				p.Track(figure.Right, figure.TrackSize(140), figure.TrackScale(scale.Linear(scale.Zero())), figure.TrackAxis(true)).
					Add(geom.Bar(volumeAtPrice(m), geom.Y("price"), geom.X("volume"),
						geom.Orient(geom.Horizontal), geom.BarWidth(1), geom.GroupBy("side"),
						geom.ColorBy("side", scale.Named(map[string]ir.Color{"buy": MarketUp, "sell": MarketDown})),
						geom.Opacity(0.7)))
			})},
		{ID: "market-bars", Group: g, Title: "OHLC bars",
			Note: "The last thirty days as OHLC bars — the open ticked left, the close right — coloured by the close against the previous close rather than the day's own open.",
			Plot: flat("geom.CandleStyle(geom.Ticks), since the previous close", 620, 400, theme.Light, func(p *figure.Plot) {
				candles, from := lastWeeks(30)
				p.X(foldedTime(from))
				p.Y(scale.Linear(scale.Nice()))
				p.Add(geom.Candle(candles, geom.X("t"), geom.OHLC("open", "high", "low", "close"),
					geom.CandleStyle(geom.Ticks), geom.Direction(geom.SincePrevious),
					geom.Rising(MarketUp, "up"), geom.Falling(MarketDown, "down")))
			}, figure.Legend(false))},
		{ID: "market-hollow", Group: g, Title: "Hollow candles",
			Note: "The same thirty days under redundant encoding, which draws the rising bodies as outlines: the print convention, readable without the colour.",
			Plot: flat("Hollow candles under theme.Redundant(true)", 620, 400, theme.Light.With(theme.Redundant(true)), func(p *figure.Plot) {
				candles, from := lastWeeks(30)
				p.X(foldedTime(from))
				p.Y(scale.Linear(scale.Nice()))
				p.Add(geom.Candle(candles, geom.X("t"), geom.OHLC("open", "high", "low", "close"),
					geom.Rising(MarketUp, "up"), geom.Falling(MarketDown, "down")))
			}, figure.Legend(false))},
	}
}

// MarketUp and MarketDown are rising and falling: blue and vermilion rather
// than green and red, the pair the most common colour deficiency cannot tell
// apart. They are exported for the live market chart in the showcase.
var (
	MarketUp   = palette.Blue
	MarketDown = palette.Vermilion
)

// marketStart is the chart's first calendar day, a Monday, and marketDays how
// many calendar days it runs: twelve weeks.
var marketStart = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)

const marketDays = 7 * 12

var marketEnd = marketStart.AddDate(0, 0, marketDays)

// marketCalendar is Monday to Friday without the Friday Independence Day is
// observed on in 2026.
var marketCalendar = scale.Closed(
	scale.Workweek(time.UTC, scale.Weekdays(time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday)),
	scale.Dates(time.Date(2026, time.July, 3, 0, 0, 0, 0, time.UTC)))

// ticks is the simulated trading: a seeded random walk between 09:30 and 16:00
// on the days the calendar opens, with each tick's volume split into what was
// bought and what was sold.
type ticks struct {
	ts, ps, vs  []float64
	buys, sells []float64
}

func simulateMarket() ticks {
	r := rand.New(rand.NewPCG(7, 85))
	var m ticks
	price := 100.0
	for d := range marketDays {
		day := marketStart.AddDate(0, 0, d)
		if len(marketCalendar.Open(nil, day)) == 0 {
			continue
		}
		// A drift that changes sign every few weeks, so the averages cross.
		drift := 0.04 * math.Sin(float64(d)/9)
		for tick := range 40 {
			at := day.Add(9*time.Hour + 30*time.Minute + time.Duration(tick)*585*time.Second)
			price *= 1 + drift/40 + r.NormFloat64()*0.004
			v := 100 + r.Float64()*900
			bought := min(max(v*(0.5+drift*4+r.NormFloat64()*0.1), 0), v)
			m.ts = append(m.ts, scale.Nanos(at))
			m.ps = append(m.ps, price)
			m.vs = append(m.vs, v)
			m.buys = append(m.buys, bought)
			m.sells = append(m.sells, v-bought)
		}
	}
	return m
}

func (m ticks) days() []stat.Candle {
	return stat.OHLC(m.ts, m.ps, m.vs, scale.Nanos(marketStart), float64(24*time.Hour))
}

// dailyTable is the day candles with the indicators beside them. A candle sits
// at the middle of its day so that its slot is the day rather than straddling
// midnight into a fold, and the indicators run over the row rather than the
// time, because a weekend is not two days of average.
func dailyTable(days []stat.Candle) *data.Table {
	n := len(days)
	t := make([]time.Time, n)
	xs := make([]float64, n)
	o, h, l, c, v := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	for i, d := range days {
		t[i] = scale.FromNanos(d.Start + float64(12*time.Hour))
		xs[i] = float64(i)
		o[i], h[i], l[i], c[i], v[i] = d.Open, d.High, d.Low, d.Close, d.Volume
	}
	ema := pointYs(stat.EMA(xs, c, 9))
	mid := pointYs(stat.TrailingMean(xs, c, 20))
	sd := pointYs(stat.RollingStdDev(xs, c, 20))
	rsi := pointYs(stat.RSI(xs, c, 14))
	lower, upper := make([]float64, n), make([]float64, n)
	for i := range n {
		lower[i], upper[i] = mid[i]-2*sd[i], mid[i]+2*sd[i]
	}
	return figure.NewTable().
		Time("t", t).
		Float64("open", o).Float64("high", h).Float64("low", l).Float64("close", c).
		Float64("volume", v).
		Float64("ema", ema).Float64("mid", mid).Float64("rsi", rsi).
		Float64("lower", lower).Float64("upper", upper)
}

// crossings is where the EMA crosses the simple mean, after the mean's first
// window: a buy under the day's low, a sell over its high.
func crossings(days []stat.Candle) (buys, sells *data.Table) {
	xs := make([]float64, len(days))
	closes := make([]float64, len(days))
	for i, d := range days {
		xs[i], closes[i] = float64(i), d.Close
	}
	ema := pointYs(stat.EMA(xs, closes, 9))
	mid := pointYs(stat.TrailingMean(xs, closes, 20))
	var bt, st []time.Time
	var ba, sa []float64
	for i := 20; i < len(days); i++ {
		at := scale.FromNanos(days[i].Start + float64(12*time.Hour))
		pad := (days[i].High - days[i].Low) * 0.4
		switch {
		case ema[i-1] <= mid[i-1] && ema[i] > mid[i]:
			bt, ba = append(bt, at), append(ba, days[i].Low-pad)
		case ema[i-1] >= mid[i-1] && ema[i] < mid[i]:
			st, sa = append(st, at), append(sa, days[i].High+pad)
		}
	}
	return figure.NewTable().Time("t", bt).Float64("at", ba),
		figure.NewTable().Time("t", st).Float64("at", sa)
}

// volumeAtPrice is the volume bought and sold in each price bucket, one row per
// bucket and side, binned over one interval so the two sides line up.
func volumeAtPrice(m ticks) *data.Table {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range m.ps {
		lo, hi = min(lo, p), max(hi, p)
	}
	b := stat.BinWeighted(m.ps, m.buys, lo, hi, 28)
	s := stat.BinWeighted(m.ps, m.sells, lo, hi, 28)
	var price, volume []float64
	var side []string
	for i := range b {
		price = append(price, b[i].Mid(), s[i].Mid())
		volume = append(volume, b[i].Sum, s[i].Sum)
		side = append(side, "buy", "sell")
	}
	return figure.NewTable().Float64("price", price).Float64("volume", volume).String("side", side)
}

func pointYs(ps []stat.Point) []float64 {
	out := make([]float64, len(ps))
	for i, p := range ps {
		out[i] = p.Y
	}
	return out
}

// foldedTime is a time axis on the market's calendar with the closed days
// between from and marketEnd folded out.
func foldedTime(from time.Time) scale.Scale {
	closed, err := scale.Folds(marketCalendar, from, marketEnd)
	if err != nil {
		panic(err)
	}
	return scale.Time(scale.TimeCalendar(marketCalendar), scale.TimeFold(closed...))
}

// lastWeeks is the daily table cut to the last n trading days, and the first
// of them: close enough for a candle's shape to read.
func lastWeeks(n int) (*data.Table, time.Time) {
	days := simulateMarket().days()
	days = days[len(days)-n:]
	return dailyTable(days), scale.FromNanos(days[0].Start)
}
