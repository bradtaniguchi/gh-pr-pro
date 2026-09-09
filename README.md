# `gh-pr-pro`

> Advanced Pull Request Analytics & Engineering Intelligence extension for the GitHub CLI (`gh`).

[![GitHub CLI Extension](https://img.shields.io/badge/gh-extension-blue.svg)](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

`gh-pr-pro` organizes pull request analytics into structured top-level domains (`time`, `quality`, `code`, `team`) with dedicated metric subcommands. It delivers statistical percentiles (p50, p75, p90, p99), historical trends, and uniform multi-format exports (`text`, `json`, `csv`, `tsv`, `markdown`).

> [!WARNING]
> **GitHub API Rate Limits**: Analyzing large repositories or expansive historical time ranges queries GitHub's GraphQL API extensively. While `gh-pr-pro` incorporates automatic backoff and persistent local disk caching to minimize requests, queries are subject to GitHub's primary point quotas and secondary computation limits. See [Rate Limits & GraphQL Query Economics](#rate-limits--graphql-query-economics) for quota economics and mitigation details.

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

## Flag Scoping & Reference

`gh-pr-pro` uses scoped command flags to ensure clean and intuitive CLI interactions. Flags are structured into **Global Flags**, **Domain / Metric Flags**, and **Cache Flags**:

### 1. Global Flags
Available across **all** commands and subcommands (including `time`, `quality`, `code`, `team`, `overview`, `export`, and `cache`):

| Flag | Short | Default | Description |
|---|---|---|---|
| `--repo` | `-R` | current | Target repository in `[HOST/]OWNER/REPO` format (defaults to current git repository) |
| `--no-cache` | | `false` | Bypass local disk cache (`~/.cache/gh-pr-pro/`) and force a complete fresh fetch from GitHub API |
| `--verbose` | `-v` | `false` | Print verbose progress, timestamped network latency, and cache activity logs to stderr |
| `--page-size` | | `100` | GraphQL page size for PR queries (1 to 100 max) |

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
Cache commands (`cache list`, `cache clean`, `cache path`) only accept cache-specific flags and inherited **Global Flags** (`--repo`, `--no-cache`, `--verbose`, `--page-size`). They do **not** accept PR filtering, timing, or metric grouping flags.

| Subcommand | Flag | Default | Description |
|---|---|---|---|
| `cache list` | `--json` | `false` | Output cached repository list in JSON format |
| `cache clean` | `--all` | `false` | Clear all cached repository records |

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
│   ├── commits       # Commits per PR, force pushes, rebase ratios
│   └── hotspots      # Files and paths modified most frequently
├── team
│   ├── throughput    # PR velocity (opened vs merged vs closed)
│   ├── reviews       # Review distribution and workload balance
│   ├── unreviewed    # Self-merged / bypass review percentage
│   └── abandonment   # Unmerged closed PR drop-off stats
├── overview          # Composite executive scorecard
├── export            # Raw data streaming (JSON/CSV)
└── cache             # Manage local persistent disk cache
    ├── list          # List cached repositories and disk sizes
    ├── clean         # Clear cache for a repo or all repos
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
⏱️  MERGE TIME (Cycle Time) — past 90d (184 merged PRs)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Period     Count   p50 (Median)   p75       p90       Mean
─────────────────────────────────────────────────────────────
2025-06    62      14.2h          26.0h     48.5h     21.3h
2025-07    58      11.5h          19.8h     38.2h     17.1h
2025-08    64      16.0h          31.4h     54.0h     24.6h
─────────────────────────────────────────────────────────────
Total      184     13.8h          25.1h     46.4h     21.0h
```

#### `time review`
Measures Time to First Review (TTFR) and reviewer response turnaround.

```bash
gh pr-pro time review --past 30d --group-by reviewer
```

```text
👥 REVIEW TURNAROUND — past 30d (128 PRs reviewed)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Reviewer    Reviews   Avg TTFR   p50 Resp   p90 Resp   Approvals / Changes
─────────────────────────────────────────────────────────────
@alice      42        1.8h       1.4h       5.2h       36 / 6
@bob        31        3.2h       2.8h       11.0h      24 / 7
@carol      19        7.5h       6.1h       24.0h      17 / 2
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
🚦 CI/CD HEALTH & STABILITY — past 60d
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Week        PRs Tested   Failed Checks %   Avg Retries   Top Failing Suite
─────────────────────────────────────────────────────────────
2025-W32    45           22.2% (10)        1.4           e2e-tests (6)
2025-W33    51           15.7% (8)         1.1           lint-and-typecheck (4)
2025-W34    48           27.1% (13)        1.8           integration-tests (9)
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
Measures commit volume per PR, force pushes, and rebase/squash ratios.

```bash
gh pr-pro code commits --past 90d
```

#### `code hotspots`
Identifies files and directories with the highest PR modification frequency.

```bash
gh pr-pro code hotspots --past 6m --top 10
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

#### `team abandonment`
Analyzes PRs closed without merging and where in the lifecycle they stalled.

```bash
gh pr-pro team abandonment --past 1y
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

```bash
gh pr-pro export --past 1y --csv > pr_export_2025.csv
gh pr-pro export --since 2025-01-01 --until 2025-06-30 --json > pr_export_h1.json
```

#### `cache`
Inspect and manage the local PR disk cache.

```bash
# List all cached repositories and disk consumption
gh pr-pro cache list

# Output cached repositories in JSON format
gh pr-pro cache list --json

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
    "until": "2025-09-02T00:00:00Z"
  },
  "summary": {
    "total_prs": 184,
    "p50_seconds": 49680,
    "p75_seconds": 90360,
    "p90_seconds": 167040,
    "mean_seconds": 75600
  },
  "groups": [
    {
      "group_key": "2025-W32",
      "count": 45,
      "p50_seconds": 46800,
      "p75_seconds": 82800,
      "p90_seconds": 154800,
      "mean_seconds": 71200
    }
  ]
}
```

### CSV Output
When passing `--csv` (e.g., `gh pr-pro time merge --past 30d --group-by week --csv`):

```csv
group_key,count,p50_hours,p75_hours,p90_hours,mean_hours
2025-W32,45,13.00,23.00,43.00,19.78
2025-W33,51,11.50,19.80,38.20,17.10
2025-W34,48,16.00,31.40,54.00,24.60
```

---

## Rate Limits & GraphQL Query Economics

GitHub's GraphQL API uses a **point-based rate limit system** rather than simple request counting:

### How Point Calculations Work
* **Hourly Quota**:
  * **5,000 points/hour** for personal user tokens and standard GitHub CLI logins (`gh auth login`).
  * **10,000 points/hour** for GitHub Enterprise Cloud and GitHub Apps.
* **Query Cost Calculation**:
  * GitHub evaluates the total number of node connections and nested sub-fields requested in the query.
  * In `gh-pr-pro`, each page request fetches up to 100 PRs with deeply nested sub-entities (commits, CI status check rollups, reviews, review requests, and draft timeline events).
  * Each paginated query batch costs approximately **4 to 7 points** (returned in `rateLimit { cost remaining resetAt }`).
* **Effective Query Volume**:
  $$\frac{5{,}000 \text{ points/hour}}{\sim 5\text{--}7 \text{ points/query}} \approx \mathbf{700\text{ to }1{,}000 \text{ pages/hour}} \ (\approx \mathbf{70{,}000\text{ to }100{,}000 \text{ PRs/hour}})$$
* **Secondary Rate Limits**: GitHub limits backend CPU execution time to **60 seconds of computation per 60-second sliding window**. `gh-pr-pro` includes exponential backoff ($2\text{s} \to 4\text{s} \to 8\text{s}$) to avoid triggering burst throttling.

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
