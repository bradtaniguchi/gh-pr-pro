package cmd

import (
	"github.com/spf13/cobra"
)

var codeCmd = &cobra.Command{
	Use:   "code [subcommand]",
	Short: "Code change size, commits, and complexity metrics",
	Long: `Commands to analyze pull request size, blast radius, commit topology, and code modifications:
  • size    - Lines added/deleted, total diff size, files changed, and S/M/L/XL sizing
  • commits - Commit count per PR and pushing patterns`,
	Example: `  # Breakdown PR size distribution grouped by author
  gh pr-pro code size --past 90d --group-by author

  # Commit count distribution per PR
  gh pr-pro code commits --past 60d`,
}

var codeSizeCmd = &cobra.Command{
	Use:   "size",
	Short: "Analyze line additions, deletions, modified files, and S/M/L/XL sizing",
	Long: `Measures PR change volume including additions, deletions, and changed files.
Categorizes PRs into size buckets: XS (1-9), S (10-99), M (100-499), L (500-999), and XL (1000+ lines).`,
	Example: `  # Size distribution for the past 6 months grouped by month
  gh pr-pro code size --past 6m --group-by month

  # Filter out large generated PRs over 2,000 lines
  gh pr-pro code size --max-lines 2000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "code", "size")
	},
}

var codeCommitsCmd = &cobra.Command{
	Use:   "commits",
	Short: "Analyze commit count per PR and pushing patterns",
	Long:  `Measures the number of commits per pull request to help assess PR atomicity and development habits.`,
	Example: `  # Commits per PR over the past 90 days
  gh pr-pro code commits --past 90d`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "code", "commits")
	},
}

func init() {
	codeCmd.AddCommand(codeSizeCmd)
	codeCmd.AddCommand(codeCommitsCmd)
}
