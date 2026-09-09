package metrics

import (
	"testing"
)

func TestCalculateStats(t *testing.T) {
	tests := []struct {
		name          string
		values        []float64
		unit          string
		expectedCount int
		expectedMin   float64
		expectedMax   float64
		expectedMean  float64
		expectedP50   float64
		expectedP90   float64
	}{
		{
			name:          "standard 1 to 10 sequence",
			values:        []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
			unit:          "seconds",
			expectedCount: 10,
			expectedMin:   10,
			expectedMax:   100,
			expectedMean:  55,
			expectedP50:   55,
			expectedP90:   91,
		},
		{
			name:          "single value dataset",
			values:        []float64{42},
			unit:          "lines",
			expectedCount: 1,
			expectedMin:   42,
			expectedMax:   42,
			expectedMean:  42,
			expectedP50:   42,
			expectedP90:   42,
		},
		{
			name:          "empty dataset",
			values:        []float64{},
			unit:          "seconds",
			expectedCount: 0,
			expectedMin:   0,
			expectedMax:   0,
			expectedMean:  0,
			expectedP50:   0,
			expectedP90:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stats := CalculateStats(tc.values, tc.unit)
			if stats.Count != tc.expectedCount {
				t.Errorf("expected count %d, got %d", tc.expectedCount, stats.Count)
			}
			if stats.Min != tc.expectedMin {
				t.Errorf("expected min %.1f, got %.1f", tc.expectedMin, stats.Min)
			}
			if stats.Max != tc.expectedMax {
				t.Errorf("expected max %.1f, got %.1f", tc.expectedMax, stats.Max)
			}
			if stats.Mean != tc.expectedMean {
				t.Errorf("expected mean %.1f, got %.1f", tc.expectedMean, stats.Mean)
			}
			if stats.P50 != tc.expectedP50 {
				t.Errorf("expected p50 %.1f, got %.1f", tc.expectedP50, stats.P50)
			}
			if stats.P90 != tc.expectedP90 {
				t.Errorf("expected p90 %.1f, got %.1f", tc.expectedP90, stats.P90)
			}
			if stats.Unit != tc.unit {
				t.Errorf("expected unit %s, got %s", tc.unit, stats.Unit)
			}
		})
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name       string
		data       []float64
		percentile float64
		expected   float64
	}{
		{
			name:       "empty data slice",
			data:       []float64{},
			percentile: 50,
			expected:   0,
		},
		{
			name:       "single element p50",
			data:       []float64{100},
			percentile: 50,
			expected:   100,
		},
		{
			name:       "two elements p50",
			data:       []float64{10, 20},
			percentile: 50,
			expected:   15,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Percentile(tc.data, tc.percentile)
			if got != tc.expected {
				t.Errorf("Percentile(%.1f) = %f, expected %f", tc.percentile, got, tc.expected)
			}
		})
	}
}

func TestFormatDurationHours(t *testing.T) {
	tests := []struct {
		name     string
		seconds  float64
		expected string
	}{
		{name: "seconds unit", seconds: 30, expected: "30s"},
		{name: "minutes unit", seconds: 120, expected: "2.0m"},
		{name: "hours unit", seconds: 7200, expected: "2.0h"},
		{name: "days unit", seconds: 86400 * 2, expected: "2.0d"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatDurationHours(tc.seconds)
			if got != tc.expected {
				t.Errorf("FormatDurationHours(%.0f) = %s, expected %s", tc.seconds, got, tc.expected)
			}
		})
	}
}

func TestDetermineSizeCategory(t *testing.T) {
	tests := []struct {
		name      string
		additions int
		deletions int
		expected  string
	}{
		{name: "small diff under 100 lines", additions: 20, deletions: 30, expected: "S (<100)"},
		{name: "medium diff 100 to 499 lines", additions: 150, deletions: 50, expected: "M (100-499)"},
		{name: "large diff 500 to 999 lines", additions: 400, deletions: 300, expected: "L (500-999)"},
		{name: "extra large diff 1000+ lines", additions: 800, deletions: 500, expected: "XL (1000+)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := DetermineSizeCategory(tc.additions, tc.deletions)
			if s != tc.expected {
				t.Errorf("DetermineSizeCategory(%d, %d) = %s, expected %s", tc.additions, tc.deletions, s, tc.expected)
			}
		})
	}
}
