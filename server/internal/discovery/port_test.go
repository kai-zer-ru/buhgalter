package discovery

import "testing"

func TestParseHTTPPort(t *testing.T) {
	tests := []struct {
		addr string
		want int
	}{
		{":8765", 8765},
		{"0.0.0.0:8765", 8765},
		{"127.0.0.1:8765", 8765},
		{"8765", 8765},
	}
	for _, tt := range tests {
		got, err := ParseHTTPPort(tt.addr)
		if err != nil {
			t.Fatalf("ParseHTTPPort(%q): %v", tt.addr, err)
		}
		if got != tt.want {
			t.Fatalf("ParseHTTPPort(%q) = %d, want %d", tt.addr, got, tt.want)
		}
	}
}

func TestParseHTTPPort_invalid(t *testing.T) {
	if _, err := ParseHTTPPort(""); err == nil {
		t.Fatal("expected error for empty addr")
	}
	if _, err := ParseHTTPPort(":0"); err == nil {
		t.Fatal("expected error for port 0")
	}
}
