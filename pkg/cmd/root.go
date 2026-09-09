// Package cmd defines the Cobra CLI command tree, flags, argument parsing,
// and coordination logic for the gh-pr-pro GitHub CLI extension.
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brad/gh-pr-pro/pkg/api"
	"github.com/brad/gh-pr-pro/pkg/cache"
	"github.com/brad/gh-pr-pro/pkg/metrics"
	"github.com/brad/gh-pr-pro/pkg/output"
	"github.com/spf13/cobra"
)

var (
	flagRepo            string
	flagPast            string
	flagSince           string
	flagUntil           string
	flagFormat          string
	flagJSON            bool
	flagCSV             bool
	flagTSV             bool
	flagMarkdown        bool
	flagGroupBy         string
	flagAuthor          string
	flagReviewer        string
	flagLabel           string
	flagBase            string
	flagState           string
	flagPercentiles     bool
	flagDetailed        bool
	flagNoCache         bool
	flagSearch          string
	flagDraft           string
	flagReviewState     string
	flagAssignee        string
	flagReviewRequested string
	flagHead            string
	flagMilestone       string
	flagChecks          string
	flagMinLines        int
	flagMaxLines        int
	flagMinFiles        int
	flagMaxFiles        int
	flagConflicts       string
	flagVerbose         bool
	flagPageSize        int
)

// RootCmd represents the base Cobra command for the gh-pr-pro CLI extension.
// It defines all global persistent flags for filtering, time ranges, and serialization formats.
var RootCmd = &cobra.Command{
	Use:   "gh-pr-pro [command]",
	Short: "Advanced PR Analytics and Lifecycle Metrics for GitHub CLI",
	Long: `gh-pr-pro is a metric-first extension for GitHub CLI providing deep historical
analytics, cycle times, review turnaround, CI stability, team velocity, and multi-format data export.

It calculates statistical percentiles (p50, p75, p90, p95, p99), distributions, and trends across:
  • time     - Lifecycle stages (merge cycle time, TTFR review latency, draft duration, pickup queue, idle time)
  • quality  - CI/CD check suite pass/fail rates, review rework cycles, merge conflicts, and reverts
  • code     - Lines changed (additions/deletions), files modified, S/M/L/XL sizing, and commit topology
  • team     - PR velocity throughput, reviewer workload distribution, and unreviewed PR bypass rates
  • overview - Multi-domain executive scorecard
  • export   - Raw, flattened per-PR records streaming for data pipelines (CSV/JSON)
  • cache    - Inspect, list, and clean local repository cache files`,
	Example: `  # Analyze time to merge over the last 90 days grouped by month
  gh pr-pro time merge --past 90d --group-by month

  # Inspect reviewer turnaround times for a specific reviewer
  gh pr-pro time review --past 30d --reviewer @octocat

  # Find CI failure rates and retry frequency on the main branch
  gh pr-pro quality ci --past 60d --base main --checks failure

  # Search PRs matching keywords with line size bounds
  gh pr-pro code size --search "refactor" --min-lines 100 --max-lines 1000

  # Export raw PR dataset for downstream analysis in CSV
  gh pr-pro export --since 2025-01-01 --until 2025-12-31 --csv > prs_2025.csv

  # Manage and inspect local cache
  gh pr-pro cache list`,
}

// Execute adds all child commands to the root command, parses os.Args, and runs the targeted subcommand.
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	// Global Repository & Diagnostic Flags (Persistent on RootCmd)
	RootCmd.PersistentFlags().StringVarP(&flagRepo, "repo", "R", "", "Target repository in '[HOST/]OWNER/REPO' format (defaults to current repository)")
	RootCmd.PersistentFlags().BoolVar(&flagNoCache, "no-cache", false, "Bypass local disk cache (~/.cache/gh-pr-pro/) and force a complete fresh fetch from GitHub API")
	RootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Print verbose progress, timestamped network latency, and cache activity to stderr")
	RootCmd.PersistentFlags().IntVar(&flagPageSize, "page-size", 100, "GraphQL page size for fetching pull requests (1-100, default: 100)")

	// Attach scoped time, filter, and output flags to metric domain command roots
	metricRoots := []*cobra.Command{
		timeCmd,
		qualityCmd,
		codeCmd,
		teamCmd,
		overviewCmd,
		exportCmd,
	}

	for _, cmd := range metricRoots {
		attachTimeFlags(cmd)
		attachFilterFlags(cmd)
		attachOutputFlags(cmd)
	}

	// Register domains and commands
	RootCmd.AddCommand(timeCmd)
	RootCmd.AddCommand(qualityCmd)
	RootCmd.AddCommand(codeCmd)
	RootCmd.AddCommand(teamCmd)
	RootCmd.AddCommand(overviewCmd)
	RootCmd.AddCommand(exportCmd)
	RootCmd.AddCommand(cacheCmd)
}

