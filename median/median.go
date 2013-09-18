// Copyright 2013 Sonia Keys.
// Licensed under MIT license.  See "license" file in this source tree.

// Median implements basic median cut color quantization.
package median

import (
	"container/heap"
	"image"
	"image/color"
	"math"
	"sort"
)

type Quantizer struct {}

// Quantize implements median cut color quantization.
//
// Argument n is the desired number of colors.  Returned is a paletted
// image with no more than n colors.
func (Quantizer) Quantize(img image.Image, n int) *image.Paletted {
	qz := newQuantizer(img, n)
	qz.cluster()         // cluster pixels by color
	return qz.paletted() // generate paletted image from clusters
}

type quantizer struct {
	img image.Image // original image
	cs  []cluster   // len(cs) is the desired number of colors
	ch  chValues    // buffer for computing median
}

type point struct{ x, y int32 }
type chValues []uint16
type queue []*cluster

type cluster struct {
	px       []point // list of points in the cluster
	widestCh int     // w const identifying channel with widest value range
}

const ( // w const
	wr = iota
	wg
	wb
)

func newQuantizer(img image.Image, nq int) *quantizer {
	b := img.Bounds()
	npx := (b.Max.X - b.Min.X) * (b.Max.Y - b.Min.Y)
	qz := &quantizer{
		img: img,
		ch:  make(chValues, npx),
		cs:  make([]cluster, nq),
	}
	// Populate initial cluster with all pixels from image.
	c := &qz.cs[0]
	px := make([]point, npx)
	c.px = px
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			px[i].x = int32(x)
			px[i].y = int32(y)
			i++
		}
	}
	return qz
}

// Cluster by repeatedly splitting clusters.
// Use a heap as priority queue for picking clusters to split.
// The rule is to spilt the cluster with the most pixels.
// Terminate when the desired number of clusters has been populated
// or when clusters cannot be further split.
func (qz *quantizer) cluster() {
	pq := new(queue)
	// Initial cluster.  populated at this point, but not analyzed.
	c := &qz.cs[0]
	var m uint32
	for i := 1; ; {
		// Only enqueue clusters that can be split.
		if qz.setWidestChannel(c) {
			heap.Push(pq, c)
		}
		// If no clusters have any color variation, mark the end of the
		// cluster list and quit early.
		if len(*pq) == 0 {
			qz.cs = qz.cs[:i]
			break
		}
		s := heap.Pop(pq).(*cluster) // get cluster to split
		m = qz.medianCut(s)
		c = &qz.cs[i] // set c to new cluster
		i++
		qz.split(s, c, m) // split s into c and s at value m
		// Normal exit is when all clusters are populated.
		if i == len(qz.cs) {
			break
		}
		if qz.setWidestChannel(s) {
			heap.Push(pq, s) // return s to queue
		}
	}
}

func (q *quantizer) setWidestChannel(c *cluster) bool {
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
	c.widestCh = wg
	min := minG
	max := maxG
	if maxR-minR > max-min {
		c.widestCh = wr
		min = minR
		max = maxR
	}
	if maxB-minB > max-min {
		c.widestCh = wb
		min = minB
		max = maxB
	}
	return max > min
}

// Arg c must have value range > 0 in channel c.widestCh.
// return value m is guararanteed to split cluster into two non-empty clusters
// by v < m where v is pixel value of channel c.Widest.
func (q *quantizer) medianCut(c *cluster) uint32 {
	px := c.px
	ch := q.ch[:len(px)]
	// Copy values from appropriate channel to buffer for computing median.
	switch c.widestCh {
	case wr:
		for i, p := range c.px {
			r, _, _, _ := q.img.At(int(p.x), int(p.y)).RGBA()
			ch[i] = uint16(r)
		}
	case wg:
		for i, p := range c.px {
			_, g, _, _ := q.img.At(int(p.x), int(p.y)).RGBA()
			ch[i] = uint16(g)
		}
	case wb:
		for i, p := range c.px {
			_, _, b, _ := q.img.At(int(p.x), int(p.y)).RGBA()
			ch[i] = uint16(b)
		}
	}
	// Find cut.
	sort.Sort(ch)
	m1 := len(ch) / 2 // median
	if ch[m1] != ch[m1-1] {
		return uint32(ch[m1])
	}
	m2 := m1
	// Dec m1 until element to left is different.
	for m1--; m1 > 0 && ch[m1] == ch[m1-1]; m1-- {
	}
	// Inc m2 until element to left is different.
	for m2++; m2 < len(ch) && ch[m2] == ch[m2-1]; m2++ {
	}
	// Return value that makes more equitable cut.
	if m1 > len(ch)-m2 {
		return uint32(ch[m1])
	}
	return uint32(ch[m2])
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
		// Split at m.
		if v < m {
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
		// Set image pixels.
		for _, p := range px {
			pi.SetColorIndex(int(p.x), int(p.y), uint8(i))
		}
	}
	return pi
}

// Implement sort.Interface for sort in median algorithm.
func (c chValues) Len() int           { return len(c) }
func (c chValues) Less(i, j int) bool { return c[i] < c[j] }
func (c chValues) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

// Implement heap.Interface for priority queue of clusters.
func (q queue) Len() int { return len(q) }

// Priority is number of pixels in cluster.
func (q queue) Less(i, j int) bool { return len(q[i].px) > len(q[j].px) }
func (q queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}
func (pq *queue) Push(x interface{}) {
	c := x.(*cluster)
	*pq = append(*pq, c)
}
func (pq *queue) Pop() interface{} {
	q := *pq
	n := len(q) - 1
	c := q[n]
	*pq = q[:n]
	return c
}
