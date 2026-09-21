package api

import (
	"fmt"
	"testing"
	"time"
)

func TestResolveRepo(t *testing.T) {
	tests := []struct {
		name          string
		repoFlag      string
		expectedHost  string
		expectedOwner string
		expectedName  string
		expectErr     bool
	}{
		{
			name:          "standard owner/repo",
			repoFlag:      "cli/cli",
			expectedOwner: "cli",
			expectedName:  "cli",
			expectErr:     false,
		},
		{
			name:          "host qualified repository",
			repoFlag:      "github.com/golang/go",
			expectedHost:  "github.com",
			expectedOwner: "golang",
			expectedName:  "go",
			expectErr:     false,
		},
		{
			name:          "enterprise host qualified repository",
			repoFlag:      "github.mycompany.com/team/repo",
			expectedHost:  "github.mycompany.com",
			expectedOwner: "team",
			expectedName:  "repo",
			expectErr:     false,
		},
		{
			name:      "invalid repository format without slash",
			repoFlag:  "invalidformatwithoutslash",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := ResolveRepo(tc.repoFlag)
			if tc.expectErr {
				if err == nil {
					t.Errorf("expected error for repo %q, got nil", tc.repoFlag)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for repo %q: %v", tc.repoFlag, err)
			}

			if r.Owner != tc.expectedOwner || r.Name != tc.expectedName {
				t.Errorf("expected %s/%s, got %s/%s", tc.expectedOwner, tc.expectedName, r.Owner, r.Name)
			}

			if tc.expectedHost != "" && r.Host != tc.expectedHost {
				t.Errorf("expected host %s, got %s", tc.expectedHost, r.Host)
			}
		})
	}
}

type mockGQLClient struct {
	doFunc func(query string, variables map[string]interface{}, response interface{}) error
}

func (m *mockGQLClient) Do(query string, variables map[string]interface{}, response interface{}) error {
	if m.doFunc != nil {
		return m.doFunc(query, variables, response)
	}
	return nil
}

