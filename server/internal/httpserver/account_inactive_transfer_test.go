package httpserver

import (
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/account"
)

func TestInactiveAccountTransferAmount(t *testing.T) {
	t.Parallel()

	cash := func(id string, balance int64) account.Account {
		return account.Account{ID: id, Type: "cash", Balance: balance}
	}

	tests := []struct {
		name     string
		acc      account.Account
		computed map[string]int64
		want     int64
	}{
		{
			name:     "uses stored balance when higher than computed",
			acc:      cash("a", 50_000),
			computed: map[string]int64{"a": 0},
			want:     50_000,
		},
		{
			name:     "uses computed when higher than stored balance",
			acc:      cash("a", 10_000),
			computed: map[string]int64{"a": 25_000},
			want:     25_000,
		},
		{
			name:     "credit card never needs transfer amount",
			acc:      account.Account{ID: "cc", Type: "credit_card", Balance: 12_000},
			computed: map[string]int64{"cc": 12_000},
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inactiveAccountTransferAmount(tt.acc, tt.computed); got != tt.want {
				t.Fatalf("inactiveAccountTransferAmount() = %d, want %d", got, tt.want)
			}
		})
	}
}