// attachTimeFlags registers historical time range and date boundary flags on a command.
func attachTimeFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagPast, "past", "p", "30d", "Relative historical time range to analyze from now (e.g., '7d', '30d', '6m', '1y', '3y')")
	cmd.PersistentFlags().StringVar(&flagSince, "since", "", "Filter PRs created on or after specific date in 'YYYY-MM-DD' format (e.g., '2025-01-01')")
	cmd.PersistentFlags().StringVar(&flagUntil, "until", "", "Filter PRs created on or before specific date in 'YYYY-MM-DD' format (e.g., '2025-12-31')")
}

// attachFilterFlags registers PR lifecycle, author, review, label, and sizing filter flags on a command.
func attachFilterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagSearch, "search", "S", "", "Search query to filter PRs containing matching text in title, description body, or head branch")
	cmd.PersistentFlags().StringVarP(&flagAuthor, "author", "a", "", "Filter PRs created by specific author username (e.g., 'octocat' or '@octocat')")
	cmd.PersistentFlags().StringVarP(&flagReviewer, "reviewer", "r", "", "Filter PRs reviewed by specific user who submitted a review (e.g., 'mona' or '@mona')")
	cmd.PersistentFlags().StringVar(&flagAssignee, "assignee", "", "Filter PRs assigned to specific user (e.g., 'octocat' or '@octocat')")
	cmd.PersistentFlags().StringVar(&flagReviewRequested, "review-requested", "", "Filter PRs with a pending review request for specific user (e.g., 'octocat' or '@octocat')")
	cmd.PersistentFlags().StringVarP(&flagState, "state", "s", "all", "Filter PRs by lifecycle state: 'all' (default), 'merged', 'open', or 'closed'")
	cmd.PersistentFlags().StringVar(&flagDraft, "draft", "all", "Filter PRs by draft status: 'all' (default), 'true' (drafts only), or 'false' (ready for review only)")
	cmd.PersistentFlags().StringVar(&flagReviewState, "review-state", "all", "Filter PRs by review verdict: 'all' (default), 'approved', 'changes_requested', 'commented', or 'none' (unreviewed)")
	cmd.PersistentFlags().StringVarP(&flagLabel, "label", "l", "", "Filter PRs matching comma-separated labels (AND matching, e.g., 'bug,frontend')")
	cmd.PersistentFlags().StringVarP(&flagBase, "base", "b", "", "Filter PRs targeting a specific base branch (e.g., 'main', 'master', 'release/v1')")
	cmd.PersistentFlags().StringVar(&flagHead, "head", "", "Filter PRs originating from a specific source/head branch or pattern (e.g., 'feature/auth')")
	cmd.PersistentFlags().StringVar(&flagMilestone, "milestone", "", "Filter PRs assigned to a specific milestone title (e.g., 'v2.0', 'Sprint 42')")
	cmd.PersistentFlags().StringVar(&flagChecks, "checks", "all", "Filter PRs by latest CI/CD status: 'all' (default), 'success' (passed checks), 'failure' (failed checks), or 'pending'")
	cmd.PersistentFlags().StringVar(&flagConflicts, "has-conflicts", "all", "Filter PRs by git merge conflict status: 'all' (default), 'true' (conflicts present), or 'false' (clean)")
	cmd.PersistentFlags().IntVar(&flagMinLines, "min-lines", 0, "Filter PRs with total diff size (additions + deletions) greater than or equal to N lines")
	cmd.PersistentFlags().IntVar(&flagMaxLines, "max-lines", 0, "Filter PRs with total diff size (additions + deletions) less than or equal to N lines")
	cmd.PersistentFlags().IntVar(&flagMinFiles, "min-files", 0, "Filter PRs modifying greater than or equal to N files")
	cmd.PersistentFlags().IntVar(&flagMaxFiles, "max-files", 0, "Filter PRs modifying less than or equal to N files")
}

