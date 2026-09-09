package cmd

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestParsePastDuration(t *testing.T) {
	refTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    string
		expected time.Time
		hasErr   bool
	}{
		{
			name:     "30 days relative duration",
			input:    "30d",
			expected: refTime.AddDate(0, 0, -30),
			hasErr:   false,
		},
		{
			name:     "2 weeks relative duration",
			input:    "2w",
			expected: refTime.AddDate(0, 0, -14),
			hasErr:   false,
		},
		{
			name:     "6 months relative duration",
			input:    "6m",
			expected: refTime.AddDate(0, -6, 0),
			hasErr:   false,
		},
		{
			name:     "1 year relative duration",
			input:    "1y",
			expected: refTime.AddDate(-1, 0, 0),
			hasErr:   false,
		},
		{
			name:     "empty duration defaults to 30 days / 1 month",
			input:    "",
			expected: refTime.AddDate(0, -1, 0),
			hasErr:   false,
		},
		{
			name:   "invalid duration string",
			input:  "invalid",
			hasErr: true,
		},
		{
			name:   "unknown time unit x",
			input:  "10x",
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := parsePastDuration(refTime, tc.input)
			if tc.hasErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if !res.Equal(tc.expected) {
				t.Errorf("for input %q, expected %v, got %v", tc.input, tc.expected, res)
			}
		})
	}
}

func createTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.PersistentFlags().StringVarP(&flagRepo, "repo", "R", "", "Target repository")
	cmd.PersistentFlags().BoolVar(&flagNoCache, "no-cache", false, "Bypass local disk cache")
	cmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Print verbose progress")
	cmd.PersistentFlags().IntVar(&flagPageSize, "page-size", 100, "GraphQL page size")
	attachTimeFlags(cmd)
	attachFilterFlags(cmd)
	attachOutputFlags(cmd)
	return cmd
}

func TestParseFilterOptions(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		validateOpts func(t *testing.T, opts metricsFilterOptionsHelper, err error)
	}{
		{
			name: "full flag configuration with json shortcut",
			args: []string{
				"--repo", "owner/repo",
				"--past", "30d",
				"--format", "text",
				"--json",
				"--group-by", "author",
				"--author", "alice",
				"--reviewer", "bob",
				"--label", "bug,core",
				"--base", "main",
				"--state", "merged",
				"--search", "oauth",
				"--draft", "false",
				"--review-state", "approved",
				"--assignee", "carol",
				"--review-requested", "dave",
				"--head", "feature/auth",
				"--milestone", "v1.0",
				"--checks", "success",
				"--min-lines", "50",
				"--max-lines", "500",
				"--min-files", "2",
				"--max-files", "10",
				"--has-conflicts", "false",
			},
			validateOpts: func(t *testing.T, opts metricsFilterOptionsHelper, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if opts.Repo != "owner/repo" {
					t.Errorf("expected repo owner/repo, got %s", opts.Repo)
				}
				if opts.Format != "json" {
					t.Errorf("expected format json, got %s", opts.Format)
				}
				if opts.Author != "alice" || opts.Reviewer != "bob" {
					t.Errorf("expected author alice, reviewer bob, got %s, %s", opts.Author, opts.Reviewer)
				}
				if opts.MinLines != 50 || opts.MaxLines != 500 {
					t.Errorf("expected line bounds [50, 500], got [%d, %d]", opts.MinLines, opts.MaxLines)
				}
				if !opts.HasSince || !opts.HasUntil {
					t.Errorf("expected HasSince and HasUntil to be true from --past")
				}
			},
		},
		{
			name: "explicit since and until dates",
			args: []string{
				"--csv",
				"--since", "2025-01-01",
				"--until", "2025-01-31",
			},
			validateOpts: func(t *testing.T, opts metricsFilterOptionsHelper, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if opts.Format != "csv" {
					t.Errorf("expected format csv, got %s", opts.Format)
				}
				if !opts.HasSince || opts.Since.Year() != 2025 || opts.Since.Month() != 1 || opts.Since.Day() != 1 {
					t.Errorf("expected Since to be 2025-01-01, got %v", opts.Since)
				}
				if !opts.HasUntil || opts.Until.Year() != 2025 || opts.Until.Month() != 1 || opts.Until.Day() != 31 {
					t.Errorf("expected Until to be 2025-01-31, got %v", opts.Until)
				}
			},
		},
		{
			name: "invalid since date format",
			args: []string{
				"--since", "invalid-date",
			},
			validateOpts: func(t *testing.T, opts metricsFilterOptionsHelper, err error) {
				if err == nil {
					t.Errorf("expected error for invalid date format in since, got nil")
				}
			},
		},
		{
			name: "invalid until date format",
			args: []string{
				"--since", "2025-01-01",
				"--until", "invalid-date",
			},
			validateOpts: func(t *testing.T, opts metricsFilterOptionsHelper, err error) {
				if err == nil {
					t.Errorf("expected error for invalid date format in until, got nil")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := createTestCommand()
			if err := cmd.ParseFlags(tc.args); err != nil {
				// If ParseFlags errors (e.g. invalid flag syntax)
				t.Fatalf("ParseFlags failed: %v", err)
			}
			opts, err := ParseFilterOptions(cmd)
			tc.validateOpts(t, metricsFilterOptionsHelper(opts), err)
		})
	}
}

