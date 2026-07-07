package qrcode

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/makiuchi-d/gozxing"
	zqr "github.com/makiuchi-d/gozxing/qrcode"
	goqr "github.com/piglig/go-qr"
	_ "golang.org/x/image/webp"
)

var ErrNotFound = errors.New("qr not found in image")

const maxSide = 2400

func DecodeFromImage(data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	for _, base := range orientedVariants(img, data) {
		if qr, ok := decodeVariants(base); ok {
			return qr, nil
		}
	}
	return "", ErrNotFound
}

func decodeVariants(img image.Image) (string, bool) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	crops := receiptCrops(w, h)
	scales := []float64{1, 1.5, 2, 3, 4}
	for _, crop := range crops {
		cropped := imaging.Crop(img, crop)
		for _, scale := range scales {
			resized := resizeCrop(cropped, crop, scale)
			if qr, ok := tryDecodeImage(resized); ok {
				return qr, true
			}
		}
	}
	return "", false
}

func receiptCrops(w, h int) []image.Rectangle {
	full := image.Rect(0, 0, w, h)
	crops := []image.Rectangle{full}
	for _, start := range []float64{0.35, 0.4, 0.45, 0.5, 0.55, 0.6} {
		y := int(float64(h) * start)
		if y >= h-32 {
			continue
		}
		crops = append(crops, image.Rect(0, y, w, h))
	}
	return crops
}

func resizeCrop(cropped image.Image, crop image.Rectangle, scale float64) image.Image {
	sw := crop.Dx()
	sh := crop.Dy()
	longSide := max(sw, sh)
	target := int(float64(longSide) * scale)
	if target > maxSide {
		target = maxSide
	}
	if target < 320 {
		target = 320
	}
	return imaging.Resize(cropped, target, 0, imaging.Lanczos)
}

func tryDecodeImage(img image.Image) (string, bool) {
	if qr, ok := decodeAnyQR(img); ok && looksLikeReceiptQR(qr) {
		return qr, true
	}
	gray := toGray(imaging.Grayscale(img))
	sharpened := toGray(imaging.Sharpen(imaging.Grayscale(img), 0.8))
	variants := []*image.Gray{
		gray,
		sharpened,
		threshold(gray, otsu(gray)),
		threshold(sharpened, otsu(sharpened)),
		invertGray(gray),
	}
	if grayBoundsPixels(gray) <= 900_000 {
		variants = append(variants,
			adaptiveThreshold(gray, 31, 5),
			adaptiveThreshold(sharpened, 31, 5),
		)
	}
	for _, variant := range variants {
		if qr, ok := decodeAnyQR(variant); ok && looksLikeReceiptQR(qr) {
			return qr, true
		}
	}
	return "", false
}

func decodeAnyQR(img image.Image) (string, bool) {
	if qr, err := goqr.Decode(img); err == nil && qr != "" {
		return qr, true
	}
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", false
	}
	result, err := zqr.NewQRCodeReader().Decode(bmp, nil)
	if err == nil && result.GetText() != "" {
		return result.GetText(), true
	}
	return "", false
}

func grayBoundsPixels(g *image.Gray) int {
	b := g.Bounds()
	return b.Dx() * b.Dy()
}

func invertGray(g *image.Gray) *image.Gray {
	out := image.NewGray(g.Bounds())
	for i, v := range g.Pix {
		out.Pix[i] = 255 - v
	}
	return out
}

func looksLikeReceiptQR(text string) bool {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)
	if strings.Contains(lower, "fn=") && (strings.Contains(lower, "t=") || strings.Contains(lower, "fp=")) {
		return true
	}
	return strings.HasPrefix(lower, "t=") && strings.Contains(lower, "s=")
}

func toGray(img image.Image) *image.Gray {
	b := img.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.SetGray(x, y, color.GrayModel.Convert(img.At(x, y)).(color.Gray))
		}
	}
	return out
}

func otsu(g *image.Gray) uint8 {
	hist := [256]int{}
	for _, v := range g.Pix {
		hist[v]++
	}
	var sum, sumB, wB, max, threshold int
	for i := 0; i < 256; i++ {
		sum += i * hist[i]
	}
	for t := 0; t < 256; t++ {
		wB += hist[t]
		if wB == 0 {
			continue
		}
		wF := len(g.Pix) - wB
		if wF == 0 {
			break
		}
		sumB += t * hist[t]
		mB := float64(sumB) / float64(wB)
		mF := float64(sum-sumB) / float64(wF)
		between := float64(wB*wF) * (mB - mF) * (mB - mF)
		if int(between) > max {
			max = int(between)
			threshold = t
		}
	}
	return uint8(threshold)
}

func threshold(g *image.Gray, th uint8) *image.Gray {
	out := image.NewGray(g.Bounds())
	for i, v := range g.Pix {
		if v > th {
			out.Pix[i] = 255
		} else {
			out.Pix[i] = 0
		}
	}
	return out
}

func adaptiveThreshold(g *image.Gray, blockSize, c int) *image.Gray {
	b := g.Bounds()
	w, h := b.Dx(), b.Dy()
	integral := make([]int64, (w+1)*(h+1))
	for y := 0; y < h; y++ {
		var rowSum int64
		for x := 0; x < w; x++ {
			rowSum += int64(g.GrayAt(x, y).Y)
			integral[(y+1)*(w+1)+(x+1)] = rowSum + integral[y*(w+1)+(x+1)]
		}
	}
	out := image.NewGray(b)
	radius := blockSize / 2
	for y := 0; y < h; y++ {
		y1 := maxInt(0, y-radius)
		y2 := minInt(h-1, y+radius)
		for x := 0; x < w; x++ {
			x1 := maxInt(0, x-radius)
			x2 := minInt(w-1, x+radius)
			count := int64((x2 - x1 + 1) * (y2 - y1 + 1))
			sum := integral[(y2+1)*(w+1)+(x2+1)] -
				integral[y1*(w+1)+(x2+1)] -
				integral[(y2+1)*(w+1)+x1] +
				integral[y1*(w+1)+x1]
			v := uint8(0)
			if int64(g.GrayAt(x, y).Y) > sum/count-int64(c) {
				v = 255
			}
			out.SetGray(x, y, color.Gray{Y: v})
		}
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
