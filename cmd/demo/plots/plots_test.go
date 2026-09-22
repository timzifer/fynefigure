package plots

import (
	"bytes"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/timzifer/figure"
)

// heavy are the entries whose data alone runs to millions of rows. They are
// still rendered, unless -short is given.
var heavy = map[string]bool{"decimation": true, "density": true}

var kebab = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func TestCatalogueShape(t *testing.T) {
	groups := Groups()
	seen := map[string]bool{}
	lastGroup := -1
	for _, e := range All() {
		if !kebab.MatchString(e.ID) {
			t.Errorf("%q: id is not kebab-case", e.ID)
		}
		if seen[e.ID] {
			t.Errorf("%q: duplicate id", e.ID)
		}
		seen[e.ID] = true
		gi := slices.Index(groups, e.Group)
		if gi < 0 {
			t.Errorf("%q: group %q is not in Groups()", e.ID, e.Group)
		}
		if gi < lastGroup {
			t.Errorf("%q: group %q out of tree order", e.ID, e.Group)
		}
		lastGroup = max(lastGroup, gi)
		if e.Title == "" || e.Note == "" {
			t.Errorf("%q: missing title or note", e.ID)
		}
		n := 0
		for _, set := range []bool{e.Plot != nil, e.Scene != nil, e.Grid != nil} {
			if set {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%q: %d of Plot/Scene/Grid set, want exactly 1", e.ID, n)
		}
	}
}

func TestEveryEntryRenders(t *testing.T) {
	for _, e := range All() {
		t.Run(e.ID, func(t *testing.T) {
			if heavy[e.ID] && testing.Short() {
				t.Skip("millions of rows; skipped under -short")
			}
			var c interface{ Render(figure.Target) error }
			switch {
			case e.Plot != nil:
				c = e.Plot()
			case e.Scene != nil:
				c = e.Scene()
			case e.Grid != nil:
				c = e.Grid()
			}
			start := time.Now()
			var buf bytes.Buffer
			if err := c.Render(figure.SVGWriter(&buf)); err != nil {
				t.Fatalf("render: %v", err)
			}
			if buf.Len() == 0 {
				t.Fatal("empty SVG")
			}
			if !bytes.Contains(buf.Bytes(), []byte("<svg")) {
				t.Fatal("output is not an SVG document")
			}
			t.Logf("%d bytes in %v", buf.Len(), time.Since(start).Round(time.Millisecond))
		})
	}
}

// TestBuildersAreFresh checks that two calls return two plots, since the app
// rebuilds a chart when it switches renderer.
func TestBuildersAreFresh(t *testing.T) {
	for _, e := range All() {
		if heavy[e.ID] {
			continue
		}
		switch {
		case e.Plot != nil:
			if e.Plot() == e.Plot() {
				t.Errorf("%q: Plot returned the same plot twice", e.ID)
			}
		case e.Scene != nil:
			if e.Scene() == e.Scene() {
				t.Errorf("%q: Scene returned the same plot twice", e.ID)
			}
		case e.Grid != nil:
			if e.Grid() == e.Grid() {
				t.Errorf("%q: Grid returned the same grid twice", e.ID)
			}
		}
	}
}

// Every id Newest names is an entry, once.
func TestNewestNamesEntries(t *testing.T) {
	ids := map[string]bool{}
	for _, e := range All() {
		ids[e.ID] = true
	}
	seen := map[string]bool{}
	for _, id := range Newest() {
		if !ids[id] {
			t.Errorf("%q: in Newest but not in All", id)
		}
		if seen[id] {
			t.Errorf("%q: in Newest twice", id)
		}
		seen[id] = true
	}
}
