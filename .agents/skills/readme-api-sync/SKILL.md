---
name: readme-api-sync
description: >-
  Review and update README.md whenever any API surface change has been made to the gh-pr-pro CLI, including new or modified commands, subcommands, flags, shorthands, default values, argument choices, or output formats. This skill must be called whenever any API surface area change has been made to ensure the README documentation stays in line with the code.
---

# README & API Surface Synchronization Skill

This skill governs the mandatory review and synchronization of [`README.md`](file:///Users/brad/Projects/gh-pr-pro/README.md) whenever any change is made to the `gh-pr-pro` command-line API surface area.

## Mandatory Invariant

> [!IMPORTANT]
> **This skill MUST be executed after any API surface change.**
> Any pull request, refactor, or feature branch that alters commands, subcommands, flags, shorthands, default values, filter parameters, or output schemas must run this workflow to guarantee that `README.md` remains 100% accurate and synchronized with the compiled CLI binary.

---

## Workflow Steps

### Step 1: Detect API Surface Changes

Inspect your working tree or branch to identify all files that affect the CLI API surface:

```bash
# Check modified files in pkg/cmd, pkg/output, and pkg/metrics
git status -s pkg/cmd/ pkg/output/ pkg/metrics/

# Review git diff for CLI flag and command changes
git diff HEAD pkg/cmd/ pkg/output/ pkg/metrics/types.go
```

Identify what changed:
1. **Commands & Subcommands**: Added, renamed, aliased, or removed commands in `pkg/cmd/*.go`.
2. **Flags**: Added, renamed, removed flags, shorthand aliases (`-X`), default values, or usage text in `attachTimeFlags`, `attachFilterFlags`, `attachOutputFlags`, or command-specific flag sets.
3. **Filter Dimensions / Enums**: Added or changed options for `--group-by`, `--state`, `--draft`, `--review-state`, `--checks`, `--has-conflicts`, or `--format`.
4. **Output Formats / Serialization**: Changes to table columns, JSON field names, or CSV export schemas in `pkg/output/` and `pkg/metrics/types.go`.

Consult the [API Surface Mapping Checklist](./references/api_surface_checklist.md) for the exact code-to-README section mappings.

---

### Step 2: Run the Automated Audit Helper

Run the built-in audit script to programmatically compare all registered Cobra commands and flags against [`README.md`](file:///Users/brad/Projects/gh-pr-pro/README.md):

```bash
# Run the automated README audit script
./.agents/skills/readme-api-sync/scripts/audit_readme.sh
```

Or invoke the Go audit tool directly:
```bash
go run .agents/skills/readme-api-sync/scripts/audit_readme.go
```

The script will inspect `cmd.RootCmd`, traverse all commands and flags, and output any discrepancies:
- Missing subcommands
- Missing flags (e.g. `--new-flag`)
- Missing shorthands (e.g. `-x`)
- Missing structural sections

---

### Step 3: Review and Update `README.md`

Based on the audit output and your changes, perform targeted updates to [`README.md`](file:///Users/brad/Projects/gh-pr-pro/README.md):

#### 1. Flag Scoping Tables
Locate the relevant flag table under `## Flag Scoping & Reference`:
- **Global Flags**: Registered on `RootCmd.PersistentFlags()` (`--repo`, `--no-cache`, `--verbose`, `--page-size`). Must be added to `### 1. Global Flags`.
- **Domain / Metric Flags**: Registered on `time`, `quality`, `code`, `team`, `overview`, `export`. Must be added to the appropriate sub-table (`Time Window Flags`, `PR Filtering & Search Flags`, `Aggregation & Grouping Flags`, or `Output & Formatting Flags`).
- **Cache Flags**: Specific to `cache` subcommands (`cache list`, `cache clean`, `cache path`). Must be added to `### 3. Cache Flags`.

Format table rows consistently:
```markdown
| `--my-flag` | `-m` | `default` | Accurate description including accepted choices |
```

#### 2. Command Hierarchy Diagram
If subcommands were added, renamed, or restructured, update the ASCII tree diagram under `## Command Hierarchy & Reference`.

#### 3. Subcommand Detail Sections
Navigate to the corresponding domain section:
- `### 1. gh pr-pro time (Lifecycle & Latency)`
- `### 2. gh pr-pro quality (Code Quality & Review Health)`
- `### 3. gh pr-pro code (Size, Commits & Complexity)`
- `### 4. gh pr-pro team (Throughput, Reviews & Workload)`
- `### 5. gh pr-pro overview`
- `### 6. gh pr-pro export`
- `### 7. gh pr-pro cache`

Add or update the command documentation:
- Header: `#### <subcommand>` (e.g., `#### quality ci`)
- Description of what the metric calculates and its data sources.
- Realistic copy-pasteable CLI examples including common flags (e.g., `--past 30d`, `--group-by week`).
- Sample output snippet or table if the output schema changed.

#### 4. Quick Start & Examples
If a major new capability or primary command was introduced, add a high-visibility example to `## Quick Start` near the top of the README.

---

### Step 4: Validate and Verify Synchronization

1. **Re-run the audit script**:
   ```bash
   ./.agents/skills/readme-api-sync/scripts/audit_readme.sh
   ```
   Ensure it reports:
   ```text
   Result: SUCCESS - README.md is in sync with CLI API surface area.
   ```

2. **Verify markdown syntax and table rendering**:
   Ensure markdown tables are properly aligned and all code blocks are cleanly formatted.

3. **Execute unit tests**:
   Ensure command and flag tests pass without regressions:
   ```bash
   go test -v -race ./pkg/cmd/...
   ```

4. **Run pre-commit suite**:
   ```bash
   make check
   ```
