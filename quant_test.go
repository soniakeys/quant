package quant_test

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/soniakeys/quant"
	"github.com/soniakeys/quant/median"
)

// TestDither tests Sierra24A on png files found in the source directory.
// Output files are prefixed with _dither_256_.  Files beginning with _
// are skipped when scanning for input files.  Thus nothing is tested
// with a fresh source tree--drop a png or two in the source directory
// before testing to give the test something to work on.
func TestDitherMedianDraw(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	srcDir, _ := filepath.Split(file)
	// ignore file names starting with _, those are result files.
	imgs, err := filepath.Glob(srcDir + "[^_]*.png")
	if err != nil {
		t.Fatal(err)
	}
	const n = 256
	// exercise draw.Quantizer interface
	var q draw.Quantizer = median.Quantizer(n)
	// exercise draw.Drawer interface
	var d draw.Drawer = quant.Sierra24A{}
	for _, p := range imgs {
		f, err := os.Open(p)
		if err != nil {
			t.Error(err) // skip files that can't be opened
			continue
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Error(err) // skip files that can't be decoded
			continue
		}
		pDir, pFile := filepath.Split(p)
		// prefix _ on file name marks this as a result
		fq, err := os.Create(fmt.Sprintf("%s_dither_median_draw_%d_%s", pDir, n, pFile))
		if err != nil {
			t.Fatal(err) // probably can't create any others
		}
		b := img.Bounds()
		pi := image.NewPaletted(b, q.Quantize(make(color.Palette, 0, n), img))
		d.Draw(pi, b, img, b.Min)
		if err = png.Encode(fq, pi); err != nil {
			t.Fatal(err) // any problem is probably a problem for all
		}
	}
}

// TestDither tests Sierra24A on png files found in the source directory.
// Output files are prefixed with _dither_256_.  Files beginning with _
// are skipped when scanning for input files.  Thus nothing is tested
// with a fresh source tree--drop a png or two in the source directory
// before testing to give the test something to work on.
func TestDitherMedianPalette(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	srcDir, _ := filepath.Split(file)
	// ignore file names starting with _, those are result files.
	imgs, err := filepath.Glob(srcDir + "[^_]*.png")
	if err != nil {
		t.Fatal(err)
	}
	const n = 256
	// exercise draw.Quantizer interface
	var q draw.Quantizer = median.Quantizer(n)
	// exercise draw.Drawer interface
	var d draw.Drawer = quant.Sierra24A{}
	for _, p := range imgs {
		f, err := os.Open(p)
		if err != nil {
			t.Error(err) // skip files that can't be opened
			continue
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Error(err) // skip files that can't be decoded
			continue
		}
		pDir, pFile := filepath.Split(p)
		// prefix _ on file name marks this as a result
		fq, err := os.Create(fmt.Sprintf("%s_dither_median_palette_%d_%s", pDir, n, pFile))
		if err != nil {
			t.Fatal(err) // probably can't create any others
		}
		b := img.Bounds()
		pi := image.NewPaletted(b, q.Quantize(make(color.Palette, 0, n), img))
		d.Draw(pi, b, img, b.Min)
		if err = png.Encode(fq, pi); err != nil {
			t.Fatal(err) // any problem is probably a problem for all
		}
	}
}
