// Package metrics provides PR lifecycle calculation, statistical modeling,
// multi-dimensional grouping, filtering, and aggregation routines.
package metrics

import "time"

// ReviewerAction represents a specific reviewer's activity and responsiveness on a pull request.
type ReviewerAction struct {
	// Login is the GitHub username of the reviewer.
	Login string `json:"login"`
	// State represents the review verdict: APPROVED, CHANGES_REQUESTED, COMMENTED, or DISMISSED.
	State string `json:"state"`
	// SubmittedAt is the timestamp when the review was posted.
	SubmittedAt time.Time `json:"submitted_at"`
	// ResponseTimeSeconds is the duration in seconds between when the PR became ready for review and when this review was submitted.
	ResponseTimeSeconds float64 `json:"response_time_seconds"`
}

// ProcessedPR represents a fully calculated, normalized, and enriched pull request record.
// It contains raw GitHub metadata augmented with computed cycle times, review turnaround metrics,
// CI check run summaries, and size classifications.
type ProcessedPR struct {
	// Number is the GitHub pull request number.
	Number int `json:"number"`
	// Title is the pull request title string.
	Title string `json:"title"`
	// Body is the pull request markdown description.
	Body string `json:"body,omitempty"`
	// Author is the GitHub username of the PR creator.
	Author string `json:"author"`
	// BaseBranch is the target branch into which changes are merged (e.g., "main").
	BaseBranch string `json:"base_branch"`
	// HeadBranch is the source feature branch name.
	HeadBranch string `json:"head_branch"`
	// State is the normalized PR lifecycle status: "OPEN", "MERGED", or "CLOSED".
	State string `json:"state"`
	// IsDraft indicates whether the PR is currently marked as a draft.
	IsDraft bool `json:"is_draft"`
	// CreatedAt is the timestamp when the pull request was opened.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the timestamp of the most recent activity on the pull request.
	UpdatedAt time.Time `json:"updated_at"`
	// ReadyForReviewAt is the timestamp when the PR was converted from draft to ready for review (or CreatedAt if never draft).
	ReadyForReviewAt time.Time `json:"ready_for_review_at"`
	// FirstReviewedAt is the timestamp of the earliest submitted review by an external peer reviewer.
	FirstReviewedAt *time.Time `json:"first_reviewed_at,omitempty"`
	// MergedAt is the timestamp when the PR was merged into the base branch, if merged.
	MergedAt *time.Time `json:"merged_at,omitempty"`
	// ClosedAt is the timestamp when the PR was closed (or merged), if closed.
	ClosedAt *time.Time `json:"closed_at,omitempty"`
	// Assignees is a slice of GitHub usernames assigned to work on the PR.
	Assignees []string `json:"assignees,omitempty"`
	// ReviewRequests is a slice of reviewer logins or team names requested to review the PR.
	ReviewRequests []string `json:"review_requests,omitempty"`
	// Milestone is the title of the GitHub milestone attached to the PR.
	Milestone string `json:"milestone,omitempty"`

	// Timing metrics (in seconds)

	// TimeToMergeSeconds is the total cycle time in seconds from creation to merge.
	TimeToMergeSeconds *float64 `json:"time_to_merge_seconds,omitempty"`
	// TimeToFirstReview is the duration in seconds from ready-for-review state to the first submitted review (TTFR).
	TimeToFirstReview *float64 `json:"time_to_first_review_seconds,omitempty"`
	// DraftDurationSeconds is the cumulative time in seconds the PR spent in draft status.
	DraftDurationSeconds float64 `json:"draft_duration_seconds"`
	// PickupDurationSeconds is the duration in seconds between ready-for-review and first review pickup.
	PickupDurationSeconds *float64 `json:"pickup_duration_seconds,omitempty"`
	// IdleDurationSeconds measures inactive waiting periods during the PR lifecycle.
	IdleDurationSeconds float64 `json:"idle_duration_seconds"`
	// ActiveReviewSeconds is the duration from first review submission to final merge.
	ActiveReviewSeconds *float64 `json:"active_review_seconds,omitempty"`

	// Code stats

	// Additions is the count of added lines of code.
	Additions int `json:"additions"`
	// Deletions is the count of deleted lines of code.
	Deletions int `json:"deletions"`
	// ChangedFiles is the number of files altered in the PR.
	ChangedFiles int `json:"changed_files"`
	// CommitCount is the total number of commits on the PR branch.
	CommitCount int `json:"commit_count"`
	// SizeCategory is the T-shirt size bucket (S, M, L, XL) determined by total line changes.
	SizeCategory string `json:"size_category"`

	// Quality & CI stats

	// HadCIFailure indicates if any status check or check run reported a failure/error.
	HadCIFailure bool `json:"had_ci_failure"`
	// CIFailedRuns is the count of individual failed check runs in the latest commit status rollup.
	CIFailedRuns int `json:"ci_failed_runs"`
	// CITotalRuns is the total number of check runs in the latest commit status rollup.
	CITotalRuns int `json:"ci_total_runs"`
	// TopFailingChecks lists the names of individual CI check suites that failed.
	TopFailingChecks []string `json:"top_failing_checks,omitempty"`
	// HasMergeConflicts indicates whether git reports merge conflicts with the target base branch.
	HasMergeConflicts bool `json:"has_merge_conflicts"`
	// IsReverted indicates if the PR matches revert naming patterns or revert labels.
	IsReverted bool `json:"is_reverted"`

	// Review stats

	// ReviewsCount is the total number of reviews submitted on the PR (excluding bots and self-reviews).
	ReviewsCount int `json:"reviews_count"`
	// ReviewRoundtrips is the number of CHANGES_REQUESTED review cycles before approval/merge.
	ReviewRoundtrips int `json:"review_roundtrips"`
	// IsUnreviewed indicates if the PR had zero external peer reviews submitted.
	IsUnreviewed bool `json:"is_unreviewed"`
	// Reviewers contains the detailed breakdown of each reviewer's actions and response times.
	Reviewers []ReviewerAction `json:"reviewers,omitempty"`
	// Labels is the list of GitHub label names applied to the PR.
	Labels []string `json:"labels,omitempty"`
}

