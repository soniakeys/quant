// Copyright 2013 Sonia Keys.
// Licensed under MIT license.  See "license" file in this source tree.

// Quant provides an interface for image color quantizers.
package quant

import (
	"image"
	"image/color"
	"image/draw"
	"math"
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
	// dither211 currently returns a new image, or nil if dithering not
	// possible.
	if s := dither211(src, pd.Palette); s != nil {
		src = s
	}
	draw.Draw(dst, r, src, image.Point{}, draw.Src)
}

// signed color type, no alpha.  signed to represent color deltas as well as
// color values 0-ffff as with colorRGBA64
type sRGB struct{ r, g, b int32 }

// a linear palette, but with signed values.  while the fields hold uint32s,
// values must be restricted to 0-ffff as with color.RGBA64.  values are signed
// here to facilitate arithmetic, not to represent some new color space.
type sPalette []sRGB

// like PaletteIndex method
func (p sPalette) index(c sRGB) int {
	// still the awful linear search
	i, min := 0, int64(math.MaxInt64)
	for j, pc := range p {
		d := int64(c.r - pc.r)
		s := d * d
		d = int64(c.g - pc.g)
		s += d * d
		d = int64(c.b - pc.b)
		s += d * d
		if s < min {
			min = s
			i = j
		}
	}
	return i
}

// currently this is strictly a helper function for Dither211.Draw, so
// not generalized to use Palette from this package.
func dither211(i0 image.Image, cp color.Palette) *image.Paletted {
	if len(cp) > 256 {
		// representation limit of image.Paletted.  a little sketchy to return
		// nil, but unworkable results are always better than wrong results.
		return nil
	}
	b := i0.Bounds()
	pi := image.NewPaletted(b, cp)
	if b.Empty() {
		return pi // no work to do
	}
	p64 := make([]color.RGBA64, len(cp))
	sp := make(sPalette, len(cp))
	for i, c := range cp {
		r, g, b, _ := c.RGBA()
		p64[i] = color.RGBA64{uint16(r), uint16(g), uint16(b), 0xffff}
		sp[i] = sRGB{int32(r), int32(g), int32(b)}
	}
	// rt, dn hold diffused errors.
	var rt sRGB
	dn := make([]sRGB, b.Dx()+1)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		rt = dn[1]
		dn[0] = sRGB{}
		for x := b.Min.X; x < b.Max.X; x++ {
			// full color from original image
			r0, g0, b0, _ := i0.At(x, y).RGBA()
			// adjusted full color = diffused err + original color
			rt.r += int32(r0)
			rt.g += int32(g0)
			rt.b += int32(b0)
			// nearest palette entry
			i := sp.index(rt)
			// set pixel in destination image
			pi.SetColorIndex(x, y, uint8(i))
			// error to be diffused = full color - palette color.
			e := rt
			pc := sp[i]
			e.r -= pc.r
			e.g -= pc.g
			e.b -= pc.b
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
