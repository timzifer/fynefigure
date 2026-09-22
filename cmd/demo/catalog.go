package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/timzifer/fynefigure/cmd/demo/plots"
)

// entry is one leaf of the tree: a chart, and how to build it on stage.
type entry struct {
	id, group, title, note string
	build                  func(env) (*view, error)
}

// catalogue is the tree: groups in order, and the entries under each.
type catalogue struct {
	entries  []*entry
	byID     map[string]*entry
	children map[string][]string // tree node → child nodes; "" is the root
	groups   map[string]string   // group node → group name
}

// groupPrefix marks a tree node as a group rather than an entry, so that the
// two can never be confused whatever an entry is called.
const groupPrefix = "group:"

// newNode is the tree's first group: the charts added last, which is the only
// group open when the demo starts. Its children repeat entries listed under
// their own group, and a tree node has one parent, so they are prefixed.
const (
	newNode   = groupPrefix + "New"
	newPrefix = "new:"
)

// newest is how many of plots.Newest the New group shows.
const newest = 8

// catalog is every chart figure draws — package plots — and the ones that
// need more than one widget or a clock — showcase.go — in the tree's order.
func catalog() *catalogue {
	var all []*entry
	for _, pe := range plots.All() {
		all = append(all, fromPlots(pe))
	}
	all = append(all, showcase()...)

	c := &catalogue{
		byID:     map[string]*entry{},
		children: map[string][]string{},
		groups:   map[string]string{},
	}
	for _, g := range append(plots.Groups(), groupInteraction) {
		node := groupPrefix + g
		c.groups[node] = g
		for _, e := range all {
			if e.group != g {
				continue
			}
			if _, dup := c.byID[e.id]; dup {
				log.Printf("catalogue: %q is listed twice; showing the first", e.id)
				continue
			}
			c.byID[e.id] = e
			c.entries = append(c.entries, e)
			c.children[node] = append(c.children[node], e.id)
		}
		if len(c.children[node]) > 0 {
			c.children[""] = append(c.children[""], node)
		}
	}
	var fresh []string
	for _, id := range plots.Newest() {
		if len(fresh) == newest {
			break
		}
		if c.byID[id] == nil {
			log.Printf("catalogue: %q is listed as new but is not in the tree", id)
			continue
		}
		fresh = append(fresh, newPrefix+id)
	}
	if len(fresh) > 0 {
		c.groups[newNode] = "New"
		c.children[newNode] = fresh
		c.children[""] = append([]string{newNode}, c.children[""]...)
	}
	for _, e := range all {
		if c.byID[e.id] == nil {
			log.Printf("catalogue: %q is in group %q, which is not in the tree", e.id, e.group)
		}
	}
	return c
}

// fromPlots makes a package plots entry buildable: a flat chart goes in a
// chart widget, a scene in an orbit widget and a grid in a picture.
func fromPlots(pe plots.Entry) *entry {
	e := &entry{id: pe.ID, group: pe.Group, title: pe.Title, note: pe.Note}
	switch {
	case pe.Plot != nil:
		e.build = func(en env) (*view, error) { return flatView(en, pe.Plot()), nil }
	case pe.Scene != nil:
		e.build = func(en env) (*view, error) { return orbitView(en, pe.Scene()), nil }
	case pe.Grid != nil:
		e.build = func(env) (*view, error) { return gridView(pe.Grid()), nil }
	default:
		e.build = func(env) (*view, error) { return nil, errors.New("the entry has nothing to build") }
	}
	return e
}

// find is the entry with the given id, or the newest one.
func (c *catalogue) find(id string) *entry {
	if e, ok := c.entry(id); ok {
		return e
	}
	if fresh := c.children[newNode]; len(fresh) > 0 {
		return c.byID[strings.TrimPrefix(fresh[0], newPrefix)]
	}
	return c.entries[0]
}

// entry is the entry a tree node shows, whether it is listed under its own
// group or under New.
func (c *catalogue) entry(node string) (*entry, bool) {
	e, ok := c.byID[strings.TrimPrefix(node, newPrefix)]
	return e, ok
}

// node is where the tree shows an entry: under New when it is new, since that
// group is open, and under its own group otherwise.
func (c *catalogue) node(e *entry) string {
	for _, n := range c.children[newNode] {
		if n == newPrefix+e.id {
			return n
		}
	}
	return e.id
}

func (c *catalogue) isBranch(id string) bool {
	return id == "" || strings.HasPrefix(id, groupPrefix)
}

func (c *catalogue) label(id string) string {
	if g, ok := c.groups[id]; ok {
		return fmt.Sprintf("%s  (%d)", g, len(c.children[id]))
	}
	if e, ok := c.entry(id); ok {
		return e.title
	}
	return id
}
