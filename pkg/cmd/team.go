package cmd

import (
	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:   "team [subcommand]",
	Short: "Team throughput, review workload, and collaboration flow",
	Long: `Commands to analyze team velocity, reviewer workload distribution, and review compliance:
  • throughput - PR velocity (opened vs merged vs closed) and net backlog accumulation
  • reviews    - Review workload distribution and responsiveness by reviewer
  • unreviewed - Self-merged PRs and PRs merged without external peer reviews`,
	Example: `  # PR throughput and backlog trends over the past year
  gh pr-pro team throughput --past 1y --group-by month

  # Review workload balance across team members
  gh pr-pro team reviews --past 90d

  # Unreviewed / self-merged PR bypass rate
  gh pr-pro team unreviewed --past 6m`,
}

var teamThroughputCmd = &cobra.Command{
	Use:   "throughput",
	Short: "Analyze PR velocity (opened vs merged vs closed) and backlog delta",
	Long: `Measures PR inflow and outflow over time, showing opened, merged, and closed PR counts
along with the net backlog delta (opened - (merged + closed)).`,
	Example: `  # Monthly throughput and backlog delta for the last year
  gh pr-pro team throughput --past 1y --group-by month`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "team", "throughput")
	},
}

var teamReviewsCmd = &cobra.Command{
	Use:   "reviews",
	Short: "Analyze review workload distribution and responsiveness by reviewer",
	Long: `Breaks down review load across team reviewers, showing total reviews submitted,
approval vs changes-requested ratio, and individual turnaround response times.`,
	Example: `  # Review workload over the last 90 days in CSV format
  gh pr-pro team reviews --past 90d --csv`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := ParseFilterOptions(cmd)
		if err != nil {
			return err
		}
		if opts.GroupBy == "" || opts.GroupBy == "none" {
			opts.GroupBy = "reviewer"
		}
		return RunMetricCommandWithOpts("time", "review", opts)
	},
}

var teamUnreviewedCmd = &cobra.Command{
	Use:   "unreviewed",
	Short: "Analyze self-merged PRs and PRs merged without external peer review",
	Long:  `Tracks the percentage of merged PRs that bypassed peer review (0 external reviews or self-merged).`,
	Example: `  # Unreviewed PR percentage over the past 6 months
  gh pr-pro team unreviewed --past 6m`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "team", "unreviewed")
	},
}

func init() {
	teamCmd.AddCommand(teamThroughputCmd)
	teamCmd.AddCommand(teamReviewsCmd)
	teamCmd.AddCommand(teamUnreviewedCmd)
}
