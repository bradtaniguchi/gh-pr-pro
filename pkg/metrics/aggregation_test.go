package metrics

import (
	"testing"
	"time"
)

func TestAggregateMetricGrouping(t *testing.T) {
	ttm1 := 3600.0  // 1 hr
	ttm2 := 7200.0  // 2 hrs
	ttm3 := 10800.0 // 3 hrs

	prs := []ProcessedPR{
		{
			Number:             1,
			CreatedAt:          time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			State:              "MERGED",
			Author:             "alice",
			TimeToMergeSeconds: &ttm1,
		},
		{
			Number:             2,
			CreatedAt:          time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
			State:              "MERGED",
			Author:             "alice",
			TimeToMergeSeconds: &ttm2,
		},
		{
			Number:             3,
			CreatedAt:          time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
			State:              "MERGED",
			Author:             "bob",
			TimeToMergeSeconds: &ttm3,
		},
	}

	tests := []struct {
		name               string
		groupBy            string
		expectedGroupCount int
		expectedKeys       []string
		expectedCounts     []int
	}{
		{
			name:               "group by month",
			groupBy:            "month",
			expectedGroupCount: 2,
			expectedKeys:       []string{"2025-01", "2025-02"},
			expectedCounts:     []int{2, 1},
		},
		{
			name:               "group by author",
			groupBy:            "author",
			expectedGroupCount: 2,
			expectedKeys:       []string{"@alice", "@bob"},
			expectedCounts:     []int{2, 1},
		},
		{
			name:               "no grouping",
			groupBy:            "none",
			expectedGroupCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := AggregateMetric(prs, "time", "merge", FilterOptions{GroupBy: tc.groupBy})
			if len(out.Groups) != tc.expectedGroupCount {
				t.Fatalf("expected %d groups, got %d", tc.expectedGroupCount, len(out.Groups))
			}

			for i, key := range tc.expectedKeys {
				if out.Groups[i].GroupKey != key {
					t.Errorf("group %d: expected key %s, got %s", i, key, out.Groups[i].GroupKey)
				}
				if out.Groups[i].Count != tc.expectedCounts[i] {
					t.Errorf("group %d (%s): expected count %d, got %d", i, key, tc.expectedCounts[i], out.Groups[i].Count)
				}
			}

			if out.Summary.Count != 3 {
				t.Errorf("expected total count 3, got %d", out.Summary.Count)
			}
		})
	}
}

