# AGENTS.md — Agent & Contributor Guidance for `gh-pr-pro`

This document provides architectural context, commands, coding standards, and invariant rules for AI agents and automated tools contributing to the `gh-pr-pro` codebase.

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

## 2. Common Developer & Agent Commands

Agents should execute standard `make` targets or native `go` commands:

```bash
# Pre-commit check: formats code, runs static analysis, audits README sync, and runs tests
make check

# Audit README synchronization against CLI API surface
make audit-readme
# or directly:
./.agents/skills/readme-api-sync/scripts/audit_readme.sh

# Run all unit tests with race detection (standard test invocation)
go test -race -count=1 ./...

# Run unit tests with verbose subtest output
go test -v -race -count=1 ./...

# Run specific package tests
go test -v -race ./pkg/metrics/...

# View test coverage summary in terminal
make cover-text

# Format Go code
gofmt -s -w .

# Run static analysis
go vet ./...

# Build the local binary
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

### 5. Mandatory README Synchronization on API Surface Changes
Whenever any change is made to the CLI API surface area (commands, subcommands, flags, shorthands, default values, filter parameters, or output schemas):
- The agent **MUST** invoke and run the [`readme-api-sync`](file:///Users/brad/Projects/gh-pr-pro/.agents/skills/readme-api-sync/SKILL.md) skill.
- Review and update [`README.md`](file:///Users/brad/Projects/gh-pr-pro/README.md) to keep flag scoping tables, command hierarchy trees, and subcommand references 100% in line with the code.
- Verify synchronization by running `make audit-readme` (or `./.agents/skills/readme-api-sync/scripts/audit_readme.sh`).
- Never conclude a change with un-synchronized CLI docs.

---

## 4. How to Add a New Metric Subcommand

When adding a metric subcommand (e.g., `gh pr-pro quality security`):

1. **Schema**: Add any required calculated fields to `ProcessedPR` in [`pkg/metrics/types.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/metrics/types.go).
2. **GraphQL Query & Model**: If new GraphQL fields are required, update query constants and struct tags in [`pkg/api/models.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/api/models.go).
3. **Calculation**: Update [`ProcessPRNode`](file:///Users/brad/Projects/gh-pr-pro/pkg/metrics/calculator.go) to compute and populate the metric.
4. **Aggregation**: Update [`computeMetricStats`](file:///Users/brad/Projects/gh-pr-pro/pkg/metrics/aggregation.go) to extract metric values and attach percentiles.
5. **CLI Registration**: In `pkg/cmd/<domain>.go`, declare the subcommand calling `RunMetricCommand(cmd, "<domain>", "<metric>")` and register it in `init()`.
6. **Tests**: Add table-driven unit tests in `pkg/metrics/*_test.go` and command flag tests in `pkg/cmd/root_test.go`.
7. **Documentation**: Invoke the [`readme-api-sync`](file:///Users/brad/Projects/gh-pr-pro/.agents/skills/readme-api-sync/SKILL.md) skill to review and update [`README.md`](file:///Users/brad/Projects/gh-pr-pro/README.md) and [`CONTRIBUTING.md`](file:///Users/brad/Projects/gh-pr-pro/CONTRIBUTING.md) command reference tables, and verify with `make audit-readme`.

---

## 5. Testing & Code Quality Expectations

- **Table-Driven Tests**: Write tests using `[]struct{ name string; ... }` with `t.Run(tt.name, func(t *testing.T) { ... })`.
- **Race Detector**: All tests must pass with `go test -race ./...`.
- **Formatting**: Always execute `gofmt -s -w .` before concluding any edit.
