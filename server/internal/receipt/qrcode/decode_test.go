package qrcode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeFromImage_fixture(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "..", "..", "web", "e2e", "fixtures", "receipt-qr.png")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("fixture missing")
	}
	qr, err := DecodeFromImage(data)
	if err != nil {
		t.Fatal(err)
	}
	if !looksLikeReceiptQR(qr) {
		t.Fatalf("unexpected qr: %s", qr)
	}
}

func TestDecodeFromImage_blurryPhoto(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "..", "..", "web", "e2e", "fixtures", "receipt-photo-blurry.jpg")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("fixture missing")
	}
	qr, err := DecodeFromImage(data)
	if err != nil {
		t.Fatal(err)
	}
	if !looksLikeReceiptQR(qr) {
		t.Fatalf("unexpected qr: %s", qr)
	}
}
