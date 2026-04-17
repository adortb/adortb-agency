package commission

import (
	"testing"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name       string
		totalSpend float64
		rate       float64
		want       float64
	}{
		{"10% rate", 10000, 0.10, 1000},
		{"15% rate", 50000, 0.15, 7500},
		{"5% rate", 0, 0.05, 0},
		{"zero rate", 10000, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Calculate(tt.totalSpend, tt.rate)
			if got != tt.want {
				t.Errorf("Calculate(%v, %v) = %v, want %v", tt.totalSpend, tt.rate, got, tt.want)
			}
		})
	}
}
