package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/brad/gh-pr-pro/pkg/metrics"
	"github.com/spf13/cobra"
)

var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "All-in-one PR analytics and health scorecard",
	Long: `Displays a unified executive scorecard combining key engineering intelligence metrics
across time (cycle time, TTFR, draft latency), quality (CI failure rates, review rework),
code complexity (diff size), and team flow (throughput, unreviewed PR rate).`,
	Example: `  # 30-day executive scorecard for the current repository
  gh pr-pro overview --past 30d

  # Scorecard for a specific target repository in JSON format
  gh pr-pro overview -R cli/cli --past 90d --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := ParseFilterOptions(cmd)
		if err != nil {
			return err
		}

		prs, err := FetchAndProcessPRs(opts)
		if err != nil {
			return err
		}

		filtered := metrics.FilterPRs(prs, opts)

		timeMerge := metrics.AggregateMetric(filtered, "time", "merge", opts)
		timeReview := metrics.AggregateMetric(filtered, "time", "review", opts)
		timeDraft := metrics.AggregateMetric(filtered, "time", "draft", opts)
		qualityCI := metrics.AggregateMetric(filtered, "quality", "ci", opts)
		qualityRework := metrics.AggregateMetric(filtered, "quality", "rework", opts)
		codeSize := metrics.AggregateMetric(filtered, "code", "size", opts)
		teamFlow := metrics.AggregateMetric(filtered, "team", "throughput", opts)
		teamUnreviewed := metrics.AggregateMetric(filtered, "team", "unreviewed", opts)

		if strings.ToLower(opts.Format) == "json" {
			overviewData := map[string]interface{}{
				"total_prs":       len(filtered),
				"time_window":     timeMerge.TimeWindow,
				"merge_time":      timeMerge.Summary,
				"review_time":     timeReview.Summary,
				"draft_time":      timeDraft.Summary,
				"ci_health":       qualityCI.Summary,
				"review_rework":   qualityRework.Summary,
				"code_size":       codeSize.Summary,
				"team_throughput": teamFlow.Summary,
				"team_unreviewed": teamUnreviewed.Summary,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(overviewData)
		}

		// Text Dashboard Output
		fmt.Printf("📊 PR Analytics Overview (%s to %s)\n",
			timeMerge.TimeWindow.Since.Format("2006-01-02"),
			timeMerge.TimeWindow.Until.Format("2006-01-02"),
		)
		fmt.Println(strings.Repeat("━", 65))
		fmt.Printf("Total PRs Analyzed: %d\n", len(filtered))
		fmt.Println()

		fmt.Println("⏱️  LIFECYCLE & TIMING (p50 / p90 / Mean)")
		fmt.Printf("  • Time to Merge (Cycle Time):   %s / %s / %s\n",
			metrics.FormatDurationHours(timeMerge.Summary.P50),
			metrics.FormatDurationHours(timeMerge.Summary.P90),
			metrics.FormatDurationHours(timeMerge.Summary.Mean),
		)
		fmt.Printf("  • Time to First Review (TTFR):  %s / %s / %s\n",
			metrics.FormatDurationHours(timeReview.Summary.P50),
			metrics.FormatDurationHours(timeReview.Summary.P90),
			metrics.FormatDurationHours(timeReview.Summary.Mean),
		)
		fmt.Printf("  • Draft Status Duration:        %s / %s / %s\n",
			metrics.FormatDurationHours(timeDraft.Summary.P50),
			metrics.FormatDurationHours(timeDraft.Summary.P90),
			metrics.FormatDurationHours(timeDraft.Summary.Mean),
		)
		fmt.Println()

		fmt.Println("🚦 QUALITY & CI HEALTH")
		ciFailPct, _ := qualityCI.Summary.ExtraMetrics["failed_rate_percent"].(float64)
		unrevPct, _ := teamUnreviewed.Summary.ExtraMetrics["unreviewed_rate_percent"].(float64)
		fmt.Printf("  • PRs with CI Failures:         %.1f%%\n", ciFailPct)
		fmt.Printf("  • Avg Review Roundtrips:        %.1f revisions\n", qualityRework.Summary.Mean)
		fmt.Printf("  • Unreviewed / Self-Merged:     %.1f%%\n", unrevPct)
		fmt.Println()

		fmt.Println("📦 CODE & COMPLEXITY")
		fmt.Printf("  • Avg Total Diff per PR:        %.0f lines\n", codeSize.Summary.Mean)
		fmt.Println(strings.Repeat("━", 65))

		return nil
	},
}
