package cmd

import (
	"os"

	"github.com/brad/gh-pr-pro/pkg/metrics"
	"github.com/brad/gh-pr-pro/pkg/output"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Stream raw, unaggregated PR records for data pipelines",
	Long: `Exports flattened, unaggregated per-PR records with all calculated timestamps, review latencies,
CI status check runs, reviewers, labels, and commit metrics. Designed for streaming into data pipelines,
DuckDB, Pandas, Snowflake, or PostgreSQL.`,
	Example: `  # Stream 1 year of raw PR records to CSV
  gh pr-pro export --past 1y --csv > pr_metrics_2025.csv

  # Export PR records for a specific date window in JSON
  gh pr-pro export --since 2025-01-01 --until 2025-06-30 --json > pr_h1.json`,
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
		format := opts.Format
		if format == "text" || format == "" {
			format = "json" // default export to json if not specified
		}

		return output.RenderRawExport(os.Stdout, filtered, format)
	},
}