func TestParseFilterOptionsVerboseAndPageSize(t *testing.T) {
	cmd := createTestCommand()
	if err := cmd.ParseFlags([]string{"--verbose", "--page-size", "50"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	opts, err := ParseFilterOptions(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !opts.Verbose {
		t.Errorf("expected Verbose=true, got false")
	}
	if opts.PageSize != 50 {
		t.Errorf("expected PageSize=50, got %d", opts.PageSize)
	}
}

func TestScopedFlagRegistration(t *testing.T) {
	// RootCmd persistent flags check
	globalFlags := []string{"repo", "no-cache", "verbose", "page-size"}
	for _, flagName := range globalFlags {
		if RootCmd.PersistentFlags().Lookup(flagName) == nil {
			t.Errorf("expected RootCmd to have persistent flag %q", flagName)
		}
	}

	// RootCmd should NOT have time/filter/output flags directly as persistent flags
	prFilterFlags := []string{
		"past", "since", "until",
		"format", "json", "csv", "tsv", "markdown", "percentiles", "detailed",
		"group-by",
		"search", "author", "reviewer", "assignee", "review-requested",
		"state", "draft", "review-state", "label", "base", "head", "milestone",
		"checks", "has-conflicts", "min-lines", "max-lines", "min-files", "max-files",
	}

	for _, flagName := range prFilterFlags {
		if RootCmd.PersistentFlags().Lookup(flagName) != nil {
			t.Errorf("RootCmd should not have persistent flag %q", flagName)
		}
	}

	// Metric domain command roots and leaf metric commands
	metricRoots := []*cobra.Command{
		timeCmd,
		qualityCmd,
		codeCmd,
		teamCmd,
		overviewCmd,
		exportCmd,
	}

	for _, cmd := range metricRoots {
		for _, flagName := range []string{"past", "author", "min-lines", "group-by", "checks", "has-conflicts"} {
			if cmd.PersistentFlags().Lookup(flagName) == nil {
				t.Errorf("expected command %s to have flag %q", cmd.Name(), flagName)
			}
		}
	}
}

func TestMetricSubcommandsAcceptMetricFlags(t *testing.T) {
	// Verify that metric subcommands accept metric, filter, time, and grouping flags
	metricSubcommands := []*cobra.Command{
		timeMergeCmd,
		timeReviewCmd,
		timeDraftCmd,
		timePickupCmd,
		timeIdleCmd,
		qualityCICmd,
		qualityReworkCmd,
		qualityConflictsCmd,
		qualityRevertsCmd,
		codeSizeCmd,
		codeCommitsCmd,
		teamThroughputCmd,
		teamReviewsCmd,
		teamUnreviewedCmd,
		overviewCmd,
		exportCmd,
	}

	expectedFlags := []string{
		"past",
		"since",
		"until",
		"author",
		"reviewer",
		"min-lines",
		"max-lines",
		"min-files",
		"max-files",
		"group-by",
		"checks",
		"has-conflicts",
		"state",
		"draft",
		"review-state",
		"label",
		"base",
		"head",
		"milestone",
		"search",
		"format",
		"json",
		"csv",
		"tsv",
		"markdown",
		"percentiles",
		"detailed",
		// Inherited global flags from RootCmd
		"repo",
		"no-cache",
		"verbose",
		"page-size",
	}

	for _, cmd := range metricSubcommands {
		t.Run(cmd.Name(), func(t *testing.T) {
			for _, flagName := range expectedFlags {
				if cmd.Flag(flagName) == nil {
					t.Errorf("expected subcommand %q to have flag %q", cmd.Name(), flagName)
				}
			}

			// Verify that parsing metric flags works on this subcommand
			testArgs := []string{
				"--past", "60d",
				"--author", "octocat",
				"--min-lines", "10",
				"--group-by", "month",
				"--checks", "success",
				"--has-conflicts", "false",
			}
			if err := cmd.ParseFlags(testArgs); err != nil {
				t.Errorf("subcommand %q failed to parse metric flags: %v", cmd.Name(), err)
			}
		})
	}
}

func TestCacheCommandsFlagScoping(t *testing.T) {
	cacheSubcommands := []*cobra.Command{
		cacheCmd,
		cacheListCmd,
		cacheCleanCmd,
		cachePathCmd,
	}

	disallowedFilterFlags := []string{
		"min-lines",
		"max-lines",
		"min-files",
		"max-files",
		"has-conflicts",
		"checks",
		"group-by",
		"author",
		"reviewer",
		"assignee",
		"review-requested",
		"draft",
		"review-state",
		"label",
		"base",
		"head",
		"milestone",
		"search",
		"past",
		"since",
		"until",
		"percentiles",
		"detailed",
	}

	requiredGlobalFlags := []string{
		"repo",
		"no-cache",
		"verbose",
		"page-size",
	}

	for _, cmd := range cacheSubcommands {
		t.Run(cmd.Name()+"_disallowed_flags", func(t *testing.T) {
			for _, flagName := range disallowedFilterFlags {
				if cmd.Flag(flagName) != nil {
					t.Errorf("cache command %q must NOT have PR filter flag %q", cmd.Name(), flagName)
				}
			}
		})

		t.Run(cmd.Name()+"_global_flags", func(t *testing.T) {
			for _, flagName := range requiredGlobalFlags {
				if cmd.Flag(flagName) == nil {
					t.Errorf("cache command %q should inherit global flag %q", cmd.Name(), flagName)
				}
			}
		})
	}

	// Verify cache clean specific flags
	if cacheCleanCmd.Flag("all") == nil {
		t.Errorf("expected cache clean command to have 'all' flag")
	}

	// Verify cache list specific flags
	if cacheListCmd.Flag("json") == nil {
		t.Errorf("expected cache list command to have 'json' flag")
	}

	// Verify that passing PR filter flags to cache commands returns an error
	t.Run("cache_list_rejects_pr_filter_flags", func(t *testing.T) {
		err := cacheListCmd.ParseFlags([]string{"--min-lines", "10"})
		if err == nil {
			t.Errorf("expected cache list to reject '--min-lines', but got nil error")
		}
		err = cacheListCmd.ParseFlags([]string{"--group-by", "month"})
		if err == nil {
			t.Errorf("expected cache list to reject '--group-by', but got nil error")
		}
		err = cacheListCmd.ParseFlags([]string{"--checks", "failure"})
		if err == nil {
			t.Errorf("expected cache list to reject '--checks', but got nil error")
		}
		err = cacheListCmd.ParseFlags([]string{"--has-conflicts", "true"})
		if err == nil {
			t.Errorf("expected cache list to reject '--has-conflicts', but got nil error")
		}
	})

	t.Run("cache_clean_rejects_pr_filter_flags", func(t *testing.T) {
		err := cacheCleanCmd.ParseFlags([]string{"--min-lines", "10"})
		if err == nil {
			t.Errorf("expected cache clean to reject '--min-lines', but got nil error")
		}
		err = cacheCleanCmd.ParseFlags([]string{"--has-conflicts", "false"})
		if err == nil {
			t.Errorf("expected cache clean to reject '--has-conflicts', but got nil error")
		}
	})
}

// metricsFilterOptionsHelper alias for cleaner tests
type metricsFilterOptionsHelper = struct {
	Repo            string
	Past            string
	Since           time.Time
	Until           time.Time
	HasSince        bool
	HasUntil        bool
	State           string
	Author          string
	Reviewer        string
	Label           string
	BaseBranch      string
	GroupBy         string
	Format          string
	Percentiles     bool
	Detailed        bool
	NoCache         bool
	Verbose         bool
	PageSize        int
	Search          string
	Draft           string
	ReviewState     string
	Assignee        string
	ReviewRequested string
	HeadBranch      string
	Milestone       string
	Checks          string
	MinLines        int
	MaxLines        int
	MinFiles        int
	MaxFiles        int
	HasConflicts    string
}
