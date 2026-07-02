package account

import "testing"

func TestRequiresBalanceTransfer(t *testing.T) {
	tests := []struct {
		name string
		acc  Account
		want bool
	}{
		{
			name: "cash with balance",
			acc:  Account{Type: "cash", Balance: 100},
			want: true,
		},
		{
			name: "bank with balance",
			acc:  Account{Type: "bank", Balance: 1},
			want: true,
		},
		{
			name: "cash zero",
			acc:  Account{Type: "cash", Balance: 0},
			want: false,
		},
		{
			name: "credit card with balance",
			acc:  Account{Type: "credit_card", Balance: 50000},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RequiresBalanceTransfer(tt.acc); got != tt.want {
				t.Fatalf("RequiresBalanceTransfer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteTransferDescription(t *testing.T) {
	got := DeleteTransferDescription("Сбербанк")
	want := `Удаление счёта "Сбербанк"`
	if got != want {
		t.Fatalf("DeleteTransferDescription() = %q, want %q", got, want)
	}
}

func TestValidateCreditCardFullyPaid(t *testing.T) {
	limit := int64(100_000)
	tests := []struct {
		name    string
		acc     Account
		wantErr bool
	}{
		{
			name: "cash skipped",
			acc:  Account{Type: "cash", Balance: 0},
		},
		{
			name:    "credit card under limit",
			acc:     Account{Type: "credit_card", Balance: 50_000, CreditLimit: &limit},
			wantErr: true,
		},
		{
			name: "credit card at limit",
			acc:  Account{Type: "credit_card", Balance: 100_000, CreditLimit: &limit},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreditCardFullyPaid(tt.acc)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
