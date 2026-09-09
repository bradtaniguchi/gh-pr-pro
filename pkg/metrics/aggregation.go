package metrics

import (
	"fmt"
	"sort"
	"strings"
)

// AggregateMetric filters the provided pull requests according to opts and computes statistical
// distributions, percentiles (p50, p75, p90, p95, p99), mean, min, max, and domain-specific extra metrics.
//
// If opts.GroupBy is specified (e.g., "month", "reviewer", "author", "size", "label"),
// the PRs are partitioned into sorted groups, computing statistics for each subgroup in addition to the overall summary.
func AggregateMetric(prs []ProcessedPR, domain, metric string, opts FilterOptions) MetricOutput {
	filtered := FilterPRs(prs, opts)

	output := MetricOutput{
		Domain: domain,
		Metric: metric,
		TimeWindow: TimeWindowInfo{
			Since: opts.Since,
			Until: opts.Until,
			Past:  opts.Past,
		},
		DetailedPRs: filtered,
	}

	if opts.GroupBy == "" || opts.GroupBy == "none" {
		output.Summary = computeMetricStats(filtered, domain, metric)
		return output
	}

	// Grouping
	groupsMap := make(map[string][]ProcessedPR)
	if strings.ToLower(opts.GroupBy) == "reviewer" {
		for _, pr := range filtered {
			if len(pr.Reviewers) == 0 {
				groupsMap["(unreviewed)"] = append(groupsMap["(unreviewed)"], pr)
			} else {
				for _, r := range pr.Reviewers {
					key := "@" + r.Login
					groupsMap[key] = append(groupsMap[key], pr)
				}
			}
		}
	} else {
		for _, pr := range filtered {
			key := getGroupKey(pr, opts.GroupBy)
			groupsMap[key] = append(groupsMap[key], pr)
		}
	}

	var groupKeys []string
	for k := range groupsMap {
		groupKeys = append(groupKeys, k)
	}
	sort.Strings(groupKeys)

	for _, k := range groupKeys {
		groupPRs := groupsMap[k]
		stat := computeMetricStats(groupPRs, domain, metric)
		stat.GroupKey = k
		output.Groups = append(output.Groups, stat)
	}

	output.Summary = computeMetricStats(filtered, domain, metric)
	return output
}

