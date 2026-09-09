package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/brad/gh-pr-pro/pkg/metrics"
	"github.com/brad/gh-pr-pro/pkg/output"
)

func main() {
	fmt.Println("Running output schema regression verification...")
	fmt.Println()

	failed := false

	// Test 1: Aggregate Metric CSV Schema
	if err := verifyMetricCSV(); err != nil {
		fmt.Printf("✗ Aggregate Metric CSV Schema: %v\n", err)
		failed = true
	} else {
		fmt.Println("✓ Aggregate Metric CSV Schema conforms to 11-column contract")
	}

	// Test 2: Aggregate Metric TSV Schema
	if err := verifyMetricTSV(); err != nil {
		fmt.Printf("✗ Aggregate Metric TSV Schema: %v\n", err)
		failed = true
	} else {
		fmt.Println("✓ Aggregate Metric TSV Schema conforms to tab-delimited 11-column contract")
	}

	// Test 3: Aggregate Metric JSON Schema
	if err := verifyMetricJSON(); err != nil {
		fmt.Printf("✗ Aggregate Metric JSON Schema: %v\n", err)
		failed = true
	} else {
		fmt.Println("✓ Aggregate Metric JSON Schema conforms to root and summary key contracts")
	}

	// Test 4: Markdown Table Format
	if err := verifyMetricMarkdown(); err != nil {
		fmt.Printf("✗ Markdown Table Format: %v\n", err)
		failed = true
	} else {
		fmt.Println("✓ Markdown Table Format conforms to GFM table contract")
	}

	// Test 5: Text Table Format
	if err := verifyMetricTextTable(); err != nil {
		fmt.Printf("✗ Text Table Format: %v\n", err)
		failed = true
	} else {
		fmt.Println("✓ Text Table Format renders successfully")
	}

	// Test 6: Raw Dataset Export CSV Schema (23 columns)
	if err := verifyRawExportCSV(); err != nil {
		fmt.Printf("✗ Raw Export CSV Schema: %v\n", err)
		failed = true
	} else {
		fmt.Println("✓ Raw Export CSV Schema conforms to 23-column data streamer contract")
	}

	// Test 7: Raw Dataset Export JSON Schema
	if err := verifyRawExportJSON(); err != nil {
		fmt.Printf("✗ Raw Export JSON Schema: %v\n", err)
		failed = true
	} else {
		fmt.Println("✓ Raw Export JSON Schema serializes valid ProcessedPR records")
	}

	fmt.Println()
	if failed {
		fmt.Println("Result: Output schema regression detected! Review discrepancies above.")
		os.Exit(1)
	}

	fmt.Println("Result: SUCCESS - All 7 output format schemas comply with contract specifications.")
}

func sampleMetricOutput() metrics.MetricOutput {
	return metrics.MetricOutput{
		Domain: "time",
		Metric: "merge",
		TimeWindow: metrics.TimeWindowInfo{
			Since: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Until: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			Past:  "30d",
		},
		Summary: metrics.GroupSummary{
			GroupKey: "Total",
			Count:    10,
			P50:      3600.0,
			P75:      7200.0,
			P90:      14400.0,
			P95:      18000.0,
			P99:      21600.0,
			Mean:     5400.0,
			Min:      600.0,
			Max:      25000.0,
			Unit:     "seconds",
		},
		Groups: []metrics.GroupSummary{
			{
				GroupKey: "2025-01",
				Count:    10,
				P50:      3600.0,
				P75:      7200.0,
				P90:      14400.0,
				P95:      18000.0,
				P99:      21600.0,
				Mean:     5400.0,
				Min:      600.0,
				Max:      25000.0,
				Unit:     "seconds",
			},
		},
	}
}

func samplePRs() []metrics.ProcessedPR {
	ttm := 7200.0
	ttfr := 1800.0
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	return []metrics.ProcessedPR{
		{
			Number:               101,
			Title:                "feat: add regression testing",
			Author:               "octocat",
			State:                "MERGED",
			IsDraft:              false,
			CreatedAt:            now.AddDate(0, 0, -2),
			ReadyForReviewAt:     now.AddDate(0, 0, -2),
			FirstReviewedAt:      &now,
			MergedAt:             &now,
			TimeToMergeSeconds:   &ttm,
			TimeToFirstReview:    &ttfr,
			DraftDurationSeconds: 0,
			Additions:            120,
			Deletions:            30,
			ChangedFiles:         4,
			CommitCount:          2,
			SizeCategory:         "M",
			HadCIFailure:         false,
			CIFailedRuns:         0,
			ReviewsCount:         2,
			ReviewRoundtrips:     1,
			Labels:               []string{"enhancement", "tests"},
		},
	}
}

