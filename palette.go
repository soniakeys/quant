// Copyright 2013 Sonia Keys.
// Licensed under MIT license.  See "license" file in this source tree.

// Quant provides an interface for image color quantizers.
package quant

import "image/color"

// Palette is a palette of color.Colors, much like color.Pallete of the
// standard library.
//
// It is defined as an interface here to allow more general implementations,
// presumably ones that maintain some data structure to achieve performance
// advantages over linear search.
type Palette interface {
	IndexNear(color.Color) int
	ColorNear(color.Color) color.Color
	ColorPalette() color.Palette
}

var _ Palette = LinearPalette{}
var _ Palette = &TreePalette{}

// LinearPalette implements the Palette interface with color.Palette
// and has no optimizations.
type LinearPalette struct {
	// Convert method of Palette satisfied by method of color.Palette.
	color.Palette
}

func (p LinearPalette) IndexNear(c color.Color) int {
	return p.Palette.Index(c)
}

func (p LinearPalette) ColorNear(c color.Color) color.Color {
	return p.Palette.Convert(c)
}

// ColorPalette satisfies interface Palette.
//
// It simply returns the internal color.Palette.
func (p LinearPalette) ColorPalette() color.Palette {
	return p.Palette
}

type TreePalette struct {
	Type int
	// for TLeaf
	Index int
	Color color.RGBA64
	// for TSplit
	Split     uint32
	Low, High *TreePalette
}

const (
	TLeaf = iota
	TSplitR
	TSplitG
	TSplitB
)

func (t *TreePalette) IndexNear(c color.Color) (i int) {
	if t == nil {
		return -1
	}
	t.search(c, func(leaf *TreePalette) { i = leaf.Index })
	return
}

func (t *TreePalette) ColorNear(c color.Color) (p color.Color) {
	if t == nil {
		return color.RGBA64{0x7fff, 0x7fff, 0x7fff, 0xfff}
	}
	t.search(c, func(leaf *TreePalette) { p = leaf.Color })
	return
}

func (t *TreePalette) search(c color.Color, f func(leaf *TreePalette)) {
	r, g, b, _ := c.RGBA()
	var lt bool
	var s func(*TreePalette)
	s = func(t *TreePalette) {
		switch t.Type {
		case TLeaf:
			f(t)
			return
		case TSplitR:
			lt = r < t.Split
		case TSplitG:
			lt = g < t.Split
		case TSplitB:
			lt = b < t.Split
		}
		if lt {
			s(t.Low)
		} else {
			s(t.High)
		}
	}
	s(t)
	return
}

func (t *TreePalette) ColorPalette() (p color.Palette) {
	if t == nil {
		return
	}
	var walk func(*TreePalette)
	walk = func(t *TreePalette) {
		if t.Type == TLeaf {
			p = append(p, color.Color(t.Color))
			return
		}
		walk(t.Low)
		walk(t.High)
	}
	walk(t)
	return
}
