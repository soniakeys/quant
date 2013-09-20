// Copyright 2013 Sonia Keys.
// Licensed under MIT license.  See "license" file in this source tree.

// Mean is a simple color quantizer.
package mean

import (
	"image"
	"image/color"
	"math"

	"github.com/soniakeys/quant"
)

// Quantizer implements quant.Quantizer with a simple mean-based algorithm.
type Quantizer struct{}

var _ quant.Quantizer = Quantizer{}

// Image performs color quantization and returns a paletted image.
//
// Argument n is the desired number of colors.  Returned is a paletted
// image with no more than n colors.
func (Quantizer) Image(img image.Image, n int) *image.Paletted {
	if n > 256 {
		n = 256
	}
	qz := newQuantizer(img, n)
	if n > 1 {
		qz.cluster() // cluster pixels by color
	}
	return qz.paletted() // generate paletted image from clusters
}

func (Quantizer) Palette(img image.Image, n int) quant.Palette {
	qz := newQuantizer(img, n)
	if n > 1 {
		qz.cluster() // cluster pixels by color
	}
	return qz.palette()
}

type quantizer struct {
	img image.Image // original image
	cs  []cluster   // len(cs) is the desired number of colors
}

type point struct{ x, y int32 }

type cluster struct {
	px       []point // list of points in the cluster
	widestCh int     // w const identifying channel with widest value range
	min, max uint32  // min, max color values of widest channel
	volume   uint64  // color volume
	priority int     // early: population, late: population*volume
}

const ( // w const
	wr = iota
	wg
	wb
)

func newQuantizer(img image.Image, n int) *quantizer {
	if n < 1 {
		return &quantizer{img, nil}
	}
	// Make list of all pixels in image.
	b := img.Bounds()
	px := make([]point, (b.Max.X-b.Min.X)*(b.Max.Y-b.Min.Y))
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			px[i].x = int32(x)
			px[i].y = int32(y)
			i++
		}
	}
	// Make clusters, populate first cluster with complete pixel list.
	cs := make([]cluster, n)
	cs[0].px = px
	return &quantizer{img, cs}
}

// Cluster by repeatedly splitting clusters in two stages.  For the first
// stage, prioritize by population and split tails off distribution of
// color channel with widest range.  For the second stage, prioritize by
// the product of population and color volume, and split at the mean of
// the color channel with widest range.  Terminate when the desired number
// of clusters has been populated or when clusters cannot be further split.
func (qz *quantizer) cluster() {
	cs := qz.cs
	half := len(cs) / 2
	// cx is index of new cluster, populated at start of loop here, but
	// not yet analyzed.
	cx := 0
	c := &cs[cx]
	for {
		qz.setPriority(c, cx < half) // compute statistics for new cluster
		// determine cluster to split, sx
		sx := -1
		var maxP int
		for x := 0; x <= cx; x++ {
			// rule is to consider only clusters with non-zero color volume
			// and then split cluster with highest priority.
			if c := &cs[x]; c.max > c.min && c.priority > maxP {
				maxP = c.priority
				sx = x
			}
		}
		// If no clusters have any color variation, mark the end of the
		// cluster list and quit early.
		if sx < 0 {
			qz.cs = qz.cs[:cx+1]
			break
		}
		s := &cs[sx]
		m := qz.cutValue(s, cx < half) // get where to split cluster
		// point to next cluster to populate
		cx++
		c = &cs[cx]
		// populate c by splitting s into c and s at value m
		qz.split(s, c, m)
		// Normal exit is when all clusters are populated.
		if cx == len(cs)-1 {
			break
		}
		if cx == half {
			// change priorities on existing clusters
			for x := 0; x < cx; x++ {
				cs[x].priority =
					int(uint64(cs[x].priority) * (cs[x].volume >> 16) >> 29)
			}
		}
		qz.setPriority(s, cx < half) // set priority for newly split s
	}
}

