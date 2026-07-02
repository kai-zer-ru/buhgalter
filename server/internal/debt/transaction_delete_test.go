package debt

import "testing"

func TestRecalcAmountAfterSettleDelete(t *testing.T) {
	tests := []struct {
		name            string
		currentAmount   int64
		isSettled       int64
		othersSettleSum int64
		deletedAmount   int64
		want            int64
	}{
		{
			name:            "active partial delete only settle",
			currentAmount:   6000,
			isSettled:       0,
			othersSettleSum: 0,
			deletedAmount:   4000,
			want:            10000,
		},
		{
			name:            "active partial delete one of two",
			currentAmount:   4000,
			isSettled:       0,
			othersSettleSum: 6000,
			deletedAmount:   4000,
			want:            8000,
		},
		{
			name:            "settled delete final payment",
			currentAmount:   6000,
			isSettled:       1,
			othersSettleSum: 4000,
			deletedAmount:   6000,
			want:            6000,
		},
		{
			name:            "settled delete partial payment",
			currentAmount:   6000,
			isSettled:       1,
			othersSettleSum: 6000,
			deletedAmount:   4000,
			want:            4000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recalcAmountAfterSettleDelete(tt.currentAmount, tt.isSettled, tt.othersSettleSum, tt.deletedAmount)
			if got != tt.want {
				t.Fatalf("recalcAmountAfterSettleDelete() = %d, want %d", got, tt.want)
			}
		})
	}
}
