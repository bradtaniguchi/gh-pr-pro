# `gh-pr-pro`

[![GitHub CLI Extension](https://img.shields.io/badge/gh-extension-blue.svg)](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
[![Downloads](https://img.shields.io/github/downloads/bradtaniguchi/gh-pr-pro/total)](https://github.com/bradtaniguchi/gh-pr-pro/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Github CLI extension to provide extensive pull request metrics, with built in caching, csv/json/text output with extensive filtering.

> [!WARNING]
> **API Rate Limits & 10-Second Request Timeout**: Analyzing large repositories or expansive historical time ranges queries GitHub's GraphQL API. Queries are subject to GitHub's primary point quotas (check anytime with `gh api rate_limit`) and a strict **10-second per-request execution timeout** on GitHub's backend. `gh-pr-pro` defaults to `--page-size 25` and uses local disk caching to prevent timeouts and preserve quota. See [Rate Limits & GraphQL Query Economics](#rate-limits--graphql-query-economics) for details and tuning guidance.

---

## Installation

```bash
gh extension install bradtaniguchi/gh-pr-pro
```

---

## Quick Start

```bash
# 1. How long do PRs take to merge over the past 90 days? (Grouped by month)
gh pr-pro time merge --past 90d --group-by month

# 2. Check reviewer turnaround times for the last 30 days (CSV format)
gh pr-pro time review --past 30d --group-by reviewer --csv

# 3. Analyze how long PRs sit in draft status
gh pr-pro time draft --past 6m --group-by author

# 4. Inspect CI/CD failure rates and retries on main
gh pr-pro quality ci --past 60d --base main

# 5. Review workload and team review balance
gh pr-pro team reviews --past 90d

# 6. Full executive summary across all metrics
gh pr-pro overview --past 30d
```

---

## AI Agent & Automation Guidance

When invoking `gh-pr-pro` from an AI coding agent, CI workflow, or automation script:

* **Executive Scorecard**: Prefer `gh pr-pro overview --past 30d --json` for a high-level summary covering cycle times, review latency, CI stability, and team velocity rather than exporting raw records.
* **Domain Subcommands**: Query purpose-built subcommands with `--json` for structured metrics:
  * Merge & review cycle times: `gh pr-pro time merge --json`, `gh pr-pro time review --json`
  * CI stability & rework: `gh pr-pro quality ci --json`, `gh pr-pro quality rework --json`
  * Sizing & file distribution: `gh pr-pro code size --json`
  * Throughput & reviewer workload: `gh pr-pro team throughput --json`, `gh pr-pro team reviews --json`
* **Default Time Window**: All analytical commands default to `--past 30d`. Always pass `--past <duration>` (e.g., `--past 90d`, `--past 1y`) or `--since <YYYY-MM-DD>` (optionally with `--until`) when evaluating custom time ranges.
* **Data Pipelines**: Reserve `export` strictly for external ETL streaming (DuckDB, Pandas, Snowflake, PostgreSQL). Avoid using `export` to compute percentiles or aggregates in prompt context.

---

## Flag Scoping & Reference

`gh-pr-pro` uses scoped command flags to ensure clean and intuitive CLI interactions. Flags are structured into **Global Flags**, **Domain / Metric Flags**, and **Cache Flags**:

### 1. Global Flags
Available across **all** commands and subcommands (including `time`, `quality`, `code`, `team`, `overview`, `export`, and `cache`):

| Flag | Short | Default | Description |
|---|---|---|---|
| `--repo` | `-R` | current | Target repository in `[HOST/]OWNER/REPO` format (defaults to current git repository) |
| `--no-cache` | | `false` | Bypass local disk cache (`~/.cache/gh-pr-pro/`) and force a complete fresh fetch from GitHub API |
| `--verbose` | `-v` | `false` | Print verbose progress, timestamped network latency, and cache activity logs to stderr |
| `--page-size` | | `25` | GraphQL page size for PR queries (1 to 100 max, default: 25). Lower values (10–30) avoid GitHub 10s query execution timeouts / HTTP 504 on large repositories |

### 2. Domain / Metric Flags
Available on metric domain commands (`time`, `quality`, `code`, `team`), `overview`, and `export`.

#### Time Window Flags
| Flag | Short | Default | Description |
|---|---|---|---|
| `--past` | `-p` | `30d` | Relative historical time range to analyze from now (`7d`, `30d`, `6m`, `1y`, `3y`) |
| `--since` | | `""` | Filter PRs created on or after specific date in `YYYY-MM-DD` format |
| `--until` | | `""` | Filter PRs created on or before specific date in `YYYY-MM-DD` format |

#### PR Filtering & Search Flags
| Flag | Short | Default | Description |
|---|---|---|---|
| `--author` | `-a` | `""` | Filter PRs created by specific author username (e.g., `octocat` or `@octocat`) |
| `--reviewer` | `-r` | `""` | Filter PRs reviewed by specific user who submitted a review (e.g., `mona` or `@mona`) |
| `--assignee` | | `""` | Filter PRs assigned to specific user |
| `--review-requested` | | `""` | Filter PRs with pending review request for specific user |
| `--state` | `-s` | `all` | Filter PRs by lifecycle state: `all` (default), `merged`, `open`, `closed` |
| `--draft` | | `all` | Filter PRs by draft status: `all` (default), `true` (drafts only), `false` (ready for review only) |
| `--review-state` | | `all` | Filter PRs by review verdict: `all`, `approved`, `changes_requested`, `commented`, `none` |
| `--label` | `-l` | `""` | Filter PRs matching comma-separated labels (AND matching, e.g., `bug,frontend`) |
| `--base` | `-b` | `""` | Filter PRs targeting specific base branch (e.g., `main`) |
| `--head` | | `""` | Filter PRs originating from specific source/head branch pattern (e.g., `feature/auth`) |
| `--milestone` | | `""` | Filter PRs assigned to specific milestone title |
| `--checks` | | `all` | Filter PRs by CI/CD status: `all`, `success`, `failure`, `pending` |
| `--has-conflicts` | | `all` | Filter PRs by merge conflict status: `all`, `true`, `false` |
| `--min-lines` / `--max-lines` | | `0` | Filter PRs by total diff size (additions + deletions) threshold |
| `--min-files` / `--max-files` | | `0` | Filter PRs by modified file count threshold |
| `--search` | `-S` | `""` | Search keywords in PR title, body, or head branch |

#### Aggregation & Grouping Flags
| Flag | Short | Default | Description |
|---|---|---|---|
| `--group-by` | `-g` | `none` | Aggregation dimension: `day`, `week`, `month`, `quarter`, `year`, `author`, `reviewer`, `label`, `base`, `size` |

#### Output & Formatting Flags
| Flag | Short | Default | Description |
|---|---|---|---|
| `--format` | `-f` | `text` | Output rendering format: `text` (table), `json`, `csv`, `tsv`, `markdown` |
| `--json` | | `false` | Shortcut for `--format json` |
| `--csv` | | `false` | Shortcut for `--format csv` |
| `--tsv` | | `false` | Shortcut for `--format tsv` |
| `--markdown` | | `false` | Shortcut for `--format markdown` |
| `--percentiles` | | `false` | Display extended statistical percentiles breakdown (p50, p75, p90, p95, p99) in table output |
| `--detailed` | | `false` | Include detailed list of individual matching PR records alongside summary |

### 3. Cache Flags
Cache commands (`cache list`, `cache status`, `cache migrate`, `cache clean`, `cache path`) only accept cache-specific flags and inherited **Global Flags** (`--repo`, `--no-cache`, `--verbose`, `--page-size`). They do **not** accept PR filtering, timing, or metric grouping flags.

| Subcommand | Flag | Default | Description |
|---|---|---|---|
| `cache list` | `--json` | `false` | Output cached repository list in JSON format |
| `cache status` | `--json` | `false` | Output cache health and migration status in JSON format |
| `cache migrate` | `--all` | `false` | Migrate all outdated cached repository records |
| `cache migrate` | `--dry-run` | `false` | Preview cache migration without modifying files on disk |
| `cache clean` | `--all` | `false` | Clear all cached repository records |
| `cache clean` | `--outdated` | `false` | Clear only outdated cache entries requiring refresh |
| `cache clean` | `--corrupt` | `false` | Clear only corrupted cache files with invalid JSON |

---

## Command Hierarchy & Reference

```
gh pr-pro
├── time
│   ├── merge         # Cycle time from creation to merge
│   ├── review        # TTFR and reviewer turnaround latency
│   ├── draft         # Duration spent in draft mode
│   ├── pickup        # Queue time from review request to first review
│   └── idle          # Inactivity / waiting duration
├── quality
│   ├── ci            # CI/CD check runs, failure rates, and retries
│   ├── rework        # Review roundtrips and post-review churn
│   ├── conflicts     # Merge conflicts frequency and resolution time
│   └── reverts       # Post-merge revert rate and time-to-revert
├── code
│   ├── size          # Lines added/deleted, files changed, sizing
│   └── commits       # Commit count per PR and commit distribution
├── team
│   ├── throughput    # PR velocity (opened vs merged vs closed)
│   ├── reviews       # Review distribution and workload balance
│   └── unreviewed    # Self-merged / bypass review percentage
├── overview          # Composite executive scorecard
├── export            # Raw data streaming (JSON/CSV)
└── cache             # Manage local persistent disk cache
    ├── list          # List cached repositories and disk sizes
    ├── status        # Inspect cache health, schema versions, and migration guidance
    ├── migrate       # Upgrade outdated cache files to current schema version
    ├── clean         # Clear cache for a repo, outdated, corrupt, or all repos
    └── path          # Print local cache filesystem path
```

---

### 1. `gh pr-pro time` (Lifecycle & Latency)

#### `time merge`
Measures total cycle time from creation/ready-for-review to merge.

```bash
gh pr-pro time merge --past 90d --group-by month
```

```text
📊 TIME MERGE Analytics
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Group            Count    p50 (Median) p75          p90          Mean      
─────────────────────────────────────────────────────────────
2025-06          62       14.2h        26.0h        48.5h        21.3h     
2025-07          58       11.5h        19.8h        38.2h        17.1h     
2025-08          64       16.0h        31.4h        54.0h        24.6h     
─────────────────────────────────────────────────────────────
Summary: 184 PRs analyzed | p50: 13.8h | p75: 25.1h | p90: 46.4h | Mean: 21.0h
```

#### `time review`
Measures Time to First Review (TTFR) and reviewer response turnaround.

```bash
gh pr-pro time review --past 30d --group-by reviewer
```

```text
📊 TIME REVIEW Analytics
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Group            Count    p50 (Median) p75          p90          Mean      
─────────────────────────────────────────────────────────────
@alice           42       1.4h         2.5h         5.2h         1.8h      
@bob             31       2.8h         5.0h         11.0h        3.2h      
@carol           19       6.1h         12.0h        24.0h        7.5h      
─────────────────────────────────────────────────────────────
Summary: 92 PRs analyzed | p50: 2.2h | p75: 4.8h | p90: 12.0h | Mean: 3.5h
```

#### `time draft`
Measures how long PRs remain in draft status before ready for review.

```bash
gh pr-pro time draft --past 6m --group-by size
```

#### `time pickup`
Measures queue time between review requests and first reviewer action.

```bash
gh pr-pro time pickup --past 30d
```

#### `time idle`
Breaks down inactivity duration (waiting on author vs waiting on reviewer).

```bash
gh pr-pro time idle --past 60d
```

---

### 2. `gh pr-pro quality` (CI/CD, Churn & Stability)

#### `quality ci`
Analyzes CI/CD check suite stability, failure rates, and rerun counts.

```bash
gh pr-pro quality ci --past 60d --group-by week
```

```text
📊 QUALITY CI Analytics
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Group            Count    p50 (Median) p75          p90          Mean      
─────────────────────────────────────────────────────────────
2025-W32         45       22.2%        22.2%        22.2%        22.2%     
2025-W33         51       15.7%        15.7%        15.7%        15.7%     
2025-W34         48       27.1%        27.1%        27.1%        27.1%     
─────────────────────────────────────────────────────────────
Summary: 144 PRs analyzed | p50: 21.5% | p75: 21.5% | p90: 21.5% | Mean: 21.5%

Top Failing Check Suites:
  • e2e-tests: 6 failures
  • lint-and-typecheck: 4 failures
```

#### `quality rework`
Measures review roundtrips and commits pushed after the initial review.

```bash
gh pr-pro quality rework --past 90d --group-by author
```

#### `quality conflicts`
Tracks PRs experiencing merge conflicts and resolution overhead.

```bash
gh pr-pro quality conflicts --past 6m
```

#### `quality reverts`
Tracks PRs reverted after merge and time elapsed before reversion.

```bash
gh pr-pro quality reverts --past 1y
```

---

### 3. `gh pr-pro code` (Size, Commits & Scope)

#### `code size`
Evaluates line additions, deletions, modified files, and size distributions (S/M/L/XL).

```bash
gh pr-pro code size --past 30d --group-by month
```

#### `code commits`
Measures commit volume per PR and evaluates pull request atomicity.

```bash
gh pr-pro code commits --past 90d
```

---

### 4. `gh pr-pro team` (Throughput, Load & Flow)

#### `team throughput`
Tracks PR velocity: opened vs merged vs closed PRs and net backlog delta.

```bash
gh pr-pro team throughput --past 1y --group-by month
```

#### `team reviews`
Analyzes review workload distribution across team members.

```bash
gh pr-pro team reviews --past 90d
```

#### `team unreviewed`
Identifies PRs merged without external reviews or bypassing review policies.

```bash
gh pr-pro team unreviewed --past 6m
```


---

### 5. Root Commands

#### `overview`
Displays an all-in-one executive scorecard combining key metrics across all domains.

```bash
gh pr-pro overview --past 30d
```

#### `export`
Streams unaggregated, per-PR raw records containing all metric dimensions for downstream BI, DuckDB, or Pandas ingestion.

> [!NOTE]
> **Automation & Agent Guidance**:
> * **Default Time Window**: Defaults to `--past 30d`. For broader historical exports, explicitly specify `--past <duration>` (e.g., `--past 1y`) or `--since <YYYY-MM-DD>`.
> * **Avoid for Aggregate Metrics**: Do not use `export` to compute percentiles, cycle times, or CI failure rates. Use domain commands (`overview`, `time`, `quality`, `code`, `team`) with `--json` which calculate statistical percentiles and aggregations natively.

```bash
gh pr-pro export --past 1y --csv > pr_export_2025.csv
gh pr-pro export --since 2025-01-01 --until 2025-06-30 --json > pr_export_h1.json
```

#### `cache`
Inspect, diagnose, migrate, and manage the local PR disk cache.

```bash
# List all cached repositories and disk consumption
gh pr-pro cache list

# Check cache health, schema versions, and migration guidance
gh pr-pro cache status

# Output cache status in JSON format
gh pr-pro cache status --json

# Migrate all outdated cache files to the current schema version
gh pr-pro cache migrate --all

# Preview cache migration for a specific repository
gh pr-pro cache migrate cli/cli --dry-run

# Clear only outdated cache entries
gh pr-pro cache clean --outdated

# Clear only corrupted cache files with invalid JSON
gh pr-pro cache clean --corrupt

# Clear cache for a specific repository
gh pr-pro cache clean cli/cli

# Clear all cached repositories
gh pr-pro cache clean --all

# Print the local cache directory path
gh pr-pro cache path
```

---

## Output Schemas

### JSON Output
When passing `--json` (e.g., `gh pr-pro time merge --past 30d --group-by week --json`):

```json
{
  "domain": "time",
  "metric": "merge",
  "time_window": {
    "since": "2025-08-03T00:00:00Z",
    "until": "2025-09-02T00:00:00Z",
    "past": "30d"
  },
  "summary": {
    "group_key": "Total",
    "count": 184,
    "p50": 49680,
    "p75": 90360,
    "p90": 167040,
    "p95": 216000,
    "p99": 288000,
    "mean": 75600,
    "min": 3600,
    "max": 324000,
    "unit": "seconds"
  },
  "groups": [
    {
      "group_key": "2025-W32",
      "count": 45,
      "p50": 46800,
      "p75": 82800,
      "p90": 154800,
      "p95": 198000,
      "p99": 252000,
      "mean": 71200,
      "min": 3600,
      "max": 288000,
      "unit": "seconds"
    }
  ]
}
```

### CSV Output
When passing `--csv` (e.g., `gh pr-pro time merge --past 30d --group-by week --csv`):

```csv
group_key,count,p50,p75,p90,p95,p99,mean,min,max,unit
2025-W32,45,13.00,23.00,43.00,55.00,70.00,19.78,1.00,80.00,seconds
2025-W33,51,11.50,19.80,38.20,48.00,62.00,17.10,0.50,75.00,seconds
2025-W34,48,16.00,31.40,54.00,68.00,85.00,24.60,2.00,90.00,seconds
```

---

## Rate Limits & GraphQL Query Economics

GitHub's GraphQL API uses a **point-based rate limit system** and enforces strict backend execution time limits:

### How Point Calculations Work
* **Hourly Quota**:
  * **5,000 points/hour** for personal user tokens and standard GitHub CLI logins (`gh auth login`).
  * **10,000 points/hour** for GitHub Enterprise Cloud and GitHub Apps.
* **Inspecting Your Live Quota**:
  You can check your remaining points, consumption, and reset timestamp directly using GitHub CLI:
  ```bash
  # Check overall rate limit quota across all GitHub API resources
  gh api rate_limit

  # Check GraphQL API quota only (used by gh-pr-pro)
  gh api rate_limit --jq '.resources.graphql'
  ```
* **Point Cost per Query**:
  * GitHub evaluates the total number of node connections and nested sub-fields requested.
  * In `gh-pr-pro`, each page request fetches rich PR metadata (commits, CI status check rollups, reviews, review requests, and draft timeline events).
  * With `--page-size 25`, each query batch costs approximately **2 points** (returned in `rateLimit { cost remaining resetAt }`).
* **Effective Query Volume**:
  $$\frac{5{,}000 \text{ points/hour}}{\sim 2 \text{ points/query}} \approx \mathbf{2{,}500 \text{ pages/hour}} \ (\approx \mathbf{62{,}500 \text{ PRs/hour}})$$
* **Secondary Rate Limits**: GitHub limits backend CPU execution time to **60 seconds of computation per 60-second sliding window**. `gh-pr-pro` includes exponential backoff ($2\text{s} \to 4\text{s} \to 8\text{s}$) to avoid triggering burst throttling.

---

### The 10-Second GraphQL Execution Limit & Page Size Tuning

> [!NOTE]
> **Quota vs. Execution Timeout**: An hourly rate limit exhaustion returns `HTTP 403 / 429`, whereas an individual query execution timeout returns `HTTP 504 Gateway Timeout`. Even with thousands of quota points remaining (checked via `gh api rate_limit`), an individual query can still time out if it takes longer than 10 seconds on GitHub's backend.

GitHub enforces a hard **10-second backend execution timeout** on any individual GraphQL query.

#### Why Query Complexity Matters
Each PR node requests nested sub-trees:
* Up to 30 reviews (including bodies and timestamps)
* Up to 20 CI check run / status context entries per latest commit
* Up to 10 review requests and 30 draft conversion timeline events

In large repositories with thousands of PRs and massive CI matrices (e.g. `cli/cli`, `kubernetes/kubernetes`), requesting `100` PRs at once requires GitHub's database to resolve tens of thousands of joined records in a single query. When resolution exceeds 10 seconds, GitHub's edge proxy terminates the connection with an `HTTP 504 Gateway Timeout`.

#### Page Size Recommendations & Trade-offs

| Page Size (`--page-size`) | Typical Latency | Point Cost | Recommended Repository Profile | 10s Timeout (HTTP 504) Risk |
|---|---|---|---|---|
| **`10 – 25` (Default: `25`)** | **1.0s – 2.5s** | **~2 points** | Large repositories, monorepos, high CI check volume | **Near Zero** |
| **`30 – 50`** | **2.5s – 5.0s** | **~3–4 points** | Medium repositories with moderate review activity | Low |
| **`75 – 100`** | **6.0s – 12.0s+** | **~5–7 points** | Small or low-activity repositories only | **High on large repos** |

#### Automatic Adaptive Fallback
If a query encounters transient timeouts or 502/504 errors, `gh-pr-pro` automatically:
1. Retries up to 3 times with exponential backoff ($2\text{s} \to 4\text{s} \to 8\text{s}$).
2. Halves the active `pageSize` (e.g. from 100 down to 50, 25, or 10) to recover automatically without user intervention.

---

### Troubleshooting & Common Issues

#### ⚠️ `GitHub API request throttled or transient error: HTTP 504`
* **Cause**: An individual GraphQL query exceeded GitHub's 10-second backend computation deadline. This is a per-request execution limit, not an hourly quota limit.
* **Remediation**:
  1. **Lower Page Size**: Run with `--page-size 25` (or `--page-size 10` for extremely dense monorepos):
     ```bash
     gh pr-pro overview --repo cli/cli --page-size 25
     ```
  2. **Enable Disk Caching (Omit `--no-cache`)**: Allow `gh-pr-pro` to store local cache. Subsequent runs will only query recent updates via delta sync instead of paginating complete history.
  3. **Narrow Time Window**: Use `--past 30d` or `--since YYYY-MM-DD` to reduce the number of historical records evaluated.

#### ⚠️ `Rate Limit Exceeded (HTTP 403 / 429)`
* **Cause**: Your account's 5,000 points/hour GraphQL quota has been exhausted.
* **Remediation**:
  1. **Check Quota & Reset Time**:
     ```bash
     gh api rate_limit --jq '.resources.graphql'
     ```
  2. **Rely on Local Cache**: `gh-pr-pro` automatically falls back to locally cached PR records if the API is exhausted.
  3. **Wait for Reset**: Points replenish automatically at the `reset` timestamp.

---

## Caching & Offline Resilience

`gh-pr-pro` features an intelligent local disk cache designed to make querying instant and resilient—even when approaching or hitting GitHub API rate limits:

### How Caching Overcomes Rate Limits
* **Incremental Delta Sync (0–1 API Calls)**:
  * Merged and closed PRs are immutable. `gh-pr-pro` tracks the `last_fetched` timestamp in `~/.cache/gh-pr-pro/<owner>_<repo>.json`.
  * On repeated queries, `gh-pr-pro` only fetches PRs created or updated *since* that timestamp. Analyzing 1 year of data usually requires **0 to 1 API call** instead of dozens of roundtrips.
* **Progressive Hydration Across Rate Limits**:
  * If a large historical fetch on an active repo is stopped by a rate limit, all successfully fetched pages are safely saved to local disk cache.
  * Re-running the command after the quota reset picks up directly from where it stopped via delta sync—**never re-fetching historical records twice**.
* **Graceful Degradation / Fallback on API Exhaustion**:
  * If GitHub returns an `HTTP 403` / `429 Rate Limit Exceeded` error or is temporarily unreachable, `gh-pr-pro` automatically falls back to your local cached dataset to complete calculations and render scorecards rather than crashing.
* **Proactive Quota Safety**:
  * When remaining quota drops to critical thresholds ($\le 10$ points), `gh-pr-pro` automatically pauses until `resetAt` before resuming pagination.

### Cache Management & Storage
* **Location**:
  * macOS / Linux: `~/.cache/gh-pr-pro/<owner>_<repo>.json`
  * Windows: `%USERPROFILE%\.cache\gh-pr-pro\<owner>_<repo>.json`
* **CLI Cache Commands**:
  ```bash
  gh pr-pro cache list               # View all cached repos, record counts, and disk usage
  gh pr-pro cache list --json        # Export cache metadata in JSON
  gh pr-pro cache clean <owner/repo> # Delete cache for specific repo
  gh pr-pro cache clean --all        # Delete all cache files
  gh pr-pro cache path               # Print cache directory
  ```
* **Bypassing Cache**: Use `--no-cache` on any command to bypass disk storage and query GitHub directly.

---

## Contributing & License

- To contribute to `gh-pr-pro`, see the [Contributing Guide](CONTRIBUTING.md).
- Please review our [Code of Conduct](CODE_OF_CONDUCT.md) before participating.
- For security vulnerability disclosures, refer to our [Security Policy](SECURITY.md).
- Distributed under the [MIT License](LICENSE).
