// Copyright 2013 Sonia Keys.
// Licensed under MIT license.  See "license" file in this source tree.

// Quant provides an interface for image color quantizers.
package quant

import "image"

// Quantizer defines a color quantizer for images.
type Quantizer interface {
	// Quantize int argument specifies the desired number of colors
	// in the result image.
	Quantize(image.Image, int) *image.Paletted
}
