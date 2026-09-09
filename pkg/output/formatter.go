// Package output handles rendering and formatting of calculated metrics
// across multiple output targets (ANSI text tables, JSON, CSV, TSV, and Markdown).
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/brad/gh-pr-pro/pkg/metrics"
)

// RenderOutput formats and writes the provided MetricOutput to the given io.Writer.
// Supported format strings:
//   - "text", "table", "" : Formatted terminal-friendly tabular output with ANSI borders and summaries
//   - "json"              : Pretty-printed JSON object with 2-space indent
//   - "csv"               : Comma-separated values with column headers
//   - "tsv"               : Tab-separated values with column headers
//   - "markdown", "md"    : GitHub Flavored Markdown table
//
// Returns an error if an unsupported format is specified or if writing fails.
func RenderOutput(w io.Writer, out metrics.MetricOutput, format string) error {
	switch strings.ToLower(format) {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(out)

	case "csv":
		return renderCSV(w, out, ',')

	case "tsv":
		return renderCSV(w, out, '\t')

	case "markdown", "md":
		return renderMarkdown(w, out)

	case "text", "table", "":
		return renderTextTable(w, out)

	default:
		return fmt.Errorf("unsupported output format: %s (supported: text, json, csv, tsv, markdown)", format)
	}
}

func renderCSV(w io.Writer, out metrics.MetricOutput, delimiter rune) error {
	writer := csv.NewWriter(w)
	writer.Comma = delimiter
	defer writer.Flush()

	// Header
	header := []string{"group_key", "count", "p50", "p75", "p90", "p95", "p99", "mean", "min", "max", "unit"}
	if err := writer.Write(header); err != nil {
		return err
	}

	writeRow := func(s metrics.GroupSummary) error {
		p50Val, p75Val, p90Val, p95Val, p99Val, meanVal := s.P50, s.P75, s.P90, s.P95, s.P99, s.Mean
		if s.Unit == "seconds" {
			p50Val = metrics.FormatDurationDecimalHours(s.P50)
			p75Val = metrics.FormatDurationDecimalHours(s.P75)
			p90Val = metrics.FormatDurationDecimalHours(s.P90)
			p95Val = metrics.FormatDurationDecimalHours(s.P95)
			p99Val = metrics.FormatDurationDecimalHours(s.P99)
			meanVal = metrics.FormatDurationDecimalHours(s.Mean)
		}
		row := []string{
			s.GroupKey,
			fmt.Sprintf("%d", s.Count),
			fmt.Sprintf("%.2f", p50Val),
			fmt.Sprintf("%.2f", p75Val),
			fmt.Sprintf("%.2f", p90Val),
			fmt.Sprintf("%.2f", p95Val),
			fmt.Sprintf("%.2f", p99Val),
			fmt.Sprintf("%.2f", meanVal),
			fmt.Sprintf("%.2f", s.Min),
			fmt.Sprintf("%.2f", s.Max),
			s.Unit,
		}
		return writer.Write(row)
	}

	if len(out.Groups) > 0 {
		for _, g := range out.Groups {
			if err := writeRow(g); err != nil {
				return err
			}
		}
	} else {
		out.Summary.GroupKey = "Total"
		if err := writeRow(out.Summary); err != nil {
			return err
		}
	}

	return nil
}

func renderMarkdown(w io.Writer, out metrics.MetricOutput) error {
	fmt.Fprintf(w, "### PR Analytics: `%s %s`\n\n", out.Domain, out.Metric)
	fmt.Fprintf(w, "| Group | Count | p50 | p75 | p90 | Mean |\n")
	fmt.Fprintf(w, "|---|---|---|---|---|---|\n")

	formatVal := func(s float64, unit string) string {
		if unit == "seconds" {
			return metrics.FormatDurationHours(s)
		}
		if unit == "percent" {
			return fmt.Sprintf("%.1f%%", s)
		}
		return fmt.Sprintf("%.1f", s)
	}

	if len(out.Groups) > 0 {
		for _, g := range out.Groups {
			fmt.Fprintf(w, "| **%s** | %d | %s | %s | %s | %s |\n",
				g.GroupKey, g.Count,
				formatVal(g.P50, g.Unit),
				formatVal(g.P75, g.Unit),
				formatVal(g.P90, g.Unit),
				formatVal(g.Mean, g.Unit),
			)
		}
	} else {
		s := out.Summary
		fmt.Fprintf(w, "| **Total** | %d | %s | %s | %s | %s |\n",
			s.Count,
			formatVal(s.P50, s.Unit),
			formatVal(s.P75, s.Unit),
			formatVal(s.P90, s.Unit),
			formatVal(s.Mean, s.Unit),
		)
	}
	return nil
}

