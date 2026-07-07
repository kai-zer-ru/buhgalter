package qrcode

import (
	"bytes"
	"image"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
)

func exifOrientation(data []byte) int {
	x, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return 1
	}
	tag, err := x.Get(exif.Orientation)
	if err != nil {
		return 1
	}
	orient, err := tag.Int(0)
	if err != nil || orient < 1 || orient > 8 {
		return 1
	}
	return orient
}

func applyOrientation(img image.Image, orient int) image.Image {
	switch orient {
	case 2:
		return imaging.FlipH(img)
	case 3:
		return imaging.Rotate180(img)
	case 4:
		return imaging.FlipV(img)
	case 5:
		return imaging.Rotate270(imaging.FlipH(img))
	case 6:
		return imaging.Rotate270(img)
	case 7:
		return imaging.Rotate90(imaging.FlipH(img))
	case 8:
		return imaging.Rotate90(img)
	default:
		return img
	}
}

func orientedVariants(img image.Image, data []byte) []image.Image {
	oriented := applyOrientation(img, exifOrientation(data))
	variants := []image.Image{oriented}
	if oriented != img {
		variants = append(variants, img)
	}
	b := oriented.Bounds()
	w, h := b.Dx(), b.Dy()
	if h > w*12/10 {
		variants = append(variants,
			imaging.Rotate90(oriented),
			imaging.Rotate270(oriented),
		)
	}
	return variants
}