// FilterOptions contains all search, filtering, grouping, and output formatting criteria
// parsed from CLI flags or configured programmatically for PR metric analysis.
type FilterOptions struct {
	// Repo specifies the target repository ("owner/name" or "HOST/owner/name").
	Repo string
	// Past specifies a relative time duration string (e.g. "30d", "6m", "1y").
	Past string
	// Since defines the lower bound timestamp for PR creation dates.
	Since time.Time
	// Until defines the upper bound timestamp for PR creation dates.
	Until time.Time
	// HasSince indicates whether an explicit since boundary is active.
	HasSince bool
	// HasUntil indicates whether an explicit until boundary is active.
	HasUntil bool
	// State filters PRs by status: "all", "open", "merged", or "closed".
	State string
	// Author filters PRs created by a specific username.
	Author string
	// Reviewer filters PRs reviewed by a specific username.
	Reviewer string
	// Label filters PRs containing comma-separated labels (AND matching).
	Label string
	// BaseBranch filters PRs targeting a specific base branch (e.g. "main").
	BaseBranch string
	// GroupBy specifies the aggregation breakdown dimension (e.g. "month", "reviewer", "size").
	GroupBy string
	// Format defines the serialization output format: "text", "json", "csv", "tsv", "markdown".
	Format string
	// Percentiles indicates whether extended percentiles (p50..p99) should be rendered.
	Percentiles bool
	// Detailed indicates whether matching individual PR records should be included.
	Detailed bool
	// NoCache forces fresh API queries, bypassing ~/.cache/gh-pr-pro.
	NoCache bool
	// Verbose outputs detailed timestamped network and execution progress logs to stderr.
	Verbose bool
	// PageSize configures the GraphQL API page size for PR fetching (1-100, default: 100).
	PageSize int

	// Search & Advanced Filter Options

	// Search is a free-text search substring matched against PR title, body, or head branch.
	Search string
	// Draft filters PRs by draft state: "all", "true", "false".
	Draft string
	// ReviewState filters PRs by review verdict: "all", "approved", "changes_requested", "commented", "none".
	ReviewState string
	// Assignee filters PRs assigned to a specific user.
	Assignee string
	// ReviewRequested filters PRs with a pending review request for a specific user.
	ReviewRequested string
	// HeadBranch filters PRs originating from a matching source branch.
	HeadBranch string
	// Milestone filters PRs belonging to a specific milestone title.
	Milestone string
	// Checks filters PRs by CI status: "all", "success", "failure", "pending".
	Checks string
	// MinLines filters PRs with total diff (additions + deletions) >= MinLines.
	MinLines int
	// MaxLines filters PRs with total diff (additions + deletions) <= MaxLines (if > 0).
	MaxLines int
	// MinFiles filters PRs modifying >= MinFiles.
	MinFiles int
	// MaxFiles filters PRs modifying <= MaxFiles (if > 0).
	MaxFiles int
	// HasConflicts filters PRs by merge conflict status: "all", "true", "false".
	HasConflicts string
}