func renderTextTable(w io.Writer, out metrics.MetricOutput) error {
	headerTitle := fmt.Sprintf("📊 %s %s Analytics", strings.ToUpper(out.Domain), strings.ToUpper(out.Metric))
	fmt.Fprintln(w, headerTitle)
	fmt.Fprintln(w, strings.Repeat("━", 65))

	formatVal := func(s float64, unit string) string {
		if unit == "seconds" {
			return metrics.FormatDurationHours(s)
		}
		if unit == "percent" {
			return fmt.Sprintf("%.1f%%", s)
		}
		return fmt.Sprintf("%.1f", s)
	}

	if len(out.Groups) > 0 {
		fmt.Fprintf(w, "%-16s %-8s %-12s %-12s %-12s %-10s\n", "Group", "Count", "p50 (Median)", "p75", "p90", "Mean")
		fmt.Fprintln(w, strings.Repeat("─", 65))
		for _, g := range out.Groups {
			fmt.Fprintf(w, "%-16s %-8d %-12s %-12s %-12s %-10s\n",
				truncate(g.GroupKey, 15),
				g.Count,
				formatVal(g.P50, g.Unit),
				formatVal(g.P75, g.Unit),
				formatVal(g.P90, g.Unit),
				formatVal(g.Mean, g.Unit),
			)
		}
		fmt.Fprintln(w, strings.Repeat("─", 65))
	}

	s := out.Summary
	fmt.Fprintf(w, "Summary: %d PRs analyzed | p50: %s | p75: %s | p90: %s | Mean: %s\n",
		s.Count,
		formatVal(s.P50, s.Unit),
		formatVal(s.P75, s.Unit),
		formatVal(s.P90, s.Unit),
		formatVal(s.Mean, s.Unit),
	)

	// Display any extra domain metrics
	if len(s.ExtraMetrics) > 0 {
		for k, v := range s.ExtraMetrics {
			if k == "top_failing" {
				m, ok := v.(map[string]int)
				if ok && len(m) > 0 {
					fmt.Fprintln(w, "\nTop Failing Check Suites:")
					for checkName, count := range m {
						fmt.Fprintf(w, "  • %s: %d failures\n", checkName, count)
					}
				}
			}
		}
	}

	return nil
}

func truncate(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen-1] + "…"
}

// RenderRawExport serializes unaggregated, individual ProcessedPR records to the given io.Writer.
//
// It supports two export formats:
//   - "csv": Flattened comma-separated table containing all PR timestamps, cycle times in decimal hours,
//     CI failure booleans, review counts, and semicolon-delimited label lists.
//   - "json" (or any default): Pretty-printed JSON array containing the full slice of ProcessedPR records.
//
// Useful for piping raw PR telemetry into data pipelines, pandas, DuckDB, or data warehouses.
func RenderRawExport(w io.Writer, prs []metrics.ProcessedPR, format string) error {

	if strings.ToLower(format) == "csv" {
		writer := csv.NewWriter(w)
		defer writer.Flush()

		header := []string{
			"number", "title", "author", "state", "is_draft",
			"created_at", "ready_for_review_at", "first_reviewed_at", "merged_at", "closed_at",
			"ttm_hours", "ttfr_hours", "draft_hours",
			"additions", "deletions", "changed_files", "commit_count", "size_category",
			"had_ci_failure", "ci_failed_runs", "reviews_count", "review_roundtrips", "labels",
		}
		if err := writer.Write(header); err != nil {
			return err
		}

		for _, pr := range prs {
			ttmStr, ttfrStr, draftStr := "", "", ""
			if pr.TimeToMergeSeconds != nil {
				ttmStr = fmt.Sprintf("%.2f", metrics.FormatDurationDecimalHours(*pr.TimeToMergeSeconds))
			}
			if pr.TimeToFirstReview != nil {
				ttfrStr = fmt.Sprintf("%.2f", metrics.FormatDurationDecimalHours(*pr.TimeToFirstReview))
			}
			if pr.DraftDurationSeconds > 0 {
				draftStr = fmt.Sprintf("%.2f", metrics.FormatDurationDecimalHours(pr.DraftDurationSeconds))
			}

			firstRevStr, mergedStr, closedStr := "", "", ""
			if pr.FirstReviewedAt != nil {
				firstRevStr = pr.FirstReviewedAt.Format("2006-01-02T15:04:05Z")
			}
			if pr.MergedAt != nil {
				mergedStr = pr.MergedAt.Format("2006-01-02T15:04:05Z")
			}
			if pr.ClosedAt != nil {
				closedStr = pr.ClosedAt.Format("2006-01-02T15:04:05Z")
			}

			row := []string{
				fmt.Sprintf("%d", pr.Number),
				pr.Title,
				pr.Author,
				pr.State,
				fmt.Sprintf("%t", pr.IsDraft),
				pr.CreatedAt.Format("2006-01-02T15:04:05Z"),
				pr.ReadyForReviewAt.Format("2006-01-02T15:04:05Z"),
				firstRevStr,
				mergedStr,
				closedStr,
				ttmStr,
				ttfrStr,
				draftStr,
				fmt.Sprintf("%d", pr.Additions),
				fmt.Sprintf("%d", pr.Deletions),
				fmt.Sprintf("%d", pr.ChangedFiles),
				fmt.Sprintf("%d", pr.CommitCount),
				pr.SizeCategory,
				fmt.Sprintf("%t", pr.HadCIFailure),
				fmt.Sprintf("%d", pr.CIFailedRuns),
				fmt.Sprintf("%d", pr.ReviewsCount),
				fmt.Sprintf("%d", pr.ReviewRoundtrips),
				strings.Join(pr.Labels, ";"),
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
		return nil
	}

	// Default JSON
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(prs)
}
