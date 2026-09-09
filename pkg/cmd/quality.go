package cmd

import (
	"github.com/spf13/cobra"
)

var qualityCmd = &cobra.Command{
	Use:   "quality [subcommand]",
	Short: "CI/CD health, review rework, conflicts, and revert metrics",
	Long: `Commands to analyze pull request quality, build stability, merge conflicts, and review churn:
  • ci        - CI/CD check suite pass/fail health, rerun/retry count, and top failing check runs
  • rework    - Review roundtrips, changes-requested revisions, and post-review push churn
  • conflicts - Frequency of merge conflicts with target base branches
  • reverts   - Post-merge revert rate and time elapsed from merge to revert`,
	Example: `  # Analyze CI failure rates and retry counts over the past 60 days
  gh pr-pro quality ci --past 60d --base main

  # Review rework cycles grouped by PR author
  gh pr-pro quality rework --past 90d --group-by author

  # Revert rate across PRs merged over the last year
  gh pr-pro quality reverts --past 1y`,
}

var qualityCICmd = &cobra.Command{
	Use:   "ci",
	Short: "Analyze CI/CD check suite pass/fail rates, retries, and top failing checks",
	Long: `Evaluates automated build and test pipeline stability across PRs. Computes check run failure rates,
total rerun counts, and identifies the most frequently failing check suites and test names.`,
	Example: `  # Weekly CI health and top failing test suites
  gh pr-pro quality ci --past 60d --group-by week

  # Inspect only PRs that experienced CI failures
  gh pr-pro quality ci --checks failure --detailed`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "quality", "ci")
	},
}

var qualityReworkCmd = &cobra.Command{
	Use:   "rework",
	Short: "Analyze review roundtrips and revision churn after first review",
	Long: `Measures review friction and rework cycles by tracking how many 'CHANGES_REQUESTED' review
roundtrips occur before approval and merge.`,
	Example: `  # Review roundtrips grouped by PR size category
  gh pr-pro quality rework --past 90d --group-by size`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "quality", "rework")
	},
}

var qualityConflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "Analyze frequency of merge conflicts with the base branch",
	Long:  `Measures how often pull requests experience git merge conflicts against their target base branch.`,
	Example: `  # Merge conflict rate for PRs over the past 6 months
  gh pr-pro quality conflicts --past 6m`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "quality", "conflicts")
	},
}

var qualityRevertsCmd = &cobra.Command{
	Use:   "reverts",
	Short: "Analyze post-merge reverts and defect tracking",
	Long:  `Tracks the rate of merged PRs that were subsequently reverted, helping identify defect escape rates.`,
	Example: `  # Yearly post-merge revert rate
  gh pr-pro quality reverts --past 1y --group-by quarter`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunMetricCommand(cmd, "quality", "reverts")
	},
}

func init() {
	qualityCmd.AddCommand(qualityCICmd)
	qualityCmd.AddCommand(qualityReworkCmd)
	qualityCmd.AddCommand(qualityConflictsCmd)
	qualityCmd.AddCommand(qualityRevertsCmd)
}
