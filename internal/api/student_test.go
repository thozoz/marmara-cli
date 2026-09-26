package api

import (
	"context"
	"strings"
	"testing"
)

func TestScheduleTerm_Validation(t *testing.T) {
	a := &API{}

	testCases := []struct {
		name  string
		yil   int
		donem int
	}{
		{"zero year", 0, 100001},
		{"negative year", -1, 100001},
		{"zero term", 2026, 0},
		{"negative term", 2026, -1},
		{"both zero", 0, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := a.ScheduleTerm(context.Background(), tc.yil, tc.donem)
			if err == nil {
				t.Fatalf("expected error for yil=%d donem=%d, got nil", tc.yil, tc.donem)
			}
			if !strings.Contains(err.Error(), "must be positive") {
				t.Errorf("unexpected error message: %v", err)
			}
		})
	}
}