// GroupSummary holds aggregate statistics, percentiles, and counts for a specific group of PRs.
type GroupSummary struct {
	// GroupKey is the identifier for the group (e.g., "2025-01", "@octocat", "M (100-499)").
	GroupKey string `json:"group_key"`
	// Count is the total number of PRs in this group.
	Count int `json:"count"`
	// P50 is the 50th percentile (median) metric value.
	P50 float64 `json:"p50"`
	// P75 is the 75th percentile metric value.
	P75 float64 `json:"p75"`
	// P90 is the 90th percentile metric value.
	P90 float64 `json:"p90"`
	// P95 is the 95th percentile metric value.
	P95 float64 `json:"p95"`
	// P99 is the 99th percentile metric value.
	P99 float64 `json:"p99"`
	// Mean is the arithmetic average metric value.
	Mean float64 `json:"mean"`
	// Min is the minimum recorded metric value in the group.
	Min float64 `json:"min"`
	// Max is the maximum recorded metric value in the group.
	Max float64 `json:"max"`
	// Unit is the measurement unit (e.g., "seconds", "percent", "lines", "commits", "prs").
	Unit string `json:"unit"`
	// ExtraMetrics contains domain-specific key-value metric calculations (e.g., failure rates, retry counts).
	ExtraMetrics map[string]interface{} `json:"extra_metrics,omitempty"`
}

// MetricOutput is the top-level container payload returned by metric commands and aggregation functions.
type MetricOutput struct {
	// Domain is the metric domain category (e.g., "time", "quality", "code", "team").
	Domain string `json:"domain"`
	// Metric is the specific metric name analyzed (e.g., "merge", "ci", "size", "throughput").
	Metric string `json:"metric"`
	// TimeWindow describes the queried date boundaries.
	TimeWindow TimeWindowInfo `json:"time_window"`
	// Summary contains overall aggregate metrics across all matching PRs.
	Summary GroupSummary `json:"summary"`
	// Groups contains per-group aggregate metrics when grouping is requested.
	Groups []GroupSummary `json:"groups,omitempty"`
	// DetailedPRs contains the individual filtered PR records when detailed mode is enabled.
	DetailedPRs []ProcessedPR `json:"prs,omitempty"`
	// ExtraSummary contains domain-wide summary annotations.
	ExtraSummary map[string]interface{} `json:"extra_summary,omitempty"`
}

// TimeWindowInfo describes the queried start, end, and duration parameters of an analysis run.
type TimeWindowInfo struct {
	// Since is the start timestamp of the query window.
	Since time.Time `json:"since"`
	// Until is the end timestamp of the query window.
	Until time.Time `json:"until"`
	// Past is the relative duration flag string if used (e.g., "30d").
	Past string `json:"past,omitempty"`
}