// attachOutputFlags registers serialization, formatting, and aggregation breakdown flags on a command.
func attachOutputFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagFormat, "format", "f", "text", "Output rendering format: 'text' (human-readable table), 'json', 'csv', 'tsv', or 'markdown'")
	cmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Shortcut to output results in formatted JSON")
	cmd.PersistentFlags().BoolVar(&flagCSV, "csv", false, "Shortcut to output results in comma-separated values (CSV)")
	cmd.PersistentFlags().BoolVar(&flagTSV, "tsv", false, "Shortcut to output results in tab-separated values (TSV)")
	cmd.PersistentFlags().BoolVar(&flagMarkdown, "markdown", false, "Shortcut to output results as a formatted Markdown table")
	cmd.PersistentFlags().BoolVar(&flagPercentiles, "percentiles", false, "Display extended statistical percentiles breakdown (p50, p75, p90, p95, p99) in table output")
	cmd.PersistentFlags().BoolVar(&flagDetailed, "detailed", false, "Include detailed list of individual matching PR records alongside the aggregate summary")
	cmd.PersistentFlags().StringVarP(&flagGroupBy, "group-by", "g", "none", "Aggregation breakdown dimension: 'none', 'day', 'week', 'month', 'quarter', 'year', 'author', 'reviewer', 'label', 'base', or 'size'")
}

// AddMetricFlags registers all time, filter, and output flags on the given command.
func AddMetricFlags(cmd *cobra.Command) {
	attachTimeFlags(cmd)
	attachFilterFlags(cmd)
	attachOutputFlags(cmd)
}

func getStringFlag(cmd *cobra.Command, name string) string {
	if cmd == nil {
		return ""
	}
	f := cmd.Flag(name)
	if f == nil {
		return ""
	}
	return f.Value.String()
}

func getBoolFlag(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	f := cmd.Flag(name)
	if f == nil {
		return false
	}
	val, err := strconv.ParseBool(f.Value.String())
	if err != nil {
		return false
	}
	return val
}

func getIntFlag(cmd *cobra.Command, name string) int {
	if cmd == nil {
		return 0
	}
	f := cmd.Flag(name)
	if f == nil {
		return 0
	}
	val, err := strconv.Atoi(f.Value.String())
	if err != nil {
		return 0
	}
	return val
}

