package metrics

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// CalculateStats computes statistical percentiles (p50, p75, p90, p95, p99), mean, min,
// and max for a given slice of floating-point values.
//
// If values is empty, it returns an empty GroupSummary initialized with the specified unit.
// The input slice is copied and sorted to prevent mutating the caller's data.
func CalculateStats(values []float64, unit string) GroupSummary {
	if len(values) == 0 {
		return GroupSummary{
			Unit: unit,
		}
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	mean := sum / float64(len(sorted))

	return GroupSummary{
		Count: len(sorted),
		P50:   Percentile(sorted, 50),
		P75:   Percentile(sorted, 75),
		P90:   Percentile(sorted, 90),
		P95:   Percentile(sorted, 95),
		P99:   Percentile(sorted, 99),
		Mean:  mean,
		Min:   sorted[0],
		Max:   sorted[len(sorted)-1],
		Unit:  unit,
	}
}

// Percentile calculates the p-th percentile from a pre-sorted slice of float64 values
// using linear interpolation between closest ranks.
//
// p must be in the range [0, 100]. If sorted is empty, 0 is returned.
func Percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}

	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}
	fraction := index - float64(lower)
	return sorted[lower] + fraction*(sorted[upper]-sorted[lower])
}

// FormatDurationHours converts a duration in seconds into human-readable compact time units.
// Output scales dynamically across seconds ("45s"), minutes ("12.5m"), hours ("3.2h"), and days ("4.5d").
func FormatDurationHours(seconds float64) string {
	if seconds < 0 {
		return "0s"
	}
	d := time.Duration(seconds * float64(time.Second))
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	days := d.Hours() / 24.0
	return fmt.Sprintf("%.1fd", days)
}

// FormatDurationDecimalHours converts seconds into hours rounded to two decimal places (e.g., 5400s -> 1.50h).
func FormatDurationDecimalHours(seconds float64) float64 {
	return math.Round((seconds/3600.0)*100) / 100
}

// DetermineSizeCategory classifies a pull request into standard T-shirt size tiers
// based on total lines modified (additions + deletions):
//   - "S (<100)" for changes under 100 lines
//   - "M (100-499)" for changes between 100 and 499 lines
//   - "L (500-999)" for changes between 500 and 999 lines
//   - "XL (1000+)" for large changes of 1,000 lines or more
func DetermineSizeCategory(additions, deletions int) string {
	total := additions + deletions
	switch {
	case total < 100:
		return "S (<100)"
	case total < 500:
		return "M (100-499)"
	case total < 1000:
		return "L (500-999)"
	default:
		return "XL (1000+)"
	}
}
