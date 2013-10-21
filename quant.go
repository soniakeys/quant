// Copyright 2013 Sonia Keys.
// Licensed under MIT license.  See "license" file in this source tree.

// Quant provides an interface for image color quantizers.
package quant

import (
	"image"
	"image/color"
	"image/draw"
)

// Quantizer defines a color quantizer for images.
type Quantizer interface {
	// Image quantizes an image and returns a paletted image
	Image(image.Image) *image.Paletted
	// Palette quantizes an image and returns a Palette.  Note the type is
	// the Palette of this package and not image.Palette.
	Palette(image.Image) Palette
}

// Palette is a palette of color.Colors, just as color.Pallete of the standard
// library.
//
// It is defined as an interface here to allow more general implementations
// of Index, presumably ones that maintain some data structure to achieve
// performance advantages over linear search.
type Palette interface {
	Convert(color.Color) color.Color
	Index(color.Color) int
	ColorPalette() color.Palette
}

// LinearPalette implements the Palette interface with color.Palette
// and has no optimizations.
type LinearPalette struct {
	color.Palette
}

func (p LinearPalette) ColorPalette() color.Palette {
	return p.Palette
}

type Dither211 struct{}

// Dither211 satisfies draw.Drawer
var _ draw.Drawer = Dither211{}

func (d Dither211) Draw(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point) {
	pd, ok := dst.(*image.Paletted)
	if !ok {
		// dither211 currently requires a palette
		draw.Draw(dst, r, src, sp, draw.Src)
		return
	}
	// intersect r with both dst and src bounds, fix up sp.
	ir := r.Intersect(pd.Bounds()).
		Intersect(src.Bounds().Add(r.Min.Sub(sp)))
	if ir.Empty() {
		return // no work to do.
	}
	sp = ir.Min.Sub(r.Min)
	// get subimage of src
	sr := ir.Add(sp)
	if !sr.Eq(src.Bounds()) {
		s, ok := src.(interface {
			SubImage(image.Rectangle) image.Image
		})
		if !ok {
			// dither211 currently works on whole images
			draw.Draw(dst, r, src, sp, draw.Src)
			return
		}
		src = s.SubImage(sr)
	}
	// dither211 currently returns a new image
	src = dither211(src, LinearPalette{pd.Palette})
	draw.Draw(dst, r, src, image.Point{}, draw.Src)
}

func dither211(i0 image.Image, p Palette) *image.Paletted {
	cp := p.ColorPalette()
	if len(cp) > 256 {
		return nil
	}
	b := i0.Bounds()
	pi := image.NewPaletted(b, cp)
	if b.Max.Y-b.Min.Y == 0 || b.Max.X-b.Min.X == 0 {
		return pi
	}
	// rt, dn hold diffused errors.
	type diffusedError struct{ r, g, b int }
	var rt diffusedError
	dn := make([]diffusedError, b.Max.X-b.Min.X+1)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		rt = dn[0]
		dn[0] = diffusedError{}
		for x := b.Min.X; x < b.Max.X; x++ {
			// full color from original image
			r0, g0, b0, _ := i0.At(x, y).RGBA()
			// adjusted full color = original color + diffused error
			rt.r += int(r0)
			rt.g += int(g0)
			rt.b += int(b0)
			// within limits
			if rt.r < 0 {
				rt.r = 0
			} else if rt.r > 0xffff {
				rt.r = 0xffff
			}
			if rt.g < 0 {
				rt.g = 0
			} else if rt.g > 0xffff {
				rt.g = 0xffff
			}
			if rt.b < 0 {
				rt.b = 0
			} else if rt.b > 0xffff {
				rt.b = 0xffff
			}
			afc := color.RGBA64{uint16(rt.r), uint16(rt.g), uint16(rt.b), 0}
			// nearest palette entry
			i := cp.Index(afc)
			// set pixel in destination image
			pi.SetColorIndex(x, y, uint8(i))
			// error to be diffused = full color - palette color.
			e := rt
			pr, pg, pb, _ := cp[i].RGBA()
			e.r -= int(pr)
			e.g -= int(pg)
			e.b -= int(pb)
			// half of error goes right
			rt.r = e.r / 2
			rt.g = e.g / 2
			rt.b = e.b / 2
			// the other half goes down
			e.r -= rt.r
			e.g -= rt.g
			e.b -= rt.b
			dx := x - b.Min.X
			dn[dx+1].r = e.r / 2
			dn[dx+1].g = e.g / 2
			dn[dx+1].b = e.b / 2
			dn[dx].r += e.r - dn[dx+1].r
			dn[dx].g += e.g - dn[dx+1].g
			dn[dx].b += e.b - dn[dx+1].b
		}
	}
	return pi
}
