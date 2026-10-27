package transaction

import (
	"errors"
	"testing"
)

func TestTransferToAmount(t *testing.T) {
	cases := []struct {
		name             string
		from, to         string
		amount, toAmount int64
		want             int64
		wantErr          error
	}{
		{"same currency without to_amount", "RUB", "RUB", 10000, 0, 10000, nil},
		{"same currency equal to_amount", "RUB", "RUB", 10000, 10000, 10000, nil},
		{"same currency different to_amount", "RUB", "RUB", 10000, 9000, 0, ErrTransferToAmountMismatch},
		{"exchange with to_amount", "RUB", "USD", 900000, 10000, 10000, nil},
		{"exchange without to_amount", "RUB", "USD", 900000, 0, 0, ErrTransferToAmountRequired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := transferToAmount(c.from, c.to, c.amount, c.toAmount)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err %v, want %v", err, c.wantErr)
			}
			if got != c.want {
				t.Fatalf("got %d, want %d", got, c.want)
			}
		})
	}
}

func TestParseTransferInputToAmount(t *testing.T) {
	base := func(toAmount *string) transferRequest {
		return transferRequest{
			FromAccountID: "a", ToAccountID: "b", Amount: "100.00",
			ToAmount: toAmount, TransactionDate: "2020-06-01 10:00:00",
		}
	}
	str := func(s string) *string { return &s }

	for _, raw := range []*string{nil, str(""), str("  ")} {
		in, err := parseTransferInput(base(raw))
		if err != nil || in.ToAmount != 0 {
			t.Fatalf("absent to_amount %v: %+v %v", raw, in, err)
		}
	}
	in, err := parseTransferInput(base(str("90.50")))
	if err != nil || in.ToAmount != 9050 {
		t.Fatalf("to_amount 90.50: %+v %v", in, err)
	}
	for _, bad := range []string{"0", "0.00", "-1.00", "abc"} {
		if _, err := parseTransferInput(base(str(bad))); err == nil {
			t.Fatalf("to_amount %q accepted", bad)
		}
	}
}
