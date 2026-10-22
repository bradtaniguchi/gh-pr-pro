package metrics

import (
	"testing"
	"time"

	"github.com/brad/gh-pr-pro/pkg/api"
)

func TestProcessPRNode(t *testing.T) {
	created := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	reviewed := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	merged := time.Date(2025, 1, 1, 18, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		buildNode  func() api.GraphQLPRNode
		validatePR func(t *testing.T, pr ProcessedPR)
	}{
		{
			name: "merged PR with approved review, assignees, review request, and milestone",
			buildNode: func() api.GraphQLPRNode {
				node := api.GraphQLPRNode{
					Number:       101,
					Title:        "feat: add auth",
					Body:         "Adds OAuth2 authentication",
					State:        "MERGED",
					CreatedAt:    created,
					MergedAt:     &merged,
					IsDraft:      false,
					BaseRefName:  "main",
					HeadRefName:  "feat/auth",
					Additions:    120,
					Deletions:    30,
					ChangedFiles: 5,
				}
				node.Author.Login = "developer"
				node.Milestone = &struct {
					Title string `json:"title"`
				}{Title: "v1.0"}

				node.Assignees.Nodes = append(node.Assignees.Nodes, struct {
					Login string `json:"login"`
				}{Login: "developer"})

				node.ReviewRequests.Nodes = append(node.ReviewRequests.Nodes, struct {
					RequestedReviewer struct {
						Typename string `json:"__typename"`
						Login    string `json:"login,omitempty"`
						Name     string `json:"name,omitempty"`
					} `json:"requestedReviewer"`
				}{
					RequestedReviewer: struct {
						Typename string `json:"__typename"`
						Login    string `json:"login,omitempty"`
						Name     string `json:"name,omitempty"`
					}{
						Typename: "User",
						Login:    "lead-dev",
					},
				})

				node.Reviews.Nodes = append(node.Reviews.Nodes, struct {
					Author struct {
						Login string `json:"login"`
					} `json:"author"`
					State       string    `json:"state"`
					SubmittedAt time.Time `json:"submittedAt"`
					Body        string    `json:"body"`
				}{
					Author: struct {
						Login string `json:"login"`
					}{Login: "reviewer1"},
					State:       "APPROVED",
					SubmittedAt: reviewed,
					Body:        "LGTM",
				})

				return node
			},
			validatePR: func(t *testing.T, pr ProcessedPR) {
				if pr.Number != 101 {
					t.Errorf("expected number 101, got %d", pr.Number)
				}
				if pr.Body != "Adds OAuth2 authentication" {
					t.Errorf("expected body to match, got %s", pr.Body)
				}
				if pr.Milestone != "v1.0" {
					t.Errorf("expected milestone v1.0, got %s", pr.Milestone)
				}
				if len(pr.Assignees) != 1 || pr.Assignees[0] != "developer" {
					t.Errorf("expected assignee developer, got %v", pr.Assignees)
				}
				if len(pr.ReviewRequests) != 1 || pr.ReviewRequests[0] != "lead-dev" {
					t.Errorf("expected review request lead-dev, got %v", pr.ReviewRequests)
				}
				if pr.SizeCategory != "M (100-499)" {
					t.Errorf("expected size M, got %s", pr.SizeCategory)
				}
				if pr.TimeToFirstReview == nil || *pr.TimeToFirstReview != 7200 {
					t.Errorf("expected TTFR 7200s, got %v", pr.TimeToFirstReview)
				}
				if pr.TimeToMergeSeconds == nil || *pr.TimeToMergeSeconds != 28800 {
					t.Errorf("expected TTM 28800s, got %v", pr.TimeToMergeSeconds)
				}
				if pr.IsUnreviewed {
					t.Errorf("expected PR to not be unreviewed")
				}
			},
		},
		{
			name: "unreviewed closed PR with merge conflicts and revert pattern",
			buildNode: func() api.GraphQLPRNode {
				node := api.GraphQLPRNode{
					Number:       102,
					Title:        "revert: \"feat: bad commit\"",
					State:        "CLOSED",
					CreatedAt:    created,
					Mergeable:    "CONFLICTING",
					Additions:    10,
					Deletions:    5,
					ChangedFiles: 1,
				}
				node.Author.Login = "developer"
				return node
			},
			validatePR: func(t *testing.T, pr ProcessedPR) {
				if !pr.IsUnreviewed {
					t.Errorf("expected PR to be unreviewed")
				}
				if !pr.HasMergeConflicts {
					t.Errorf("expected merge conflicts to be true")
				}
				if !pr.IsReverted {
					t.Errorf("expected PR to be detected as reverted")
				}
			},
		},
		{
			name: "draft PR with timeline conversion duration calculation",
			buildNode: func() api.GraphQLPRNode {
				readyAt := created.Add(2 * time.Hour)
				node := api.GraphQLPRNode{
					Number:    103,
					Title:     "feat: draft pr",
					State:     "OPEN",
					CreatedAt: created,
					IsDraft:   false,
				}
				node.TimelineItems.Nodes = append(node.TimelineItems.Nodes,
					struct {
						Typename  string    `json:"__typename"`
						CreatedAt time.Time `json:"createdAt"`
						Actor     struct {
							Login string `json:"login"`
						} `json:"actor"`
					}{
						Typename:  "ConvertToDraftEvent",
						CreatedAt: created,
					},
					struct {
						Typename  string    `json:"__typename"`
						CreatedAt time.Time `json:"createdAt"`
						Actor     struct {
							Login string `json:"login"`
						} `json:"actor"`
					}{
						Typename:  "ReadyForReviewEvent",
						CreatedAt: readyAt,
					},
				)
				return node
			},
			validatePR: func(t *testing.T, pr ProcessedPR) {
				if pr.DraftDurationSeconds != 7200 {
					t.Errorf("expected draft duration 7200s, got %v", pr.DraftDurationSeconds)
				}
			},
		},
		{
			name: "PR with check runs: timings, queue delay, compute sum, slowest bottleneck check",
			buildNode: func() api.GraphQLPRNode {
				node := api.GraphQLPRNode{
					Number:    104,
					Title:     "feat: ci timing check",
					State:     "OPEN",
					CreatedAt: created,
				}
				commitDate := created.Add(10 * time.Minute)
				job1Start := commitDate.Add(2 * time.Minute)
				job1End := job1Start.Add(5 * time.Minute) // 300s
				job2Start := commitDate.Add(3 * time.Minute)
				job2End := job2Start.Add(10 * time.Minute) // 600s

				type checkNode = struct {
					Typename    string     `json:"__typename"`
					Name        string     `json:"name,omitempty"`
					Conclusion  string     `json:"conclusion,omitempty"`
					Status      string     `json:"status,omitempty"`
					StartedAt   *time.Time `json:"startedAt,omitempty"`
					CompletedAt *time.Time `json:"completedAt,omitempty"`
					Context     string     `json:"context,omitempty"`
					State       string     `json:"state,omitempty"`
					CreatedAt   *time.Time `json:"createdAt,omitempty"`
				}

				commitNode := struct {
					Commit struct {
						CommittedDate     time.Time `json:"committedDate"`
						StatusCheckRollup *struct {
							State    string `json:"state"`
							Contexts struct {
								TotalCount int         `json:"totalCount"`
								Nodes      []checkNode `json:"nodes"`
							} `json:"contexts"`
						} `json:"statusCheckRollup"`
					} `json:"commit"`
				}{}
				commitNode.Commit.CommittedDate = commitDate
				commitNode.Commit.StatusCheckRollup = &struct {
					State    string `json:"state"`
					Contexts struct {
						TotalCount int         `json:"totalCount"`
						Nodes      []checkNode `json:"nodes"`
					} `json:"contexts"`
				}{
					State: "SUCCESS",
				}
				commitNode.Commit.StatusCheckRollup.Contexts.Nodes = []checkNode{
					{
						Typename:    "CheckRun",
						Name:        "lint",
						Conclusion:  "SUCCESS",
						Status:      "COMPLETED",
						StartedAt:   &job1Start,
						CompletedAt: &job1End,
					},
					{
						Typename:    "CheckRun",
						Name:        "e2e-tests",
						Conclusion:  "SUCCESS",
						Status:      "COMPLETED",
						StartedAt:   &job2Start,
						CompletedAt: &job2End,
					},
				}
				node.Commits.Nodes = append(node.Commits.Nodes, commitNode)
				return node
			},
			validatePR: func(t *testing.T, pr ProcessedPR) {
				if pr.CIStatus != "SUCCESS" {
					t.Errorf("expected CIStatus SUCCESS, got %s", pr.CIStatus)
				}
				if pr.CITotalRuns != 2 {
					t.Errorf("expected 2 runs, got %d", pr.CITotalRuns)
				}
				if pr.CIDurationSeconds == nil || *pr.CIDurationSeconds != 660 { // job2End (13m) - job1Start (2m) = 11m = 660s
					t.Errorf("expected wall-clock duration 660s, got %v", pr.CIDurationSeconds)
				}
				if pr.CIQueueSeconds == nil || *pr.CIQueueSeconds != 120 { // job1Start - commitDate = 2m = 120s
					t.Errorf("expected queue seconds 120s, got %v", pr.CIQueueSeconds)
				}
				if pr.CITotalComputeSeconds == nil || *pr.CITotalComputeSeconds != 900 { // 300 + 600 = 900s
					t.Errorf("expected compute seconds 900s, got %v", pr.CITotalComputeSeconds)
				}
				if pr.CISlowestCheckName != "e2e-tests" {
					t.Errorf("expected slowest check e2e-tests, got %s", pr.CISlowestCheckName)
				}
				if pr.CISlowestCheckSeconds == nil || *pr.CISlowestCheckSeconds != 600 {
					t.Errorf("expected slowest duration 600s, got %v", pr.CISlowestCheckSeconds)
				}
			},
		},
		{
			name: "PR with TIMED_OUT check run and CANCELLED check run",
			buildNode: func() api.GraphQLPRNode {
				node := api.GraphQLPRNode{
					Number:    105,
					Title:     "feat: timeout test",
					State:     "OPEN",
					CreatedAt: created,
				}
				type checkNode = struct {
					Typename    string     `json:"__typename"`
					Name        string     `json:"name,omitempty"`
					Conclusion  string     `json:"conclusion,omitempty"`
					Status      string     `json:"status,omitempty"`
					StartedAt   *time.Time `json:"startedAt,omitempty"`
					CompletedAt *time.Time `json:"completedAt,omitempty"`
					Context     string     `json:"context,omitempty"`
					State       string     `json:"state,omitempty"`
					CreatedAt   *time.Time `json:"createdAt,omitempty"`
				}

				commitNode := struct {
					Commit struct {
						CommittedDate     time.Time `json:"committedDate"`
						StatusCheckRollup *struct {
							State    string `json:"state"`
							Contexts struct {
								TotalCount int         `json:"totalCount"`
								Nodes      []checkNode `json:"nodes"`
							} `json:"contexts"`
						} `json:"statusCheckRollup"`
					} `json:"commit"`
				}{}
				commitNode.Commit.StatusCheckRollup = &struct {
					State    string `json:"state"`
					Contexts struct {
						TotalCount int         `json:"totalCount"`
						Nodes      []checkNode `json:"nodes"`
					} `json:"contexts"`
				}{
					State: "FAILURE",
				}
				commitNode.Commit.StatusCheckRollup.Contexts.Nodes = []checkNode{
					{
						Typename:   "CheckRun",
						Name:       "integration-tests",
						Conclusion: "TIMED_OUT",
						Status:     "COMPLETED",
					},
					{
						Typename:   "CheckRun",
						Name:       "lint",
						Conclusion: "CANCELLED",
						Status:     "COMPLETED",
					},
				}
				node.Commits.Nodes = append(node.Commits.Nodes, commitNode)
				return node
			},
			validatePR: func(t *testing.T, pr ProcessedPR) {
				if !pr.HadCITimeout {
					t.Errorf("expected HadCITimeout to be true")
				}
				if pr.CIStatus != "TIMED_OUT" {
					t.Errorf("expected CIStatus TIMED_OUT, got %s", pr.CIStatus)
				}
				if pr.CITimedOutRuns != 1 {
					t.Errorf("expected 1 timed out run, got %d", pr.CITimedOutRuns)
				}
				if pr.CICancelledRuns != 1 {
					t.Errorf("expected 1 cancelled run, got %d", pr.CICancelledRuns)
				}
				if len(pr.TopTimedOutChecks) != 1 || pr.TopTimedOutChecks[0] != "integration-tests" {
					t.Errorf("expected TopTimedOutChecks [integration-tests], got %v", pr.TopTimedOutChecks)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node := tc.buildNode()
			pr := ProcessPRNode(node)
			tc.validatePR(t, pr)
		})
	}
}
