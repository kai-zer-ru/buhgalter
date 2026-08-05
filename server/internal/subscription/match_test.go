package subscription

import "testing"

func TestNorm(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Яндекс.Плюс", "яндекс плюс"},
		{"  Яндекс  Плюс ", "яндекс плюс"},
		{"Yandex-Plus", "yandex plus"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := Norm(tc.in); got != tc.want {
			t.Fatalf("Norm(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestSimilarityExact(t *testing.T) {
	if Similarity("Яндекс.Плюс", "яндекс плюс") != 1 {
		t.Fatal("expected exact similarity after norm")
	}
}

func TestScoreSameAccountAmountDesc(t *testing.T) {
	desc := "Яндекс Плюс"
	score, reasons := scoreCandidate(scoreInput{
		TxAccountID: "a1", TxAmount: 29900, TxDescription: &desc,
		SubAccountID: "a1", SubAmount: 29900, RefText: "Яндекс.Плюс",
	})
	if score < 50 {
		t.Fatalf("score %d reasons %v", score, reasons)
	}
}

func TestScoreDifferentAmountNonEmptyDesc(t *testing.T) {
	desc := "Netflix"
	score, _ := scoreCandidate(scoreInput{
		TxAccountID: "a2", TxAmount: 99900, TxDescription: &desc,
		SubAccountID: "a1", SubAmount: 49900, RefText: "Netflix",
	})
	if score < 40 {
		t.Fatalf("expected text match score, got %d", score)
	}
}