func TestSetProgressFunc(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

	type progressCall struct {
		page    int
		prCount int
	}

	tests := []struct {
		name          string
		pages         [][]GraphQLPRNode
		maxPRs        int
		since         time.Time
		lastUpdated   time.Time
		withCallback  bool
		expectedCalls []progressCall
		expectedPRs   int
	}{
		{
			name: "single page fetch",
			pages: [][]GraphQLPRNode{
				{
					{Number: 1, Title: "PR 1", CreatedAt: now, UpdatedAt: now},
					{Number: 2, Title: "PR 2", CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
				},
			},
			withCallback: true,
			expectedCalls: []progressCall{
				{page: 1, prCount: 0},
				{page: 1, prCount: 2},
			},
			expectedPRs: 2,
		},
		{
			name: "multi-page pagination across 3 pages",
			pages: [][]GraphQLPRNode{
				{
					{Number: 5, Title: "PR 5", CreatedAt: now, UpdatedAt: now},
					{Number: 4, Title: "PR 4", CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
				},
				{
					{Number: 3, Title: "PR 3", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
					{Number: 2, Title: "PR 2", CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
				},
				{
					{Number: 1, Title: "PR 1", CreatedAt: now.Add(-4 * time.Hour), UpdatedAt: now.Add(-4 * time.Hour)},
				},
			},
			withCallback: true,
			expectedCalls: []progressCall{
				{page: 1, prCount: 0},
				{page: 1, prCount: 2},
				{page: 2, prCount: 2},
				{page: 2, prCount: 4},
				{page: 3, prCount: 4},
				{page: 3, prCount: 5},
			},
			expectedPRs: 5,
		},
		{
			name: "pagination stopped early by maxPRs threshold",
			pages: [][]GraphQLPRNode{
				{
					{Number: 4, Title: "PR 4", CreatedAt: now, UpdatedAt: now},
					{Number: 3, Title: "PR 3", CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
				},
				{
					{Number: 2, Title: "PR 2", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
					{Number: 1, Title: "PR 1", CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
				},
			},
			maxPRs:       3,
			withCallback: true,
			expectedCalls: []progressCall{
				{page: 1, prCount: 0},
				{page: 1, prCount: 2},
				{page: 2, prCount: 2},
				{page: 2, prCount: 3},
			},
			expectedPRs: 3,
		},
		{
			name: "delta sync stop condition when PR older than lastUpdated cache",
			pages: [][]GraphQLPRNode{
				{
					{Number: 2, Title: "PR 2", CreatedAt: now, UpdatedAt: now},
					{Number: 1, Title: "PR 1", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
				},
				{
					{Number: 0, Title: "PR 0", CreatedAt: now.Add(-4 * time.Hour), UpdatedAt: now.Add(-4 * time.Hour)},
				},
			},
			lastUpdated:  now.Add(-1 * time.Hour),
			withCallback: true,
			expectedCalls: []progressCall{
				{page: 1, prCount: 0},
				{page: 1, prCount: 2},
			},
			expectedPRs: 2,
		},
		{
			name: "since filter stop condition when PR older than since",
			pages: [][]GraphQLPRNode{
				{
					{Number: 2, Title: "PR 2", CreatedAt: now, UpdatedAt: now},
					{Number: 1, Title: "PR 1", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
				},
				{
					{Number: 0, Title: "PR 0", CreatedAt: now.Add(-4 * time.Hour), UpdatedAt: now.Add(-4 * time.Hour)},
				},
			},
			since:        now.Add(-1 * time.Hour),
			withCallback: true,
			expectedCalls: []progressCall{
				{page: 1, prCount: 0},
				{page: 1, prCount: 2},
			},
			expectedPRs: 2,
		},
		{
			name:         "empty repository with 0 PRs",
			pages:        [][]GraphQLPRNode{{}},
			withCallback: true,
			expectedCalls: []progressCall{
				{page: 1, prCount: 0},
				{page: 1, prCount: 0},
			},
			expectedPRs: 0,
		},
		{
			name: "nil progress callback executes safely without panicking",
			pages: [][]GraphQLPRNode{
				{
					{Number: 1, Title: "PR 1", CreatedAt: now, UpdatedAt: now},
				},
			},
			withCallback:  false,
			expectedCalls: nil,
			expectedPRs:   1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pageIdx := 0
			mockClient := &mockGQLClient{
				doFunc: func(query string, variables map[string]interface{}, response interface{}) error {
					res, ok := response.(*GraphQLRepoPRResponse)
					if !ok {
						return fmt.Errorf("unexpected response type")
					}
					if pageIdx >= len(tc.pages) {
						res.Repository.PullRequests.Nodes = nil
						res.Repository.PullRequests.PageInfo.HasNextPage = false
						return nil
					}
					nodes := tc.pages[pageIdx]
					hasNext := pageIdx+1 < len(tc.pages)
					res.Repository.PullRequests.Nodes = nodes
					res.Repository.PullRequests.PageInfo.HasNextPage = hasNext
					if hasNext {
						res.Repository.PullRequests.PageInfo.EndCursor = fmt.Sprintf("cursor-%d", pageIdx+1)
					}
					pageIdx++
					return nil
				},
			}

			client := &Client{
				gqlClient: mockClient,
				PageSize:  100,
			}

			var recordedCalls []progressCall
			if tc.withCallback {
				client.SetProgressFunc(func(page int, prCount int) {
					recordedCalls = append(recordedCalls, progressCall{page: page, prCount: prCount})
				})
			}

			prs, err := client.FetchPRsDelta("testowner", "testrepo", tc.since, tc.lastUpdated, tc.maxPRs)
			if err != nil {
				t.Fatalf("unexpected FetchPRsDelta error: %v", err)
			}

			if len(prs) != tc.expectedPRs {
				t.Errorf("expected %d PRs, got %d", tc.expectedPRs, len(prs))
			}

			if tc.withCallback {
				if len(recordedCalls) != len(tc.expectedCalls) {
					t.Fatalf("expected %d progress calls, got %d: %+v", len(tc.expectedCalls), len(recordedCalls), recordedCalls)
				}
				for i, call := range recordedCalls {
					expected := tc.expectedCalls[i]
					if call.page != expected.page || call.prCount != expected.prCount {
						t.Errorf("call %d: expected {page: %d, prCount: %d}, got {page: %d, prCount: %d}",
							i, expected.page, expected.prCount, call.page, call.prCount)
					}
				}
			} else {
				if len(recordedCalls) != 0 {
					t.Errorf("expected no progress calls, got %d", len(recordedCalls))
				}
			}
		})
	}
}

func TestSetPageSize(t *testing.T) {
	tests := []struct {
		name         string
		inputSize    int
		expectedSize int
	}{
		{
			name:         "valid page size",
			inputSize:    50,
			expectedSize: 50,
		},
		{
			name:         "zero page size resets to 25",
			inputSize:    0,
			expectedSize: 25,
		},
		{
			name:         "negative page size resets to 25",
			inputSize:    -5,
			expectedSize: 25,
		},
		{
			name:         "page size exceeding 100 clamped to 100",
			inputSize:    150,
			expectedSize: 100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{}
			c.SetPageSize(tc.inputSize)
			if c.PageSize != tc.expectedSize {
				t.Errorf("expected PageSize %d, got %d", tc.expectedSize, c.PageSize)
			}
		})
	}
}

func TestSetVerbose(t *testing.T) {
	c := &Client{}
	if c.Verbose != false {
		t.Errorf("expected default Verbose to be false")
	}
	c.SetVerbose(true)
	if c.Verbose != true {
		t.Errorf("expected Verbose to be true")
	}
}