func (q *quantizer) setPriority(c *cluster, early bool) {
	// Find extents of color values in each channel.
	var maxR, maxG, maxB uint32
	minR := uint32(math.MaxUint32)
	minG := uint32(math.MaxUint32)
	minB := uint32(math.MaxUint32)
	for _, p := range c.px {
		r, g, b, _ := q.img.At(int(p.x), int(p.y)).RGBA()
		if r < minR {
			minR = r
		}
		if r > maxR {
			maxR = r
		}
		if g < minG {
			minG = g
		}
		if g > maxG {
			maxG = g
		}
		if b < minB {
			minB = b
		}
		if b > maxB {
			maxB = b
		}
	}
	// See which channel had the widest range.
	w := wg
	min := minG
	max := maxG
	if maxR-minR > max-min {
		w = wr
		min = minR
		max = maxR
	}
	if maxB-minB > max-min {
		w = wb
		min = minB
		max = maxB
	}
	// store statistics
	c.widestCh = w
	c.min = min
	c.max = max
	c.volume = uint64(maxR-minR) * uint64(maxG-minG) * uint64(maxB-minB)
	c.priority = len(c.px)
	if !early {
		c.priority = int(uint64(c.priority) * (c.volume >> 16) >> 29)
	}
}

func (q *quantizer) cutValue(c *cluster, early bool) uint32 {
	var sum uint64
	switch c.widestCh {
	case wr:
		for _, p := range c.px {
			r, _, _, _ := q.img.At(int(p.x), int(p.y)).RGBA()
			sum += uint64(r)
		}
	case wg:
		for _, p := range c.px {
			_, g, _, _ := q.img.At(int(p.x), int(p.y)).RGBA()
			sum += uint64(g)
		}
	case wb:
		for _, p := range c.px {
			_, _, b, _ := q.img.At(int(p.x), int(p.y)).RGBA()
			sum += uint64(b)
		}
	}
	mean := uint32(sum / uint64(len(c.px)))
	if early {
		// split in middle of longer tail rather than at mean
		if c.max-mean > mean-c.min {
			mean = (mean + c.max) / 2
		} else {
			mean = (mean + c.min) / 2
		}
	}
	return mean
}

func (q *quantizer) split(s, c *cluster, m uint32) {
	px := s.px
	var v uint32
	i := 0
	last := len(px) - 1
	for i <= last {
		// Get pixel value of appropriate channel.
		r, g, b, _ := q.img.At(int(px[i].x), int(px[i].y)).RGBA()
		switch s.widestCh {
		case wr:
			v = r
		case wg:
			v = g
		case wb:
			v = b
		}
		// Split into two non-empty parts at m.
		if v < m || m == s.min && v == m {
			i++
		} else {
			px[last], px[i] = px[i], px[last]
			last--
		}
	}
	// Split the pixel list.
	s.px = px[:i]
	c.px = px[i:]
}

func (qz *quantizer) paletted() *image.Paletted {
	cp := make(color.Palette, len(qz.cs))
	pi := image.NewPaletted(qz.img.Bounds(), cp)
	for i := range qz.cs {
		px := qz.cs[i].px
		// Average values in cluster to get palette color.
		var rsum, gsum, bsum int64
		for _, p := range px {
			r, g, b, _ := qz.img.At(int(p.x), int(p.y)).RGBA()
			rsum += int64(r)
			gsum += int64(g)
			bsum += int64(b)
		}
		n64 := int64(len(px) << 8)
		cp[i] = color.RGBA{
			uint8(rsum / n64),
			uint8(gsum / n64),
			uint8(bsum / n64),
			0xff,
		}
		// set image pixels
		for _, p := range px {
			pi.SetColorIndex(int(p.x), int(p.y), uint8(i))
		}
	}
	return pi
}

func (qz *quantizer) palette() quant.Palette {
	cp := make(color.Palette, len(qz.cs))
	for i := range qz.cs {
		px := qz.cs[i].px
		// Average values in cluster to get palette color.
		var rsum, gsum, bsum int64
		for _, p := range px {
			r, g, b, _ := qz.img.At(int(p.x), int(p.y)).RGBA()
			rsum += int64(r)
			gsum += int64(g)
			bsum += int64(b)
		}
		n64 := int64(len(px) << 8)
		cp[i] = color.RGBA{
			uint8(rsum / n64),
			uint8(gsum / n64),
			uint8(bsum / n64),
			0xff,
		}
	}
	return quant.LinearPalette{cp}
}
