package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/brad/gh-pr-pro/pkg/metrics"
)

func TestRenderOutput(t *testing.T) {
	out := metrics.MetricOutput{
		Domain: "quality",
		Metric: "ci",
		TimeWindow: metrics.TimeWindowInfo{
			Since: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Until: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		Summary: metrics.GroupSummary{
			GroupKey: "Total",
			Count:    10,
			P50:      20.0,
			Mean:     25.0,
			Unit:     "percent",
			ExtraMetrics: map[string]interface{}{
				"top_failing": map[string]int{
					"unit-tests": 3,
				},
			},
		},
		Groups: []metrics.GroupSummary{
			{
				GroupKey: "2025-01",
				Count:    10,
				P50:      20.0,
				P75:      30.0,
				P90:      40.0,
				Mean:     25.0,
				Unit:     "percent",
			},
		},
	}

	tests := []struct {
		name           string
		format         string
		expectInOutput []string
		expectErr      bool
	}{
		{
			name:   "JSON format output",
			format: "json",
			expectInOutput: []string{
				`"domain": "quality"`,
				`"metric": "ci"`,
				`"p50": 20`,
			},
			expectErr: false,
		},
		{
			name:   "CSV format output with header and group rows",
			format: "csv",
			expectInOutput: []string{
				"group_key,count,p50",
				"2025-01,10,20.00",
			},
			expectErr: false,
		},
		{
			name:   "TSV format output with tab separators",
			format: "tsv",
			expectInOutput: []string{
				"group_key\tcount\tp50",
				"2025-01\t10\t20.00",
			},
			expectErr: false,
		},
		{
			name:   "Markdown table format",
			format: "markdown",
			expectInOutput: []string{
				"### PR Analytics: `quality ci`",
				"| **2025-01** |",
			},
			expectErr: false,
		},
		{
			name:   "Markdown shortcut md",
			format: "md",
			expectInOutput: []string{
				"### PR Analytics: `quality ci`",
			},
			expectErr: false,
		},
		{
			name:   "Human-readable text table format with extra failure metrics",
			format: "text",
			expectInOutput: []string{
				"QUALITY CI Analytics",
				"unit-tests: 3 failures",
			},
			expectErr: false,
		},
		{
			name:   "Default table format",
			format: "table",
			expectInOutput: []string{
				"QUALITY CI Analytics",
			},
			expectErr: false,
		},
		{
			name:      "Unsupported format returns descriptive error",
			format:    "xml",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := RenderOutput(&buf, out, tc.format)
			if tc.expectErr {
				if err == nil {
					t.Errorf("expected error for format %s, got nil", tc.format)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error rendering format %s: %v", tc.format, err)
			}

			res := buf.String()
			for _, expectedStr := range tc.expectInOutput {
				if !strings.Contains(res, expectedStr) {
					t.Errorf("expected output to contain %q, but got:\n%s", expectedStr, res)
				}
			}
		})
	}
}

func TestRenderRawExport(t *testing.T) {
	ttm := 7200.0
	ttfr := 1800.0
	prs := []metrics.ProcessedPR{
		{
			Number:               42,
			Title:                "fix: typo",
			Author:               "alice",
			State:                "MERGED",
			IsDraft:              false,
			CreatedAt:            time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			ReadyForReviewAt:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			TimeToMergeSeconds:   &ttm,
			TimeToFirstReview:    &ttfr,
			DraftDurationSeconds: 0,
			Additions:            10,
			Deletions:            2,
			ChangedFiles:         1,
			CommitCount:          1,
			SizeCategory:         "S (10-99)",
			HadCIFailure:         false,
			CIFailedRuns:         0,
			ReviewsCount:         1,
			ReviewRoundtrips:     0,
			Labels:               []string{"bug", "core"},
		},
	}

	tests := []struct {
		name           string
		format         string
		expectInOutput []string
	}{
		{
			name:   "CSV raw export stream",
			format: "csv",
			expectInOutput: []string{
				"number,title,author,state",
				"42,fix: typo,alice,MERGED",
			},
		},
		{
			name:   "JSON raw export stream",
			format: "json",
			expectInOutput: []string{
				`"number": 42`,
				`"title": "fix: typo"`,
				`"author": "alice"`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := RenderRawExport(&buf, prs, tc.format); err != nil {
				t.Fatalf("unexpected error rendering %s export: %v", tc.format, err)
			}
			outStr := buf.String()
			for _, expectedStr := range tc.expectInOutput {
				if !strings.Contains(outStr, expectedStr) {
					t.Errorf("expected %s export to contain %q, but got:\n%s", tc.format, expectedStr, outStr)
				}
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{name: "string shorter than maxLen", input: "short", maxLen: 10, expected: "short"},
		{name: "string equal to maxLen", input: "exact", maxLen: 5, expected: "exact"},
		{name: "string longer than maxLen", input: "superlongstring", maxLen: 6, expected: "super…"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncate(tc.input, tc.maxLen)
			if got != tc.expected {
				t.Errorf("truncate(%q, %d) = %q, expected %q", tc.input, tc.maxLen, got, tc.expected)
			}
		})
	}
}
