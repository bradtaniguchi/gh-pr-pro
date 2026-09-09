package metrics

import (
	"strings"
	"time"

	"github.com/brad/gh-pr-pro/pkg/api"
)

// ProcessPRNode converts a raw GraphQL PR node into a fully calculated and enriched ProcessedPR.
//
// It derives the complete pull request lifecycle timing:
//   - Size classification (S, M, L, XL) based on total line additions and deletions
//   - Revert detection via PR titles ("revert ...") and label annotations
//   - Cumulative draft duration and exact ReadyForReviewAt timestamps from timeline events
//   - Reviewer metrics, filtering out bots and self-reviews to compute Time to First Review (TTFR),
//     individual reviewer response times, and CHANGES_REQUESTED rework roundtrips
//   - Pickup queue latency and total Time to Merge (Cycle Time)
//   - CI status check rollup analysis to identify failed check runs and failing suite names
func ProcessPRNode(node api.GraphQLPRNode) ProcessedPR {

	pr := ProcessedPR{
		Number:            node.Number,
		Title:             node.Title,
		Body:              node.Body,
		Author:            node.Author.Login,
		BaseBranch:        node.BaseRefName,
		HeadBranch:        node.HeadRefName,
		State:             strings.ToUpper(node.State),
		IsDraft:           node.IsDraft,
		CreatedAt:         node.CreatedAt,
		UpdatedAt:         node.UpdatedAt,
		ReadyForReviewAt:  node.CreatedAt, // default if not draft
		MergedAt:          node.MergedAt,
		ClosedAt:          node.ClosedAt,
		Additions:         node.Additions,
		Deletions:         node.Deletions,
		ChangedFiles:      node.ChangedFiles,
		CommitCount:       node.Commits.TotalCount,
		HasMergeConflicts: node.Mergeable == "CONFLICTING",
	}

	if node.Milestone != nil {
		pr.Milestone = node.Milestone.Title
	}

	for _, a := range node.Assignees.Nodes {
		if a.Login != "" {
			pr.Assignees = append(pr.Assignees, a.Login)
		}
	}

	for _, rr := range node.ReviewRequests.Nodes {
		if rr.RequestedReviewer.Login != "" {
			pr.ReviewRequests = append(pr.ReviewRequests, rr.RequestedReviewer.Login)
		} else if rr.RequestedReviewer.Name != "" {
			pr.ReviewRequests = append(pr.ReviewRequests, rr.RequestedReviewer.Name)
		}
	}

	pr.SizeCategory = DetermineSizeCategory(pr.Additions, pr.Deletions)

	// Extract labels
	for _, l := range node.Labels.Nodes {
		pr.Labels = append(pr.Labels, l.Name)
		if strings.Contains(strings.ToLower(l.Name), "revert") {
			pr.IsReverted = true
		}
	}

	// Check title for revert pattern
	if strings.HasPrefix(strings.ToLower(pr.Title), "revert \"") || strings.HasPrefix(strings.ToLower(pr.Title), "revert:") {
		pr.IsReverted = true
	}

	// Calculate draft vs ready-for-review timestamps from timeline events
	var draftPeriodsTotal float64
	var lastDraftStart *time.Time

	if node.IsDraft {
		lastDraftStart = &node.CreatedAt
	}

	for _, item := range node.TimelineItems.Nodes {
		switch item.Typename {
		case "ConvertToDraftEvent":
			t := item.CreatedAt
			lastDraftStart = &t
		case "ReadyForReviewEvent":
			pr.ReadyForReviewAt = item.CreatedAt
			if lastDraftStart != nil {
				draftPeriodsTotal += item.CreatedAt.Sub(*lastDraftStart).Seconds()
				lastDraftStart = nil
			}
		}
	}

	if lastDraftStart != nil {
		endTime := time.Now()
		if node.ClosedAt != nil {
			endTime = *node.ClosedAt
		}
		draftPeriodsTotal += endTime.Sub(*lastDraftStart).Seconds()
	}
	pr.DraftDurationSeconds = draftPeriodsTotal

	// Process Reviews
	var firstReviewTime *time.Time
	var reviewActions []ReviewerAction
	roundtrips := 0

	for _, r := range node.Reviews.Nodes {
		// Ignore bot reviews if login ends in [bot]
		login := r.Author.Login
		if login == "" || strings.HasSuffix(login, "[bot]") || login == node.Author.Login {
			continue
		}

		submitted := r.SubmittedAt
		respTime := submitted.Sub(pr.ReadyForReviewAt).Seconds()
		if respTime < 0 {
			respTime = 0
		}

		reviewActions = append(reviewActions, ReviewerAction{
			Login:               login,
			State:               r.State,
			SubmittedAt:         submitted,
			ResponseTimeSeconds: respTime,
		})

		if firstReviewTime == nil || submitted.Before(*firstReviewTime) {
			firstReviewTime = &submitted
		}

		if r.State == "CHANGES_REQUESTED" {
			roundtrips++
		}
	}

	pr.Reviewers = reviewActions
	pr.ReviewsCount = len(reviewActions)
	pr.ReviewRoundtrips = roundtrips
	pr.IsUnreviewed = (len(reviewActions) == 0)

	if firstReviewTime != nil {
		pr.FirstReviewedAt = firstReviewTime
		ttfr := firstReviewTime.Sub(pr.ReadyForReviewAt).Seconds()
		if ttfr < 0 {
			ttfr = 0
		}
		pr.TimeToFirstReview = &ttfr
	}

	// Calculate Pickup duration (time until first review action or review request)
	if pr.FirstReviewedAt != nil {
		pickup := pr.FirstReviewedAt.Sub(pr.ReadyForReviewAt).Seconds()
		if pickup < 0 {
			pickup = 0
		}
		pr.PickupDurationSeconds = &pickup
	}

	// Time to Merge
	if pr.MergedAt != nil {
		ttm := pr.MergedAt.Sub(pr.CreatedAt).Seconds()
		if ttm < 0 {
			ttm = 0
		}
		pr.TimeToMergeSeconds = &ttm

		if pr.FirstReviewedAt != nil {
			activeReview := pr.MergedAt.Sub(*pr.FirstReviewedAt).Seconds()
			if activeReview < 0 {
				activeReview = 0
			}
			pr.ActiveReviewSeconds = &activeReview
		}
	}

	// Process CI Status Check Rollup from latest commit
	if len(node.Commits.Nodes) > 0 {
		commit := node.Commits.Nodes[len(node.Commits.Nodes)-1].Commit
		if commit.StatusCheckRollup != nil {
			rollup := commit.StatusCheckRollup
			contexts := rollup.Contexts.Nodes
			pr.CITotalRuns = len(contexts)
			failedCount := 0
			var failingNames []string

			for _, ctx := range contexts {
				isFailed := false
				name := ctx.Name
				if name == "" {
					name = ctx.Context
				}

				if ctx.Typename == "CheckRun" {
					if ctx.Conclusion == "FAILURE" || ctx.Conclusion == "TIMED_OUT" || ctx.Conclusion == "STARTUP_FAILURE" {
						isFailed = true
					}
				} else if ctx.Typename == "StatusContext" {
					if ctx.State == "FAILURE" || ctx.State == "ERROR" {
						isFailed = true
					}
				}

				if isFailed {
					failedCount++
					if name != "" {
						failingNames = append(failingNames, name)
					}
				}
			}

			pr.CIFailedRuns = failedCount
			pr.HadCIFailure = (failedCount > 0) || (rollup.State == "FAILURE" || rollup.State == "ERROR")
			pr.TopFailingChecks = failingNames
		}
	}

	return pr
}
