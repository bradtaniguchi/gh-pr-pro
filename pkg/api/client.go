// Package api provides client functionality and data models for interacting
// with GitHub's GraphQL API.
//
// It manages GraphQL queries, automated pagination, resilient retry policies
// for transient network issues or secondary rate limiting, and repository resolution.
package api

import (
	"fmt"
	"os"
	"strings"
	"time"

	ghapi "github.com/cli/go-gh/v2/pkg/api"
	ghrepo "github.com/cli/go-gh/v2/pkg/repository"
)

// Client wraps the GitHub CLI GraphQL client to execute queries against the GitHub GraphQL API.
// It includes built-in retry logic, rate limit awareness, and pagination helpers for pull request datasets.
type Client struct {
	gqlClient *ghapi.GraphQLClient
	Verbose   bool
	PageSize  int
}

// NewClient initializes and returns a new GitHub GraphQL API Client using go-gh authentication.
// It configures client options and returns an error if the underlying GraphQL client fails to initialize.
func NewClient() (*Client, error) {
	opts := ghapi.ClientOptions{
		EnableCache: false,
	}
	gql, err := ghapi.NewGraphQLClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub GraphQL client: %w", err)
	}
	return &Client{
		gqlClient: gql,
		PageSize:  100,
	}, nil
}

// SetVerbose configures whether verbose diagnostic timestamps and progress logs are written to stderr.
func (c *Client) SetVerbose(verbose bool) {
	c.Verbose = verbose
}

// SetPageSize configures the GraphQL query page size (clamped between 1 and 100).
func (c *Client) SetPageSize(pageSize int) {
	if pageSize <= 0 {
		c.PageSize = 100
	} else if pageSize > 100 {
		c.PageSize = 100
	} else {
		c.PageSize = pageSize
	}
}

func (c *Client) logVerbose(format string, args ...interface{}) {
	if c.Verbose {
		ts := time.Now().Format("2006-01-02 15:04:05.000")
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(os.Stderr, "[%s] [api] %s\n", ts, msg)
	}
}

// ResolveRepo resolves a target GitHub repository from a CLI flag or current git context.
// If repoFlag is provided (e.g., "owner/repo" or "HOST/owner/repo"), it parses that string;
// otherwise, it inspects the current git directory remotes using go-gh.
func ResolveRepo(repoFlag string) (ghrepo.Repository, error) {
	if repoFlag != "" {
		return ghrepo.Parse(repoFlag)
	}
	return ghrepo.Current()
}

const prQuery = `
query FetchPRs($owner: String!, $name: String!, $cursor: String, $pageSize: Int!) {
  rateLimit {
    cost
    remaining
    resetAt
  }
  repository(owner: $owner, name: $name) {
    pullRequests(first: $pageSize, after: $cursor, orderBy: {field: CREATED_AT, direction: DESC}) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        number
        title
        body
        state
        createdAt
        updatedAt
        mergedAt
        closedAt
        isDraft
        baseRefName
        headRefName
        additions
        deletions
        changedFiles
        mergeable
        author {
          login
        }
        assignees(first: 5) {
          nodes {
            login
          }
        }
        milestone {
          title
        }
        labels(first: 10) {
          nodes {
            name
          }
        }
        commits(last: 1) {
          totalCount
          nodes {
            commit {
              committedDate
              statusCheckRollup {
                state
                contexts(first: 20) {
                  totalCount
                  nodes {
                    __typename
                    ... on CheckRun {
                      name
                      conclusion
                      status
                    }
                    ... on StatusContext {
                      context
                      state
                    }
                  }
                }
              }
            }
          }
        }
        reviews(first: 30) {
          totalCount
          nodes {
            author {
              login
            }
            state
            submittedAt
            body
          }
        }
        reviewRequests(first: 10) {
          totalCount
          nodes {
            requestedReviewer {
              __typename
              ... on User {
                login
              }
              ... on Team {
                name
              }
            }
          }
        }
        timelineItems(first: 30, itemTypes: [READY_FOR_REVIEW_EVENT, CONVERT_TO_DRAFT_EVENT]) {
          nodes {
            __typename
            ... on ReadyForReviewEvent {
              createdAt
              actor {
                login
              }
            }
            ... on ConvertToDraftEvent {
              createdAt
              actor {
                login
              }
            }
          }
        }
      }
    }
  }
}
`