func verifyMetricCSV() error {
	var buf bytes.Buffer
	if err := output.RenderOutput(&buf, sampleMetricOutput(), "csv"); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("csv parse error: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("expected at least 2 rows (header + data), got %d", len(records))
	}

	expectedHeader := []string{"group_key", "count", "p50", "p75", "p90", "p95", "p99", "mean", "min", "max", "unit"}
	if !reflect.DeepEqual(records[0], expectedHeader) {
		return fmt.Errorf("header mismatch:\n  expected: %v\n  got:      %v", expectedHeader, records[0])
	}

	return nil
}

func verifyMetricTSV() error {
	var buf bytes.Buffer
	if err := output.RenderOutput(&buf, sampleMetricOutput(), "tsv"); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	reader := csv.NewReader(&buf)
	reader.Comma = '\t'
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("tsv parse error: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("expected at least 2 rows, got %d", len(records))
	}

	expectedHeader := []string{"group_key", "count", "p50", "p75", "p90", "p95", "p99", "mean", "min", "max", "unit"}
	if !reflect.DeepEqual(records[0], expectedHeader) {
		return fmt.Errorf("header mismatch:\n  expected: %v\n  got:      %v", expectedHeader, records[0])
	}

	return nil
}

func verifyMetricJSON() error {
	var buf bytes.Buffer
	if err := output.RenderOutput(&buf, sampleMetricOutput(), "json"); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	var root map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &root); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}

	requiredRootKeys := []string{"domain", "metric", "time_window", "summary", "groups"}
	for _, k := range requiredRootKeys {
		if _, ok := root[k]; !ok {
			return fmt.Errorf("missing required root JSON key: %q", k)
		}
	}

	summary, ok := root["summary"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("summary is not a JSON object")
	}

	requiredSummaryKeys := []string{"group_key", "count", "p50", "p75", "p90", "p95", "p99", "mean", "min", "max", "unit"}
	for _, k := range requiredSummaryKeys {
		if _, ok := summary[k]; !ok {
			return fmt.Errorf("missing required summary JSON key: %q", k)
		}
	}

	return nil
}

func verifyMetricMarkdown() error {
	var buf bytes.Buffer
	if err := output.RenderOutput(&buf, sampleMetricOutput(), "markdown"); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	content := buf.String()
	if !strings.Contains(content, "| Group |") || !strings.Contains(content, "| Count |") {
		return fmt.Errorf("markdown table missing standard column headers")
	}
	if !strings.Contains(content, "|---|") {
		return fmt.Errorf("markdown table missing GFM separator row")
	}
	return nil
}

func verifyMetricTextTable() error {
	var buf bytes.Buffer
	if err := output.RenderOutput(&buf, sampleMetricOutput(), "text"); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}
	content := buf.String()
	if len(content) == 0 {
		return fmt.Errorf("text table produced empty output")
	}
	return nil
}

func verifyRawExportCSV() error {
	var buf bytes.Buffer
	if err := output.RenderRawExport(&buf, samplePRs(), "csv"); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("export csv parse error: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("expected at least 2 rows (header + PR record), got %d", len(records))
	}

	expectedHeader := []string{
		"number", "title", "author", "state", "is_draft",
		"created_at", "ready_for_review_at", "first_reviewed_at", "merged_at", "closed_at",
		"ttm_hours", "ttfr_hours", "draft_hours",
		"additions", "deletions", "changed_files", "commit_count", "size_category",
		"had_ci_failure", "ci_failed_runs", "reviews_count", "review_roundtrips", "labels",
	}

	if !reflect.DeepEqual(records[0], expectedHeader) {
		return fmt.Errorf("raw export header mismatch (expected %d columns, got %d):\n  expected: %v\n  got:      %v",
			len(expectedHeader), len(records[0]), expectedHeader, records[0])
	}

	return nil
}

func verifyRawExportJSON() error {
	var buf bytes.Buffer
	if err := output.RenderRawExport(&buf, samplePRs(), "json"); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	var parsed []metrics.ProcessedPR
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		return fmt.Errorf("raw export json unmarshal failed: %w", err)
	}

	if len(parsed) != 1 {
		return fmt.Errorf("expected 1 parsed PR, got %d", len(parsed))
	}
	if parsed[0].Number != 101 {
		return fmt.Errorf("expected PR number 101, got %d", parsed[0].Number)
	}

	return nil
}
