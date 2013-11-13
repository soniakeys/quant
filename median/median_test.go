package median_test

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/soniakeys/quant"
	"github.com/soniakeys/quant/median"
)

// TestMedian tests the median quantizer on png files found in the source
// directory.  Output files are prefixed with _median_.  Files begining with
// _ are skipped when scanning for input files.  Thus nothing is tested
// with a fresh source tree--drop a png or two in the source directory
// before testing to give the test something to work on.  Png files in the
// parent directory are similarly used for testing.  Put files there
// to compare results of the different quantizers.
func TestMedian(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller fail")
	}
	srcDir, _ := filepath.Split(file)
	// ignore file names starting with _, those are result files.
	imgs, err := filepath.Glob(srcDir + "[^_]*.png")
	if err != nil {
		t.Fatal(err)
	}
	if srcDir > "" {
		parentDir, _ := filepath.Split(srcDir[:len(srcDir)-1])
		parentImgs, err := filepath.Glob(parentDir + "[^_]*.png")
		if err != nil {
			t.Fatal(err)
		}
		imgs = append(parentImgs, imgs...)
	}
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
		for _, n := range []int{16, 256} {
			// prefix _ on file name marks this as a result
			fq, err := os.Create(fmt.Sprintf("%s_median_%d_%s", pDir, n, pFile))
			if err != nil {
				t.Fatal(err) // probably can't create any others
			}
			var q quant.Quantizer = median.Quantizer(n)
			if err = png.Encode(fq, q.Image(img)); err != nil {
				t.Fatal(err) // any problem is probably a problem for all
			}
		}
	}
}