// executeQueryWithRetry executes a GraphQL query with automatic retries on rate limits,
// 5xx server errors, and network timeouts using exponential backoff.
//
// It also checks GitHub's remaining rate limit points and pauses execution if
// the limit drops to critically low thresholds (< 10 points).
func (c *Client) executeQueryWithRetry(query string, variables map[string]interface{}, response *GraphQLRepoPRResponse) error {
	maxRetries := 3
	backoff := 2 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		reqStart := time.Now()
		err := c.gqlClient.Do(query, variables, response)
		reqDuration := time.Since(reqStart)

		if err == nil {
			// Check rate limit safety
			if response.RateLimit.Remaining > 0 && response.RateLimit.Remaining <= 10 {
				sleepDuration := time.Until(response.RateLimit.ResetAt) + (2 * time.Second)
				if sleepDuration > 0 && sleepDuration < 2*time.Minute {
					ts := time.Now().Format("2006-01-02 15:04:05.000")
					fmt.Fprintf(os.Stderr, "[%s] ⚠️ GitHub API rate limit critically low (%d remaining). Pausing for %v...\n",
						ts, response.RateLimit.Remaining, sleepDuration.Round(time.Second))
					time.Sleep(sleepDuration)
				}
			} else if response.RateLimit.Remaining > 0 && response.RateLimit.Remaining < 100 {
				ts := time.Now().Format("2006-01-02 15:04:05.000")
				fmt.Fprintf(os.Stderr, "[%s] ⚠️ GitHub API rate limit low: %d points remaining (resets at %s)\n",
					ts, response.RateLimit.Remaining, response.RateLimit.ResetAt.Format("15:04:05"))
			}
			return nil
		}

		errStr := err.Error()
		isRateLimitOrTransient := strings.Contains(errStr, "rate limit") ||
			strings.Contains(errStr, "secondary rate limit") ||
			strings.Contains(errStr, "502") ||
			strings.Contains(errStr, "503") ||
			strings.Contains(errStr, "504") ||
			strings.Contains(errStr, "timeout")

		if isRateLimitOrTransient && attempt < maxRetries {
			ts := time.Now().Format("2006-01-02 15:04:05.000")
			fmt.Fprintf(os.Stderr, "[%s] ⚠️ GitHub API request throttled or transient error (after %v): %v. Retrying in %v (attempt %d/%d)...\n",
				ts, reqDuration.Round(time.Millisecond), err, backoff, attempt+1, maxRetries)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		return fmt.Errorf("GraphQL query failed: %w", err)
	}

	return fmt.Errorf("GraphQL query exceeded maximum retries")
}

// FetchPRs queries pull requests for a given repository owner and name.
// It retrieves pull requests created on or after the specified since timestamp,
// up to a maximum limit of maxPRs (or all available matching PRs if maxPRs <= 0).
func (c *Client) FetchPRs(owner, name string, since time.Time, maxPRs int) ([]GraphQLPRNode, error) {
	return c.FetchPRsDelta(owner, name, since, time.Time{}, maxPRs)
}