// ParseFilterOptions reads bound command-line flags from the given command and converts them into a typed metrics.FilterOptions struct.
// It parses explicit time formats ("YYYY-MM-DD" for --since and --until) and computes relative
// past durations (e.g., "30d", "6m", "1y") from the current timestamp.
//
// Returns a populated FilterOptions struct or an error if date or duration flags cannot be parsed.
func ParseFilterOptions(cmd *cobra.Command) (metrics.FilterOptions, error) {
	pageSize := getIntFlag(cmd, "page-size")
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}

	format := getStringFlag(cmd, "format")
	if format == "" {
		format = "text"
	}
	if getBoolFlag(cmd, "json") {
		format = "json"
	} else if getBoolFlag(cmd, "csv") {
		format = "csv"
	} else if getBoolFlag(cmd, "tsv") {
		format = "tsv"
	} else if getBoolFlag(cmd, "markdown") {
		format = "markdown"
	}

	state := getStringFlag(cmd, "state")
	if state == "" {
		state = "all"
	}
	draft := getStringFlag(cmd, "draft")
	if draft == "" {
		draft = "all"
	}
	reviewState := getStringFlag(cmd, "review-state")
	if reviewState == "" {
		reviewState = "all"
	}
	checks := getStringFlag(cmd, "checks")
	if checks == "" {
		checks = "all"
	}
	hasConflicts := getStringFlag(cmd, "has-conflicts")
	if hasConflicts == "" {
		hasConflicts = "all"
	}
	groupBy := getStringFlag(cmd, "group-by")
	if groupBy == "" {
		groupBy = "none"
	}

	opts := metrics.FilterOptions{
		Repo:            getStringFlag(cmd, "repo"),
		Past:            getStringFlag(cmd, "past"),
		State:           state,
		Author:          getStringFlag(cmd, "author"),
		Reviewer:        getStringFlag(cmd, "reviewer"),
		Label:           getStringFlag(cmd, "label"),
		BaseBranch:      getStringFlag(cmd, "base"),
		GroupBy:         groupBy,
		Format:          format,
		Percentiles:     getBoolFlag(cmd, "percentiles"),
		Detailed:        getBoolFlag(cmd, "detailed"),
		NoCache:         getBoolFlag(cmd, "no-cache"),
		Search:          getStringFlag(cmd, "search"),
		Draft:           draft,
		ReviewState:     reviewState,
		Assignee:        getStringFlag(cmd, "assignee"),
		ReviewRequested: getStringFlag(cmd, "review-requested"),
		HeadBranch:      getStringFlag(cmd, "head"),
		Milestone:       getStringFlag(cmd, "milestone"),
		Checks:          checks,
		MinLines:        getIntFlag(cmd, "min-lines"),
		MaxLines:        getIntFlag(cmd, "max-lines"),
		MinFiles:        getIntFlag(cmd, "min-files"),
		MaxFiles:        getIntFlag(cmd, "max-files"),
		HasConflicts:    hasConflicts,
		Verbose:         getBoolFlag(cmd, "verbose"),
		PageSize:        pageSize,
	}

	sinceStr := getStringFlag(cmd, "since")
	untilStr := getStringFlag(cmd, "until")

	now := time.Now()

	if sinceStr != "" {
		t, err := time.Parse("2006-01-02", sinceStr)
		if err != nil {
			return opts, fmt.Errorf("invalid --since format (expected YYYY-MM-DD): %w", err)
		}
		opts.Since = t
		opts.HasSince = true
	}

	if untilStr != "" {
		t, err := time.Parse("2006-01-02", untilStr)
		if err != nil {
			return opts, fmt.Errorf("invalid --until format (expected YYYY-MM-DD): %w", err)
		}
		// End of the day
		opts.Until = t.Add(24*time.Hour - time.Nanosecond)
		opts.HasUntil = true
	}

	// If no explicit since was given, parse --past
	if !opts.HasSince && opts.Past != "" {
		sinceTime, err := parsePastDuration(now, opts.Past)
		if err != nil {
			return opts, err
		}
		opts.Since = sinceTime
		opts.HasSince = true
		opts.Until = now
		opts.HasUntil = true
	}

	return opts, nil
}

// parsePastDuration converts a relative duration string like "30d", "6w", "3m", "1y" into a past time.Time.
func parsePastDuration(now time.Time, past string) (time.Time, error) {
	past = strings.TrimSpace(strings.ToLower(past))
	if len(past) < 2 {
		return now.AddDate(0, -1, 0), nil // default 30d
	}

	unit := past[len(past)-1]
	numStr := past[:len(past)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return now, fmt.Errorf("invalid --past duration '%s': %w", past, err)
	}

	switch unit {
	case 'd':
		return now.AddDate(0, 0, -num), nil
	case 'w':
		return now.AddDate(0, 0, -num*7), nil
	case 'm':
		return now.AddDate(0, -num, 0), nil
	case 'y':
		return now.AddDate(-num, 0, 0), nil
	default:
		return now, fmt.Errorf("unknown time unit in --past '%s' (use d, w, m, y)", past)
	}
}

