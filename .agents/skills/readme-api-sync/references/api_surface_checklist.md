# API Surface to README Mapping & Review Checklist

This document details the exact mapping between `gh-pr-pro` Go source files representing the CLI API surface area and the corresponding sections in [`README.md`](../../../../README.md). Use this checklist during any API surface review.

---

## 1. Source File to README Mapping

| Source File | API Surface Component | Corresponding `README.md` Section |
|---|---|---|
| [`pkg/cmd/root.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/root.go) | Root command, Global Flags (`--repo`, `--no-cache`, `--verbose`, `--page-size`) | `## Quick Start`, `### 1. Global Flags` |
| [`pkg/cmd/root.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/root.go) (`attachTimeFlags`) | Time range flags (`--past`, `--since`, `--until`) | `### 2. Domain / Metric Flags` -> `#### Time Window Flags` |
| [`pkg/cmd/root.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/root.go) (`attachFilterFlags`) | PR filtering & search flags (`--search`, `--author`, `--reviewer`, `--assignee`, `--review-requested`, `--state`, `--draft`, `--review-state`, `--label`, `--base`, `--head`, `--milestone`, `--checks`, `--has-conflicts`, `--min-lines`, `--max-lines`, `--min-files`, `--max-files`) | `### 2. Domain / Metric Flags` -> `#### PR Filtering & Search Flags` |
| [`pkg/cmd/root.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/root.go) (`attachOutputFlags`) | Output formatting & aggregation flags (`--format`, `--json`, `--csv`, `--tsv`, `--markdown`, `--percentiles`, `--detailed`, `--group-by`) | `### 2. Domain / Metric Flags` -> `#### Aggregation & Grouping Flags`, `#### Output & Formatting Flags` |
| [`pkg/cmd/time.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/time.go) | `time` domain: `merge`, `review`, `draft`, `pickup`, `idle` | `## Command Hierarchy & Reference`, `### 1. gh pr-pro time (Lifecycle & Latency)` |
| [`pkg/cmd/quality.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/quality.go) | `quality` domain: `ci`, `rework`, `conflicts`, `reverts` | `## Command Hierarchy & Reference`, `### 2. gh pr-pro quality (Code Quality & Review Health)` |
| [`pkg/cmd/code.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/code.go) | `code` domain: `size`, `commits`, `hotspots` | `## Command Hierarchy & Reference`, `### 3. gh pr-pro code (Size, Commits & Complexity)` |
| [`pkg/cmd/team.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/team.go) | `team` domain: `throughput`, `reviews`, `unreviewed`, `abandonment` | `## Command Hierarchy & Reference`, `### 4. gh pr-pro team (Throughput, Reviews & Workload)` |
| [`pkg/cmd/overview.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/overview.go) | `overview` composite scorecard | `## Command Hierarchy & Reference`, `### 5. gh pr-pro overview` |
| [`pkg/cmd/export.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/export.go) | `export` raw data streamer | `## Command Hierarchy & Reference`, `### 6. gh pr-pro export` |
| [`pkg/cmd/cache.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/cmd/cache.go) | `cache` management: `list`, `clean`, `path` and specific flags (`--all`, `--json`) | `### 3. Cache Flags`, `### 7. gh pr-pro cache` |
| [`pkg/output/formatter.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/output/formatter.go) | Render formats (`text`, `json`, `csv`, `tsv`, `markdown`) | `#### Output & Formatting Flags`, `## Output Formats` |
| [`pkg/metrics/types.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/metrics/types.go) | Group-by dimensions, percentiles, calculated metrics | `#### Aggregation & Grouping Flags`, individual metric calculation sections |

---

## 2. API Surface Change Review Checklist

When an API surface change is detected, evaluate each item:

### A. Subcommands & Command Tree
- [ ] Has any subcommand been added, renamed, aliased, or removed?
- [ ] Is the ASCII diagram under `## Command Hierarchy & Reference` updated?
- [ ] Is there a dedicated section with usage syntax, description, and copy-pasteable examples for the command?
- [ ] If this is a major workflow addition, is it showcased in `## Quick Start`?

### B. Flags & Arguments
- [ ] Has any flag been added, renamed, deprecated, or removed?
- [ ] Is the flag placed in the correct scoping table?
  - Global flag: `### 1. Global Flags`
  - Metric domain flag: `### 2. Domain / Metric Flags`
  - Cache command flag: `### 3. Cache Flags`
- [ ] Does the table accurately state:
  - **Flag Name**: e.g., `--new-flag`
  - **Short Flag**: e.g., `-n` (or blank if none)
  - **Default Value**: e.g., `""`, `0`, `false`, `30d`
  - **Description**: accurate, concise explanation including accepted values
- [ ] If the flag has allowed enum values (e.g., `--group-by`, `--state`, `--format`), are all accepted values enumerated?

### C. Output Formats & Schemas
- [ ] Have output fields or table columns been added or altered?
- [ ] Are JSON / CSV field names accurately documented in `export` or `overview` sections?
- [ ] Are sample outputs or tables in `README.md` still representative of actual CLI output?

### D. Automated Audit
- [ ] Run the audit script:
  ```bash
  ./.agents/skills/readme-api-sync/scripts/audit_readme.sh
  ```
- [ ] Confirm 100% command, flag, and section check pass with exit code `0`.