// FetchPRsDelta retrieves pull requests with incremental delta sync support.
// It paginates backwards through repository PRs, stopping early when a PR's
// UpdatedAt is older than lastUpdated or CreatedAt is older than since.
// This minimizes API payload sizes and saves rate limit quota when syncing against existing cache entries.
func (c *Client) FetchPRsDelta(owner, name string, since, lastUpdated time.Time, maxPRs int) ([]GraphQLPRNode, error) {
	var allNodes []GraphQLPRNode
	var cursor *string
	pageSize := c.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}
	if maxPRs > 0 && maxPRs < pageSize {
		pageSize = maxPRs
	}

	pageNum := 1
	startTotal := time.Now()

	sinceStr := "none"
	if !since.IsZero() {
		sinceStr = since.Format("2006-01-02 15:04:05")
	}
	lastUpdatedStr := "none"
	if !lastUpdated.IsZero() {
		lastUpdatedStr = lastUpdated.Format("2006-01-02 15:04:05")
	}

	c.logVerbose("Initiating PR fetch for %s/%s (pageSize: %d, since: %s, delta cutoff: %s)",
		owner, name, pageSize, sinceStr, lastUpdatedStr)

	for {
		variables := map[string]interface{}{
			"owner":    owner,
			"name":     name,
			"pageSize": pageSize,
		}
		if cursor != nil {
			variables["cursor"] = *cursor
		}

		c.logVerbose("Requesting page %d (pageSize: %d, total fetched so far: %d)...",
			pageNum, pageSize, len(allNodes))
		pageStart := time.Now()

		var response GraphQLRepoPRResponse
		err := c.executeQueryWithRetry(prQuery, variables, &response)
		if err != nil {
			// If pageSize is still > 10 and we hit a timeout / bad gateway error, retry with smaller pageSize
			if pageSize > 10 && (strings.Contains(err.Error(), "502") || strings.Contains(err.Error(), "504") || strings.Contains(err.Error(), "timeout")) {
				pageSize = pageSize / 2
				if pageSize < 10 {
					pageSize = 10
				}
				ts := time.Now().Format("2006-01-02 15:04:05.000")
				fmt.Fprintf(os.Stderr, "[%s] ⚠️ Query timed out on large repository. Reducing page size to %d and retrying page %d...\n",
					ts, pageSize, pageNum)
				continue
			}
			return nil, err
		}

		pageDuration := time.Since(pageStart)
		prs := response.Repository.PullRequests.Nodes
		c.logVerbose("Page %d received in %v (%d PRs returned | rate limit remaining: %d, cost: %d)",
			pageNum, pageDuration.Round(time.Millisecond), len(prs), response.RateLimit.Remaining, response.RateLimit.Cost)

		if len(prs) == 0 {
			c.logVerbose("Page %d returned 0 PRs. Stopping pagination.", pageNum)
			break
		}

		shouldStop := false
		var stopReason string
		for _, node := range prs {
			allNodes = append(allNodes, node)
			if maxPRs > 0 && len(allNodes) >= maxPRs {
				shouldStop = true
				stopReason = fmt.Sprintf("reached maximum PR threshold of %d", maxPRs)
				break
			}
			// Delta cutoff: if we have a lastUpdated cache timestamp and we passed it
			if !lastUpdated.IsZero() && node.UpdatedAt.Before(lastUpdated) {
				shouldStop = true
				stopReason = fmt.Sprintf("PR #%d updated at %s is older than cache timestamp %s",
					node.Number, node.UpdatedAt.Format("2006-01-02 15:04:05"), lastUpdated.Format("2006-01-02 15:04:05"))
				break
			}
			// If we passed the 'since' boundary on created_at, we can safely stop paginating backwards
			if !since.IsZero() && node.CreatedAt.Before(since) {
				shouldStop = true
				stopReason = fmt.Sprintf("PR #%d created at %s is older than since boundary %s",
					node.Number, node.CreatedAt.Format("2006-01-02 15:04:05"), since.Format("2006-01-02 15:04:05"))
				break
			}
		}

		if shouldStop {
			c.logVerbose("Pagination stop condition reached on page %d: %s", pageNum, stopReason)
			break
		}

		if !response.Repository.PullRequests.PageInfo.HasNextPage {
			c.logVerbose("GitHub API indicates no more pages available (hasNextPage: false).")
			break
		}

		endCursor := response.Repository.PullRequests.PageInfo.EndCursor
		cursor = &endCursor
		pageNum++
	}

	c.logVerbose("Fetch complete for %s/%s: %d PRs retrieved across %d page(s) in %v",
		owner, name, len(allNodes), pageNum, time.Since(startTotal).Round(time.Millisecond))

	return allNodes, nil
}
