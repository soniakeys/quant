// Copyright 2013 Sonia Keys.
// Licensed under MIT license.  See "license" file in this source tree.

// Quant provides an interface for image color quantizers.
package quant

import (
	"image/color"
)

// Palette is a palette of color.Colors, just as color.Pallete of the standard
// library.
//
// It is defined as an interface here to allow more general implementations
// of Index, presumably ones that maintain some data structure to achieve
// performance advantages over linear search.
type Palette interface {
	Convert(color.Color) color.Color
	ColorPalette() color.Palette
}

// LinearPalette implements the Palette interface with color.Palette
// and has no optimizations.
type LinearPalette struct {
	// Convert method of Palette satisfied by method of color.Palette.
	color.Palette
}

// ColorPalette satisfies interface Palette.
//
// It simply returns the internal color.Palette.
func (p LinearPalette) ColorPalette() color.Palette {
	return p.Palette
}

type TreePalette struct {
	Type  int
	Color color.RGBA64 // for TLeaf
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

func (t *TreePalette) Convert(c color.Color) (p color.Color) {
	if t == nil {
		return color.RGBA64{0x7fff, 0x7fff, 0x7fff, 0xfff}
	}
	r, g, b, _ := c.RGBA()
	var lt bool
	var search func(*TreePalette)
	search = func(t *TreePalette) {
		switch t.Type {
		case TLeaf:
			p = t.Color
			return
		case TSplitR:
			lt = r < t.Split
		case TSplitG:
			lt = g < t.Split
		case TSplitB:
			lt = b < t.Split
		}
		if lt {
			search(t.Low)
		} else {
			search(t.High)
		}
	}
	search(t)
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
