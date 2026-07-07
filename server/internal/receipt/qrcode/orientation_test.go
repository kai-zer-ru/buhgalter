package qrcode

import (
	"image"
	"testing"

	"github.com/disintegration/imaging"
)

func TestApplyOrientation(t *testing.T) {
	t.Parallel()
	src := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for orient, rotate := range map[int]func(image.Image) image.Image{
		1: func(img image.Image) image.Image { return img },
		3: func(img image.Image) image.Image { return imaging.Rotate180(img) },
		6: func(img image.Image) image.Image { return imaging.Rotate270(img) },
		8: func(img image.Image) image.Image { return imaging.Rotate90(img) },
	} {
		got := applyOrientation(src, orient)
		want := rotate(src)
		gb, wb := got.Bounds(), want.Bounds()
		if gb.Dx() != wb.Dx() || gb.Dy() != wb.Dy() {
			t.Fatalf("orient %d: bounds %v vs %v", orient, gb, wb)
		}
	}
}
