# AGENTS.md — Agent & Contributor Guidance for `gh-pr-pro`

This document provides architectural context, commands, and invariant rules for AI agents and automated tools contributing to the `gh-pr-pro` codebase.

---

## 1. Project Overview & Architecture

`gh-pr-pro` is an engineering intelligence extension for the GitHub CLI (`gh`) written in Go. It queries pull request metadata via GitHub's GraphQL API, calculates statistical percentiles (p50, p75, p90, p95, p99), maintains an incremental local disk cache, and renders metrics across multiple formats (`text`, `json`, `csv`, `tsv`, `markdown`).

### Core Packages & Responsibilities

| Package | Responsibility |
|---|---|
| [`main.go`](file:///Users/brad/Projects/gh-pr-pro/main.go) | Application entrypoint; calls [`cmd.Execute()`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/root.go). |
| [`pkg/api`](file:///Users/brad/Projects/gh-pr-pro/pkg/api) | GitHub GraphQL communication via `cli/go-gh/v2`. Implements rate-limit budget tracking, delta pagination, and backoff. |
| [`pkg/cache`](file:///Users/brad/Projects/gh-pr-pro/pkg/cache) | Disk cache stored under `~/.cache/gh-pr-pro/<owner>_<repo>.json`. Merges cached PRs with newly fetched deltas and provides offline fallback. |
| [`pkg/cmd`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd) | Cobra command tree (`time`, `quality`, `code`, `team`, `overview`, `export`, `cache`), flag parsing, and execution pipeline. |
| [`pkg/metrics`](file:///Users/brad/Projects/gh-pr-pro/pkg/metrics) | Transforms raw GraphQL nodes into [`ProcessedPR`](file:///Users/brad/Projects/gh-pr-pro/pkg/metrics/types.go), computes lifecycle durations, applies filter matrices, and aggregates percentiles across dimensions. |
| [`pkg/output`](file:///Users/brad/Projects/gh-pr-pro/pkg/output) | Serializes [`MetricOutput`](file:///Users/brad/Projects/gh-pr-pro/pkg/metrics/types.go) into human-readable text tables, JSON, CSV, TSV, or Markdown. |

---

## 2. Common Commands

Run standard `make` targets or native `go` commands:

```bash
# Pre-commit verification: module tidy check, static analysis, race-detected tests, and audits
make check

# Configure Git pre-commit hooks (.githooks)
make init-hooks

# Run all unit tests with race detection
go test -race -count=1 ./...

# Run specific package tests
go test -v -race ./pkg/metrics/...

# View test coverage summary in terminal
make cover-text

# Format Go code (strictly enforces gofumpt via golangci-lint)
make fmt

# Run comprehensive static analysis
make lint

# Build local binary
make build
```

---

## 3. Key Invariants & Architectural Rules

### 1. Flag Scoping Rules
- **Global Flags** (`--repo`/`-R`, `--no-cache`, `--verbose`/`-v`, `--page-size`) are registered as persistent flags on `RootCmd`.
- **Metric/Filter/Output Flags** (`--past`, `--since`, `--group-by`, `--json`, etc.) must **only** attach to metric domains (`time`, `quality`, `code`, `team`), `overview`, and `export`.
- **Cache Commands** (`cache list`, `cache clean`, `cache path`) must **never** register or accept metric/filtering flags. Enforced by unit tests in [`pkg/cmd/cache_test.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/cache_test.go).

### 2. Rate Limit & Offline Fallback Safety
- Always calculate PR updates using delta timestamps (`last_fetched` in local cache).
- Never fail hard on network disconnection or rate limits if cached data exists; fall back gracefully to cached records as implemented in [`FetchAndProcessPRs`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/root.go).
- Paginated queries must remain budgeted to max 100 items per page.

### 3. Metric Calculations & Zero-Panic Policy
- All duration calculations in [`pkg/metrics/calculator.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/metrics/calculator.go) are represented as `float64` seconds.
- Handle `nil` timestamps gracefully (e.g., draft PRs without `ready_for_review_at`, unmerged PRs without `merged_at`).
- **Never call `panic()`** in production code. Return typed or wrapped errors (`fmt.Errorf("...: %w", err)`).

### 4. Grouping & Dimensions
Supported `--group-by` dimensions are: `none`, `day`, `week`, `month`, `quarter`, `year`, `author`, `reviewer`, `label`, `base`, `size`. When adding new metrics, ensure [`pkg/metrics/aggregation.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/metrics/aggregation.go) correctly formats summary and group records.

### 5. Table-Driven Tests
- Use Go's standard table-driven testing pattern (`[]struct{ name string; ... }` with `t.Run(...)`) across all packages.
- Ensure all tests pass with the race detector enabled (`go test -race ./...`).
