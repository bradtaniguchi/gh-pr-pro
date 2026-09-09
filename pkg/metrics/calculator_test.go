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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node := tc.buildNode()
			pr := ProcessPRNode(node)
			tc.validatePR(t, pr)
		})
	}
}
