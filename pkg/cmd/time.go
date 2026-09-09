package cmd

import (
	"github.com/spf13/cobra"
)

var timeCmd = &cobra.Command{
	Use:   "time [subcommand]",
	Short: "Lifecycle, latency, and duration metrics",
	Long: `Commands to analyze pull request timing and latency metrics across each lifecycle stage:
  • merge  - Total cycle time from PR creation/ready-for-review state to merge
  • review - Time to First Review (TTFR) and reviewer response turnaround
  • draft  - Duration PRs spend in draft mode before marked ready for review
  • pickup - Queue latency between review requests and first reviewer action
  • idle   - Inactivity and stalled waiting durations during PR lifecycle`,
	Example: `  # Analyze merge cycle time over the past 90 days grouped by month
  gh pr-pro time merge --past 90d --group-by month

  # Inspect Time to First Review (TTFR) grouped by reviewer
  gh pr-pro time review --past 30d --group-by reviewer

  # Check draft duration for PRs authored by a specific team member
  gh pr-pro time draft --past 6m --author @octocat`,
}

var timeMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Analyze time from PR creation/ready state to merge (Cycle Time)",
	Long: `Measures total cycle time from when a PR is opened (or converted from draft to ready)
until it is merged into the base branch. Calculates median (p50), p75, p90, and mean durations.`,
	Example: `  # Monthly merge cycle time trends for the last 6 months
  gh pr-pro time merge --past 6m --group-by month

  # Merge times on main branch with extended percentiles
  gh pr-pro time merge --base main --percentiles`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "time", "merge")
	},
}

var timeReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Analyze Time to First Review (TTFR) and reviewer response turnaround",
	Long: `Measures Time to First Review (TTFR) from when a PR is ready for review until the first
review submission, as well as individual reviewer response turnaround latencies.`,
	Example: `  # TTFR grouped by reviewer over the last 30 days
  gh pr-pro time review --past 30d --group-by reviewer

  # Detailed individual review turnaround in JSON format
  gh pr-pro time review --past 14d --detailed --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "time", "review")
	},
}

var timeDraftCmd = &cobra.Command{
	Use:   "draft",
	Short: "Analyze duration PRs spend in draft mode before ready for review",
	Long: `Measures how long pull requests remain in draft status before being marked as ready for review.
Helps identify whether PRs are being developed in draft mode for long periods before peer review.`,
	Example: `  # Draft duration grouped by PR size category (S/M/L/XL)
  gh pr-pro time draft --past 90d --group-by size`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "time", "draft")
	},
}

var timePickupCmd = &cobra.Command{
	Use:   "pickup",
	Short: "Analyze queue time from review request to first review action",
	Long: `Measures the queue latency between when a reviewer is formally requested on a PR
and when that reviewer submits their initial review action (approval, comments, or changes requested).`,
	Example: `  # Review pickup delay for the last 60 days
  gh pr-pro time pickup --past 60d`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "time", "pickup")
	},
}

var timeIdleCmd = &cobra.Command{
	Use:   "idle",
	Short: "Analyze inactivity and waiting time during PR lifecycle",
	Long: `Measures durations where PRs sit stalled without activity (waiting on author revisions
vs. waiting on peer reviews).`,
	Example: `  # Inactivity duration for open PRs
  gh pr-pro time idle --state open`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "time", "idle")
	},
}

func init() {
	timeCmd.AddCommand(timeMergeCmd)
	timeCmd.AddCommand(timeReviewCmd)
	timeCmd.AddCommand(timeDraftCmd)
	timeCmd.AddCommand(timePickupCmd)
	timeCmd.AddCommand(timeIdleCmd)
}
