package api

import "time"

// GraphQLRateLimit captures API quota and rate limit metadata returned in GraphQL query headers.
type GraphQLRateLimit struct {
	// Cost is the number of points deducted from the rate limit by the GraphQL query.
	Cost int `json:"cost"`
	// Remaining is the number of API rate limit points left in the current hourly window.
	Remaining int `json:"remaining"`
	// ResetAt is the timestamp at which the rate limit quota resets to its full capacity.
	ResetAt time.Time `json:"resetAt"`
}

// GraphQLPRNode represents the raw Pull Request data structure unmarshaled from GitHub's GraphQL API.
// It includes core PR metadata, author, reviewers, status check rollups, reviews, and timeline events.
type GraphQLPRNode struct {
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	State        string     `json:"state"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	MergedAt     *time.Time `json:"mergedAt"`
	ClosedAt     *time.Time `json:"closedAt"`
	IsDraft      bool       `json:"isDraft"`
	BaseRefName  string     `json:"baseRefName"`
	HeadRefName  string     `json:"headRefName"`
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
	ChangedFiles int        `json:"changedFiles"`
	Mergeable    string     `json:"mergeable"`

	Author struct {
		Login string `json:"login"`
	} `json:"author"`

	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`

	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`

	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`

	Commits struct {
		TotalCount int `json:"totalCount"`
		Nodes      []struct {
			Commit struct {
				CommittedDate     time.Time `json:"committedDate"`
				StatusCheckRollup *struct {
					State    string `json:"state"`
					Contexts struct {
						TotalCount int `json:"totalCount"`
						Nodes      []struct {
							Typename string `json:"__typename"`
							// For CheckRun
							Name       string `json:"name,omitempty"`
							Conclusion string `json:"conclusion,omitempty"`
							Status     string `json:"status,omitempty"`
							// For StatusContext
							Context string `json:"context,omitempty"`
							State   string `json:"state,omitempty"`
						} `json:"nodes"`
					} `json:"contexts"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`

	Reviews struct {
		TotalCount int `json:"totalCount"`
		Nodes      []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			State       string    `json:"state"`
			SubmittedAt time.Time `json:"submittedAt"`
			Body        string    `json:"body"`
		} `json:"nodes"`
	} `json:"reviews"`

	ReviewRequests struct {
		TotalCount int `json:"totalCount"`
		Nodes      []struct {
			RequestedReviewer struct {
				Typename string `json:"__typename"`
				Login    string `json:"login,omitempty"`
				Name     string `json:"name,omitempty"`
			} `json:"requestedReviewer"`
		} `json:"nodes"`
	} `json:"reviewRequests"`

	TimelineItems struct {
		Nodes []struct {
			Typename  string    `json:"__typename"`
			CreatedAt time.Time `json:"createdAt"`
			// For ReadyForReviewEvent / ConvertToDraftEvent
			Actor struct {
				Login string `json:"login"`
			} `json:"actor"`
		} `json:"nodes"`
	} `json:"timelineItems"`
}

// GraphQLRepoPRResponse is the top-level container structure for decoding responses from the FetchPRs GraphQL query.
type GraphQLRepoPRResponse struct {
	RateLimit  GraphQLRateLimit `json:"rateLimit"`
	Repository struct {
		PullRequests struct {
			TotalCount int `json:"totalCount"`
			PageInfo   struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Nodes []GraphQLPRNode `json:"nodes"`
		} `json:"pullRequests"`
	} `json:"repository"`
}