// FetchAndProcessPRs resolves the target repository, checks the local disk cache,
// issues incremental delta GraphQL queries against GitHub, enriches the PR lifecycle metrics,
// and saves the updated merged dataset back to disk.
//
// If the GitHub API is temporarily unreachable but local cached data exists, it falls back
// gracefully to the cached PR dataset.
func FetchAndProcessPRs(opts metrics.FilterOptions) ([]metrics.ProcessedPR, error) {
	logCmdVerbose := func(format string, args ...interface{}) {
		if opts.Verbose {
			ts := time.Now().Format("2006-01-02 15:04:05.000")
			msg := fmt.Sprintf(format, args...)
			fmt.Fprintf(os.Stderr, "[%s] [cmd] %s\n", ts, msg)
		}
	}

	repo, err := api.ResolveRepo(opts.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to determine repository: %w (use -R owner/repo)", err)
	}
	repoFullName := fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
	logCmdVerbose("Resolved target repository: %s", repoFullName)

	var cachedPRs []metrics.ProcessedPR
	var cacheMgr *cache.CacheManager
	var lastFetched time.Time

	if !opts.NoCache {
		mgr, err := cache.NewCacheManager()
		if err == nil {
			cacheMgr = mgr
			logCmdVerbose("Checking local disk cache for %s...", repoFullName)
			loaded, ts, err := mgr.Load(repoFullName)
			if err == nil && len(loaded) > 0 {
				cachedPRs = loaded
				lastFetched = ts
				logCmdVerbose("Cache hit: loaded %d cached PRs (last synced: %s)", len(cachedPRs), ts.Format("2006-01-02 15:04:05"))
			} else {
				logCmdVerbose("Cache miss / no existing cache entry for %s", repoFullName)
			}
		}
	} else {
		logCmdVerbose("Cache disabled via --no-cache")
	}

	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}
	client.SetVerbose(opts.Verbose)
	if opts.PageSize > 0 {
		client.SetPageSize(opts.PageSize)
	}

	// Fetch fresh PRs from GitHub GraphQL with delta sync and rate limit safety
	nodes, err := client.FetchPRsDelta(repo.Owner, repo.Name, opts.Since, lastFetched, 2000)
	if err != nil {
		// If network error but we have cache, fallback gracefully
		if len(cachedPRs) > 0 {
			logCmdVerbose("API query failed (%v). Falling back gracefully to %d cached PRs.", err, len(cachedPRs))
			return cachedPRs, nil
		}
		return nil, err
	}

	logCmdVerbose("Processing and calculating lifecycle metrics for %d freshly fetched PRs...", len(nodes))
	var freshPRs []metrics.ProcessedPR
	for _, node := range nodes {
		freshPRs = append(freshPRs, metrics.ProcessPRNode(node))
	}

	allPRs := cache.MergePRs(cachedPRs, freshPRs)
	logCmdVerbose("PR dataset merged: %d total PRs (%d from cache + %d fresh from API)", len(allPRs), len(cachedPRs), len(freshPRs))

	if cacheMgr != nil && len(freshPRs) > 0 {
		logCmdVerbose("Saving %d PR records to local cache...", len(allPRs))
		if saveErr := cacheMgr.Save(repoFullName, allPRs); saveErr != nil {
			logCmdVerbose("Warning: cache write failed: %v", saveErr)
		} else {
			logCmdVerbose("Local cache saved successfully.")
		}
	}

	return allPRs, nil
}

// RunMetricCommand parses command-line flags from the given command, fetches/enriches PRs, computes aggregations
// for the given domain and metric name, and renders the result to os.Stdout.
func RunMetricCommand(cmd *cobra.Command, domain, metric string) error {
	opts, err := ParseFilterOptions(cmd)
	if err != nil {
		return err
	}
	return RunMetricCommandWithOpts(domain, metric, opts)
}

// RunMetricCommandWithOpts executes PR fetching, filtering, statistical aggregation,
// and output formatting using explicitly provided filter options.
func RunMetricCommandWithOpts(domain, metric string, opts metrics.FilterOptions) error {
	prs, err := FetchAndProcessPRs(opts)
	if err != nil {
		return err
	}

	out := metrics.AggregateMetric(prs, domain, metric, opts)
	return output.RenderOutput(os.Stdout, out, opts.Format)
}
