package versioncheck

import (
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		current string
		latest  string
		want    int
	}{
		{"1.2.1", "1.2.2", -1},
		{"v1.2.2", "1.2.2", 0},
		{"1.3.0", "1.2.9", 1},
		{"1.2.2", "1.2.2-beta", 0},
	}
	for _, tc := range cases {
		if got := compareVersions(tc.current, tc.latest); got != tc.want {
			t.Fatalf("compareVersions(%q, %q) = %d, want %d", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestCheckerUsesCache(t *testing.T) {
	t.Parallel()

	checker := NewChecker("1.0.0")
	checker.cacheTTL = time.Hour
	checker.storeRelease(cachedRelease{
		fetchedAt: time.Now(),
		tagName:   "v9.9.9",
		htmlURL:   "https://example.com/release",
	})

	first := checker.Check(t.Context())
	second := checker.Check(t.Context())

	if !first.UpdateAvailable || first.LatestVersion != "9.9.9" {
		t.Fatalf("unexpected first result: %+v", first)
	}
	if !second.UpdateAvailable || second.LatestVersion != "9.9.9" {
		t.Fatalf("unexpected second result: %+v", second)
	}
}

func TestCheckerWithoutCurrentVersion(t *testing.T) {
	t.Parallel()

	result := NewChecker("").Check(t.Context())
	if result.CurrentVersion != "" || result.UpdateAvailable {
		t.Fatalf("unexpected result: %+v", result)
	}
}