// FilterPRs filters a slice of ProcessedPRs against all active criteria specified in FilterOptions.
//
// It evaluates:
//   - Creation date ranges (Since and Until timestamps)
//   - Lifecycle state ("open", "merged", "closed")
//   - Author and Reviewer logins (with optional '@' prefix handling)
//   - Assignee and ReviewRequested logins
//   - Target base branch and source head branch
//   - Milestone title matching
//   - Free-text substring query against title, body, and head branch
//   - Draft status ("true", "false")
//   - Review verdict ("approved", "changes_requested", "commented", "none")
//   - CI build/check status ("success", "failure", "pending")
//   - Merge conflict status ("true", "false")
//   - Total line diff boundaries (MinLines, MaxLines) and file count boundaries (MinFiles, MaxFiles)
//   - Comma-separated label sets (all specified labels must match)
func FilterPRs(prs []ProcessedPR, opts FilterOptions) []ProcessedPR {

	var result []ProcessedPR
	for _, pr := range prs {
		// Time window filter on CreatedAt (or MergedAt if merged)
		refTime := pr.CreatedAt
		if !opts.Since.IsZero() && refTime.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && refTime.After(opts.Until) {
			continue
		}

		// State filter
		if opts.State != "" && opts.State != "all" {
			if strings.ToUpper(opts.State) != pr.State {
				continue
			}
		}

		// Author filter
		if opts.Author != "" {
			reqAuthor := strings.TrimPrefix(strings.ToLower(opts.Author), "@")
			if strings.ToLower(pr.Author) != reqAuthor {
				continue
			}
		}

		// Reviewer filter
		if opts.Reviewer != "" {
			reqReviewer := strings.TrimPrefix(strings.ToLower(opts.Reviewer), "@")
			found := false
			for _, r := range pr.Reviewers {
				if strings.ToLower(r.Login) == reqReviewer {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Base branch filter
		if opts.BaseBranch != "" {
			if !strings.EqualFold(pr.BaseBranch, opts.BaseBranch) {
				continue
			}
		}

		// Head branch filter
		if opts.HeadBranch != "" {
			if !strings.EqualFold(pr.HeadBranch, opts.HeadBranch) && !strings.Contains(strings.ToLower(pr.HeadBranch), strings.ToLower(opts.HeadBranch)) {
				continue
			}
		}

		// Assignee filter
		if opts.Assignee != "" {
			reqAssignee := strings.TrimPrefix(strings.ToLower(opts.Assignee), "@")
			found := false
			for _, a := range pr.Assignees {
				if strings.ToLower(a) == reqAssignee {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Review Requested filter
		if opts.ReviewRequested != "" {
			reqRR := strings.TrimPrefix(strings.ToLower(opts.ReviewRequested), "@")
			found := false
			for _, rr := range pr.ReviewRequests {
				if strings.ToLower(rr) == reqRR {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Milestone filter
		if opts.Milestone != "" {
			if !strings.EqualFold(pr.Milestone, opts.Milestone) {
				continue
			}
		}

		// Free-text Search (in title, body, or head branch)
		if opts.Search != "" {
			q := strings.ToLower(opts.Search)
			matchTitle := strings.Contains(strings.ToLower(pr.Title), q)
			matchBody := strings.Contains(strings.ToLower(pr.Body), q)
			matchHead := strings.Contains(strings.ToLower(pr.HeadBranch), q)
			if !matchTitle && !matchBody && !matchHead {
				continue
			}
		}

		// Draft filter
		if opts.Draft != "" && opts.Draft != "all" {
			dLower := strings.ToLower(opts.Draft)
			if (dLower == "true" || dLower == "yes" || dLower == "draft") && !pr.IsDraft {
				continue
			}
			if (dLower == "false" || dLower == "no" || dLower == "ready") && pr.IsDraft {
				continue
			}
		}

		// Review State filter
		if opts.ReviewState != "" && opts.ReviewState != "all" {
			rs := strings.ToLower(opts.ReviewState)
			switch rs {
			case "approved":
				hasApproved := false
				for _, r := range pr.Reviewers {
					if r.State == "APPROVED" {
						hasApproved = true
						break
					}
				}
				if !hasApproved {
					continue
				}
			case "changes_requested", "changes-requested":
				hasCR := false
				for _, r := range pr.Reviewers {
					if r.State == "CHANGES_REQUESTED" {
						hasCR = true
						break
					}
				}
				if !hasCR {
					continue
				}
			case "commented":
				hasCommented := false
				for _, r := range pr.Reviewers {
					if r.State == "COMMENTED" {
						hasCommented = true
						break
					}
				}
				if !hasCommented {
					continue
				}
			case "none", "unreviewed":
				if len(pr.Reviewers) > 0 {
					continue
				}
			}
		}

		// Checks / CI status filter
		if opts.Checks != "" && opts.Checks != "all" {
			chk := strings.ToLower(opts.Checks)
			switch chk {
			case "failure", "failed":
				if !pr.HadCIFailure {
					continue
				}
			case "success", "passed":
				if pr.HadCIFailure || pr.CITotalRuns == 0 {
					continue
				}
			case "pending":
				if pr.CITotalRuns > 0 {
					continue
				}
			}
		}

		// Merge Conflicts filter
		if opts.HasConflicts != "" && opts.HasConflicts != "all" {
			cLower := strings.ToLower(opts.HasConflicts)
			if (cLower == "true" || cLower == "yes") && !pr.HasMergeConflicts {
				continue
			}
			if (cLower == "false" || cLower == "no") && pr.HasMergeConflicts {
				continue
			}
		}

		// Min / Max Lines filter
		totalLines := pr.Additions + pr.Deletions
		if opts.MinLines > 0 && totalLines < opts.MinLines {
			continue
		}
		if opts.MaxLines > 0 && totalLines > opts.MaxLines {
			continue
		}

		// Min / Max Files filter
		if opts.MinFiles > 0 && pr.ChangedFiles < opts.MinFiles {
			continue
		}
		if opts.MaxFiles > 0 && pr.ChangedFiles > opts.MaxFiles {
			continue
		}

		// Label filter
		if opts.Label != "" {
			reqLabels := strings.Split(opts.Label, ",")
			hasAll := true
			for _, rl := range reqLabels {
				rl = strings.TrimSpace(strings.ToLower(rl))
				labelMatch := false
				for _, pl := range pr.Labels {
					if strings.ToLower(pl) == rl {
						labelMatch = true
						break
					}
				}
				if !labelMatch {
					hasAll = false
					break
				}
			}
			if !hasAll {
				continue
			}
		}

		result = append(result, pr)
	}
	return result
}

func getGroupKey(pr ProcessedPR, groupBy string) string {
	t := pr.CreatedAt
	switch strings.ToLower(groupBy) {
	case "day":
		return t.Format("2006-01-02")
	case "week":
		year, week := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case "month":
		return t.Format("2006-01")
	case "quarter":
		q := (int(t.Month())-1)/3 + 1
		return fmt.Sprintf("%d-Q%d", t.Year(), q)
	case "year":
		return fmt.Sprintf("%d", t.Year())
	case "author":
		return "@" + pr.Author
	case "base", "branch":
		return pr.BaseBranch
	case "size":
		return pr.SizeCategory
	case "label":
		if len(pr.Labels) > 0 {
			return pr.Labels[0]
		}
		return "(no-label)"
	default:
		return t.Format("2006-01")
	}
}

func computeMetricStats(prs []ProcessedPR, domain, metric string) GroupSummary {
	var values []float64
	unit := "seconds"
	extra := make(map[string]interface{})

	switch domain {
	case "time":
		switch metric {
		case "merge":
			for _, pr := range prs {
				if pr.TimeToMergeSeconds != nil {
					values = append(values, *pr.TimeToMergeSeconds)
				}
			}
		case "review":
			for _, pr := range prs {
				if pr.TimeToFirstReview != nil {
					values = append(values, *pr.TimeToFirstReview)
				}
			}
		case "draft":
			for _, pr := range prs {
				if pr.DraftDurationSeconds > 0 {
					values = append(values, pr.DraftDurationSeconds)
				}
			}
		case "pickup":
			for _, pr := range prs {
				if pr.PickupDurationSeconds != nil {
					values = append(values, *pr.PickupDurationSeconds)
				}
			}
		case "idle":
			for _, pr := range prs {
				if pr.IdleDurationSeconds > 0 {
					values = append(values, pr.IdleDurationSeconds)
				}
			}
		}

	case "quality":
		switch metric {
		case "ci":
			unit = "percent"
			total := len(prs)
			failed := 0
			totalRetries := 0
			failingCounts := make(map[string]int)
			for _, pr := range prs {
				if pr.HadCIFailure {
					failed++
				}
				if pr.CIFailedRuns > 0 {
					totalRetries += pr.CIFailedRuns
				}
				for _, name := range pr.TopFailingChecks {
					failingCounts[name]++
				}
			}
			failPct := 0.0
			if total > 0 {
				failPct = (float64(failed) / float64(total)) * 100.0
			}
			extra["failed_prs"] = failed
			extra["failed_rate_percent"] = failPct
			extra["total_retries"] = totalRetries
			extra["top_failing"] = failingCounts
			values = append(values, failPct)

		case "rework":
			unit = "revisions"
			for _, pr := range prs {
				values = append(values, float64(pr.ReviewRoundtrips))
			}
		case "conflicts":
			unit = "percent"
			conflicts := 0
			for _, pr := range prs {
				if pr.HasMergeConflicts {
					conflicts++
				}
			}
			pct := 0.0
			if len(prs) > 0 {
				pct = (float64(conflicts) / float64(len(prs))) * 100.0
			}
			extra["conflict_prs"] = conflicts
			extra["conflict_rate_percent"] = pct
			values = append(values, pct)

		case "reverts":
			unit = "percent"
			reverts := 0
			for _, pr := range prs {
				if pr.IsReverted {
					reverts++
				}
			}
			pct := 0.0
			if len(prs) > 0 {
				pct = (float64(reverts) / float64(len(prs))) * 100.0
			}
			extra["reverted_prs"] = reverts
			extra["revert_rate_percent"] = pct
			values = append(values, pct)
		}

	case "code":
		switch metric {
		case "size":
			unit = "lines"
			for _, pr := range prs {
				values = append(values, float64(pr.Additions+pr.Deletions))
			}
		case "commits":
			unit = "commits"
			for _, pr := range prs {
				values = append(values, float64(pr.CommitCount))
			}
		}

	case "team":
		switch metric {
		case "throughput":
			unit = "prs"
			merged := 0
			closed := 0
			opened := len(prs)
			for _, pr := range prs {
				if pr.State == "MERGED" {
					merged++
				} else if pr.State == "CLOSED" {
					closed++
				}
			}
			extra["opened"] = opened
			extra["merged"] = merged
			extra["closed"] = closed
			extra["backlog_delta"] = opened - (merged + closed)
			values = append(values, float64(merged))

		case "unreviewed":
			unit = "percent"
			unreviewed := 0
			for _, pr := range prs {
				if pr.IsUnreviewed && pr.State == "MERGED" {
					unreviewed++
				}
			}
			pct := 0.0
			if len(prs) > 0 {
				pct = (float64(unreviewed) / float64(len(prs))) * 100.0
			}
			extra["unreviewed_merged"] = unreviewed
			extra["unreviewed_rate_percent"] = pct
			values = append(values, pct)
		}
	}

	stat := CalculateStats(values, unit)
	stat.Count = len(prs)
	stat.ExtraMetrics = extra
	return stat
}
