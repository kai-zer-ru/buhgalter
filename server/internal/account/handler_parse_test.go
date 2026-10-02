package account

import (
	"errors"
	"testing"
)

func TestParseAutoTopupInputInvalidThreshold(t *testing.T) {
	enabled := true
	bad := "not-a-number"
	_, err := parseAutoTopupInput(updateRequest{
		AutoTopupEnabled:   &enabled,
		AutoTopupThreshold: &bad,
	})
	if !errors.Is(err, ErrInvalidAutoTopupThreshold) {
		t.Fatalf("got %v", err)
	}
}

func TestParseAutoTopupInputInvalidTarget(t *testing.T) {
	enabled := true
	bad := "12.34.56"
	_, err := parseAutoTopupInput(updateRequest{
		AutoTopupEnabled: &enabled,
		AutoTopupTarget:  &bad,
	})
	if !errors.Is(err, ErrInvalidAutoTopupTarget) {
		t.Fatalf("got %v", err)
	}
}

func TestParseAutoTopupInputOmitsAutoTopup(t *testing.T) {
	in, err := parseAutoTopupInput(updateRequest{Name: "x"})
	if err != nil || in != nil {
		t.Fatalf("in=%v err=%v", in, err)
	}
}

func TestParseAutoTopupInputValidAmounts(t *testing.T) {
	enabled := true
	thr := "100.50"
	tgt := "5000"
	in, err := parseAutoTopupInput(updateRequest{
		AutoTopupEnabled:   &enabled,
		AutoTopupThreshold: &thr,
		AutoTopupTarget:    &tgt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !in.Enabled || in.Threshold != 10050 || in.Target != 500000 {
		t.Fatalf("in=%+v", in)
	}
}
