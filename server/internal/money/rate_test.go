package money

import "testing"

func TestRate(t *testing.T) {
	cases := []struct {
		amount, toAmount int64
		want             string
	}{
		{900000, 10000, "0.011111"},
		{10000, 10000, "1.000000"},
		{10000, 900000, "90.000000"},
		{3, 2, "0.666667"},
		{0, 100, ""},
	}
	for _, c := range cases {
		if got := Rate(c.amount, c.toAmount); got != c.want {
			t.Fatalf("Rate(%d, %d) = %q, want %q", c.amount, c.toAmount, got, c.want)
		}
	}
}