func TestFilterPRsAdvanced(t *testing.T) {
	prs := []ProcessedPR{
		{
			Number:            10,
			Title:             "feat: add oauth login",
			Body:              "Implements Google OAuth login flow",
			Author:            "alice",
			HeadBranch:        "feature/oauth",
			BaseBranch:        "main",
			IsDraft:           false,
			State:             "MERGED",
			Additions:         300,
			Deletions:         50,
			ChangedFiles:      4,
			HadCIFailure:      false,
			CITotalRuns:       5,
			HasMergeConflicts: false,
			Assignees:         []string{"alice", "carol"},
			ReviewRequests:    []string{"dave"},
			Milestone:         "v1.0",
			Reviewers: []ReviewerAction{
				{Login: "bob", State: "APPROVED"},
			},
		},
		{
			Number:            11,
			Title:             "WIP: refactor database queries",
			Body:              "Slow query optimization",
			Author:            "bob",
			HeadBranch:        "fix/db-queries",
			BaseBranch:        "main",
			IsDraft:           true,
			State:             "OPEN",
			Additions:         800,
			Deletions:         200,
			ChangedFiles:      12,
			HadCIFailure:      true,
			CITotalRuns:       3,
			HasMergeConflicts: true,
			Assignees:         []string{"bob"},
			ReviewRequests:    []string{"alice"},
			Milestone:         "v2.0",
			Reviewers: []ReviewerAction{
				{Login: "alice", State: "CHANGES_REQUESTED"},
			},
		},
		{
			Number:            12,
			Title:             "docs: update readme",
			Body:              "Fix typo in installation guide",
			Author:            "charlie",
			HeadBranch:        "patch-1",
			BaseBranch:        "develop",
			IsDraft:           false,
			State:             "CLOSED",
			Additions:         5,
			Deletions:         2,
			ChangedFiles:      1,
			HadCIFailure:      false,
			CITotalRuns:       0,
			HasMergeConflicts: false,
		},
	}

	tests := []struct {
		name              string
		opts              FilterOptions
		expectedPRCount   int
		expectedPRNumbers []int
	}{
		{
			name:              "free-text search matching title",
			opts:              FilterOptions{Search: "oauth"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{10},
		},
		{
			name:              "free-text search matching description body",
			opts:              FilterOptions{Search: "optimization"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{11},
		},
		{
			name:              "filter draft PRs only",
			opts:              FilterOptions{Draft: "true"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{11},
		},
		{
			name:              "filter non-draft ready PRs",
			opts:              FilterOptions{Draft: "false"},
			expectedPRCount:   2,
			expectedPRNumbers: []int{10, 12},
		},
		{
			name:              "filter approved review state",
			opts:              FilterOptions{ReviewState: "approved"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{10},
		},
		{
			name:              "filter unreviewed PRs without peer reviews",
			opts:              FilterOptions{ReviewState: "none"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{12},
		},
		{
			name:              "filter by assignee (@carol)",
			opts:              FilterOptions{Assignee: "@carol"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{10},
		},
		{
			name:              "filter by review requested (@alice)",
			opts:              FilterOptions{ReviewRequested: "@alice"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{11},
		},
		{
			name:              "filter by head branch substring (oauth)",
			opts:              FilterOptions{HeadBranch: "oauth"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{10},
		},
		{
			name:              "filter by milestone title (v2.0)",
			opts:              FilterOptions{Milestone: "v2.0"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{11},
		},
		{
			name:              "filter by CI failure status",
			opts:              FilterOptions{Checks: "failure"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{11},
		},
		{
			name:              "filter by CI success status",
			opts:              FilterOptions{Checks: "success"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{10},
		},
		{
			name:              "filter by min lines (500)",
			opts:              FilterOptions{MinLines: 500},
			expectedPRCount:   1,
			expectedPRNumbers: []int{11},
		},
		{
			name:              "filter by max lines (100)",
			opts:              FilterOptions{MaxLines: 100},
			expectedPRCount:   1,
			expectedPRNumbers: []int{12},
		},
		{
			name:              "filter by merge conflicts (true)",
			opts:              FilterOptions{HasConflicts: "true"},
			expectedPRCount:   1,
			expectedPRNumbers: []int{11},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := FilterPRs(prs, tc.opts)
			if len(res) != tc.expectedPRCount {
				t.Fatalf("expected %d PRs, got %d", tc.expectedPRCount, len(res))
			}

			for i, num := range tc.expectedPRNumbers {
				if res[i].Number != num {
					t.Errorf("at index %d: expected PR %d, got %d", i, num, res[i].Number)
				}
			}
		})
	}
}

func TestAggregateMetricAllDimensionsAndDomains(t *testing.T) {
	pickup1 := 1800.0
	pickup2 := 3600.0
	ttm1 := 7200.0
	ttm2 := 14400.0
	ttfr1 := 900.0

	prs := []ProcessedPR{
		{
			Number:                1,
			CreatedAt:             time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			State:                 "MERGED",
			Author:                "alice",
			BaseBranch:            "main",
			SizeCategory:          "S (10-99)",
			Labels:                []string{"feature"},
			TimeToMergeSeconds:    &ttm1,
			TimeToFirstReview:     &ttfr1,
			DraftDurationSeconds:  600,
			PickupDurationSeconds: &pickup1,
			IdleDurationSeconds:   1200,
			Additions:             50,
			Deletions:             10,
			CommitCount:           3,
			HadCIFailure:          false,
			CITotalRuns:           2,
			ReviewRoundtrips:      1,
			HasMergeConflicts:     false,
			IsReverted:            false,
			IsUnreviewed:          false,
			Reviewers: []ReviewerAction{
				{Login: "bob", State: "APPROVED"},
			},
		},
		{
			Number:                2,
			CreatedAt:             time.Date(2025, 4, 20, 10, 0, 0, 0, time.UTC),
			State:                 "CLOSED",
			Author:                "bob",
			BaseBranch:            "release/v1",
			SizeCategory:          "L (500-999)",
			Labels:                []string{"bug"},
			TimeToMergeSeconds:    &ttm2,
			DraftDurationSeconds:  3600,
			PickupDurationSeconds: &pickup2,
			IdleDurationSeconds:   2400,
			Additions:             600,
			Deletions:             200,
			CommitCount:           8,
			HadCIFailure:          true,
			CITotalRuns:           4,
			ReviewRoundtrips:      3,
			HasMergeConflicts:     true,
			IsReverted:            true,
			IsUnreviewed:          true,
		},
	}

	tests := []struct {
		name          string
		domain        string
		metric        string
		opts          FilterOptions
		expectedCount int
		expectedMean  float64
		validateExtra func(t *testing.T, extra map[string]interface{})
	}{
		{
			name:          "time domain: draft duration",
			domain:        "time",
			metric:        "draft",
			expectedCount: 2,
			expectedMean:  2100,
		},
		{
			name:          "time domain: pickup latency",
			domain:        "time",
			metric:        "pickup",
			expectedCount: 2,
			expectedMean:  2700,
		},
		{
			name:          "time domain: idle duration",
			domain:        "time",
			metric:        "idle",
			expectedCount: 2,
			expectedMean:  1800,
		},
		{
			name:          "quality domain: rework roundtrips",
			domain:        "quality",
			metric:        "rework",
			expectedCount: 2,
			expectedMean:  2,
		},
		{
			name:          "quality domain: conflicts percentage",
			domain:        "quality",
			metric:        "conflicts",
			expectedCount: 2,
			expectedMean:  50.0,
		},
		{
			name:          "quality domain: reverts percentage",
			domain:        "quality",
			metric:        "reverts",
			expectedCount: 2,
			expectedMean:  50.0,
		},
		{
			name:          "code domain: commits per PR",
			domain:        "code",
			metric:        "commits",
			expectedCount: 2,
			expectedMean:  5.5,
		},
		{
			name:          "team domain: throughput counts",
			domain:        "team",
			metric:        "throughput",
			expectedCount: 2,
			expectedMean:  1,
			validateExtra: func(t *testing.T, extra map[string]interface{}) {
				if extra["opened"] != 2 || extra["merged"] != 1 {
					t.Errorf("expected 2 opened and 1 merged, got %v", extra)
				}
			},
		},
		{
			name:          "team domain: unreviewed rate",
			domain:        "team",
			metric:        "unreviewed",
			expectedCount: 2,
			expectedMean:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := AggregateMetric(prs, tc.domain, tc.metric, tc.opts)
			if out.Summary.Count != tc.expectedCount {
				t.Errorf("expected count %d, got %d", tc.expectedCount, out.Summary.Count)
			}
			if out.Summary.Mean != tc.expectedMean {
				t.Errorf("expected mean %v, got %v", tc.expectedMean, out.Summary.Mean)
			}
			if tc.validateExtra != nil {
				tc.validateExtra(t, out.Summary.ExtraMetrics)
			}
		})
	}

	// Subtests for all GroupBy dimensions
	groupDimensions := []string{"day", "week", "quarter", "year", "author", "reviewer", "label", "base", "size"}
	for _, dim := range groupDimensions {
		t.Run("group dimension "+dim, func(t *testing.T) {
			out := AggregateMetric(prs, "time", "merge", FilterOptions{GroupBy: dim})
			if len(out.Groups) == 0 {
				t.Errorf("expected groups for dimension %s, got 0", dim)
			}
		})
	}
}
