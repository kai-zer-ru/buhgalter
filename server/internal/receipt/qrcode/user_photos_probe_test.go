package qrcode

import (
	"os"
	"testing"
)

func TestDecodeUserReceiptPhotos(t *testing.T) {
	paths := []string{
		"/home/kaizer/.cursor/projects/mnt-ssd-GoProjects-buhgalter/assets/IMG_20260706_190518-3cf0bed2-5f48-4a59-bcd6-3d312b0c8a0a.png",
		"/home/kaizer/.cursor/projects/mnt-ssd-GoProjects-buhgalter/assets/IMG_20260706_190531-7271714d-5073-45b8-a6b8-5e483c557908.png",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Skip(path)
		}
		qr, err := DecodeFromImage(data)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		if !looksLikeReceiptQR(qr) {
			t.Errorf("%s: bad qr %q", path, qr)
		}
	}
}
