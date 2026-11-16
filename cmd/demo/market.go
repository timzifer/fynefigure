package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/timzifer/figure"
	"github.com/timzifer/figure/data"
	"github.com/timzifer/figure/geom"
	"github.com/timzifer/figure/scale"
	"github.com/timzifer/figure/stat"
	"github.com/timzifer/fynefigure/chart"
	"github.com/timzifer/fynefigure/cmd/demo/plots"
)

// marketLiveShow is the gallery's market-live plate played for real: a tick
// feed folded into one-minute candles as it arrives, the candle whose minute
// has not closed revised in place, the price axis fitted to the candles in
// view, and a crosshair snapped to the candle under the pointer with its time
// and price tagged on the axes. See figure's ADR 0089 and ADR 0092.
//
// The feed runs at five simulated seconds a tick and a tick every 30ms, so a
// candle closes about three times a second and the window turns over in under
// a minute.
func marketLiveShow(e env) (*view, error) {
	const (
		window = 120                   // candles kept
		every  = 5 * time.Second       // simulated time between two ticks
		pace   = 30 * time.Millisecond // wall time between two ticks
		redraw = 50 * time.Millisecond
	)

	st := data.NewStream("start", "end", "open", "high", "low", "close", "volume").Window(window)
	feed := &tickFeed{
		r:     stat.Resampler{Origin: scale.Nanos(marketOpen), Width: float64(time.Minute)},
		rng:   rand.New(rand.NewPCG(3, 89)),
		at:    marketOpen,
		every: every,
		price: 100,
		st:    st,
	}
	// Seed the window, so the chart opens on a full screen of candles rather
	// than on one.
	for range window * int(time.Minute/every) {
		if err := feed.tick(); err != nil {
			return nil, fmt.Errorf("seeding the feed: %w", err)
		}
	}

	p := figure.New(
		figure.Responsive(true),
		figure.Size(900, 480),
		figure.Title("FIG — live, one-minute candles"),
		figure.Legend(false),
	)
	// The price axis fits the candles in view rather than every candle the
	// window holds, and a gesture leaves it alone: it is derived from the time
	// axis every frame. The same goes for the volume track.
	p.X(scale.Time()).Y(scale.Linear(scale.Nice(), scale.FitView()))
	p.Add(geom.Candle(st.Source(), geom.X("start"), geom.X2("end"),
		geom.OHLC("open", "high", "low", "close"),
		geom.Rising(plots.MarketUp, "up"), geom.Falling(plots.MarketDown, "down")))
	p.Track(figure.Bottom, figure.TrackFraction(0.2),
		figure.TrackScale(scale.Linear(scale.Zero(), scale.FitView())), figure.TrackAxis(true)).
		Add(geom.Bar(st.Source(), geom.X("start"), geom.Y("volume"), geom.BarWidth(0.7), geom.Opacity(0.6),
			geom.DirectionBy("open", "close"), geom.Rising(plots.MarketUp, "up"), geom.Falling(plots.MarketDown, "down")))
	p.Add(geom.LastValue(st.Source(), geom.X("start"), geom.Y("close"), geom.Dash(3, 3),
		geom.DirectionBy("open", "close"), geom.Rising(plots.MarketUp, "up"), geom.Falling(plots.MarketDown, "down")))

	// The crosshair snaps to the candle nearest the pointer in time, which
	// needs the chart's row index; the chart has one from its first layout, so
	// it is taken on the first hover rather than here. Live is a plain read of
	// the chart's field, which is safe from inside the chart's own handler.
	cross := &figure.Crosshair{Panel: -1, Tags: true}

	// The x axis follows the window — chart.Follow, the default — so a drag
	// has nothing behind the tip to pan to and is ignored.
	c := chart.New(p, e.chartOpts(chart.Overlay(cross))...)
	c.Stream(st)
	p.On(figure.Hover, func(ev figure.Event) {
		if cross.Snap == nil {
			if l := c.Live(); l != nil {
				cross.Snap = l.Index()
			}
		}
		cross.At, cross.Show = ev.Point, ev.Panel >= 0
		if !ev.Found {
			e.status("")
			return
		}
		// The x of a mark here is a time in nanoseconds, which the generic
		// readout would print as a number.
		e.status(fmt.Sprintf("%s   %s   %.2f   row %d", ev.Series(),
			scale.FromNanos(ev.Hit.X).Format("Jan 2 15:04"), ev.Hit.Y, ev.Hit.Row))
	})
	p.On(figure.Leave, func(figure.Event) { cross.Show = false; e.status("") })

	start := func() func() {
		done, finished := make(chan struct{}), make(chan struct{})
		// The producer is the feed's only user while it runs, and stop waits
		// for it to return, so the next time the view goes on stage starts a
		// producer that has the feed to itself again.
		go func() {
			defer close(finished)
			t := time.NewTicker(pace)
			defer t.Stop()
			for {
				select {
				case <-done:
					return
				case <-t.C:
					if err := feed.tick(); err != nil {
						log.Println("market feed:", err)
						return
					}
				}
			}
		}()
		stopDrawing := c.Animate(redraw)
		return func() {
			stopDrawing()
			close(done)
			<-finished
		}
	}
	return &view{obj: c, flats: []*chart.Chart{c}, start: start}, nil
}

// marketOpen is when the live feed starts.
var marketOpen = time.Date(2026, time.September, 28, 9, 30, 0, 0, time.UTC)

// tickFeed is a seeded random walk folded into candles: each tick either opens
// a candle, when its minute is a new one, or revises the candle still open.
// Only one goroutine calls tick at a time.
type tickFeed struct {
	r     stat.Resampler
	rng   *rand.Rand
	at    time.Time
	every time.Duration
	price float64
	n     int
	st    *data.Stream
	row   [7]float64
}

func (f *tickFeed) tick() error {
	f.n++
	f.at = f.at.Add(f.every)
	f.price *= 1 + 0.0004*math.Sin(float64(f.n)/120) + f.rng.NormFloat64()*0.0016
	cd, fresh, err := f.r.Add(scale.Nanos(f.at), f.price, 1+f.rng.Float64()*9)
	if err != nil {
		return err
	}
	f.row = [7]float64{cd.Start, cd.End, cd.Open, cd.High, cd.Low, cd.Close, cd.Volume}
	if fresh {
		return f.st.Append(f.row[:]...)
	}
	// The newest row is the one candle still being measured, and the one
	// revision a stream admits.
	return f.st.ReplaceLast(f.row[:]...)
}
