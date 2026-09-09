---
name: schema-regression-test
description: Audit, test, and verify output format schemas (JSON, CSV, TSV, text, Markdown) in gh-pr-pro to prevent breaking changes in exported fields, column headers, units, or serialization contracts for downstream consumers.
---

# Output Schema Regression Testing Skill

This skill governs the verification and maintenance of output serialization schemas across all formats supported by `gh-pr-pro` (`text`, `json`, `csv`, `tsv`, `markdown`).

## When to Use This Skill

Execute this skill whenever you:
- Modify or add fields to [`pkg/metrics/types.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/metrics/types.go) (`MetricOutput`, `GroupSummary`, `ProcessedPR`, or `OverviewReport`)
- Touch rendering logic in [`pkg/output/formatter.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/output/formatter.go)
- Alter column headers, formatting precision, units, or JSON key tags
- Before submitting PRs that affect CLI output data pipelines or external consumer integrations

---

## The Contract Invariants

`gh-pr-pro` outputs are widely consumed by automated CI scripts, jq filters, python pandas jobs, and data warehouses. The following invariants must remain backward compatible:

### 1. Metric Aggregate CSV & TSV Header Invariant
Metric table exports (`--csv` and `--tsv`) MUST always output exactly 11 standard columns in this exact sequence:
```text
group_key,count,p50,p75,p90,p95,p99,mean,min,max,unit
```
*(Delimiter is `,` for CSV and `\t` for TSV).*

### 2. Raw Dataset Export CSV Header Invariant
The raw export streamer (`gh pr-pro export --csv`) MUST output exactly 23 standard columns in this exact sequence:
```text
number,title,author,state,is_draft,created_at,ready_for_review_at,first_reviewed_at,merged_at,closed_at,ttm_hours,ttfr_hours,draft_hours,additions,deletions,changed_files,commit_count,size_category,had_ci_failure,ci_failed_runs,reviews_count,review_roundtrips,labels
```

### 3. Duration Unit Conversion Invariant
- In **JSON** output: raw float values are preserved in their base unit (e.g. seconds for lifecycle metrics, line count for size).
- In **Text, Markdown, CSV, and TSV** outputs: if `unit == "seconds"`, duration values (`p50`..`p99`, `mean`) are automatically formatted as decimal hours formatted to 2 decimal places (`%.2f`).

### 4. JSON Serialization Invariant
The root structure of `MetricOutput` in JSON format MUST contain:
- `domain` (string)
- `metric` (string)
- `time_window` (object with `since`, `until`, `past`)
- `summary` (object with `group_key`, `count`, `p50`, `p75`, `p90`, `p95`, `p99`, `mean`, `min`, `max`, `unit`)
- `groups` (array of `GroupSummary` objects, present or empty)
- `prs` (array of `ProcessedPR` objects, omitted unless `--detailed` is passed)

---

## Workflow Steps

### Step 1: Run the Schema Regression Suite

Run the automated schema verification tool:

```bash
# Run schema audit
go run .agents/skills/schema-regression-test/scripts/verify_schemas.go

# Or via shell wrapper
./.agents/skills/schema-regression-test/scripts/verify_schemas.sh
```

The tool will exercise `RenderOutput` and `RenderRawExport` across all 5 formats with synthetic PR datasets and verify that:
- [x] CSV headers and delimiters match the contract
- [x] TSV headers and delimiters match the contract
- [x] JSON encodes properly and unmarshals into the expected contract types
- [x] Markdown tables contain valid GFM header rows and pipes
- [x] Text tables contain ANSI box-drawing borders and headers
- [x] Raw export CSV headers contain all 23 required fields

---

### Step 2: Review Discrepancies & Diffs

If the verification script reports a discrepancy:
1. Review [`references/schema_contracts.md`](file:///Users/brad/Projects/gh-pr-pro/.agents/skills/schema-regression-test/references/schema_contracts.md) to understand the contract requirements.
2. Determine if the schema change was intentional:
   - **If intentional** (e.g., adding an optional column or extending `GroupSummary`): update the contract definitions and the verification suite in `scripts/verify_schemas.go`.
   - **If unintended** (e.g., accidental rename of JSON tag, changing decimal formatting, or dropping CSV columns): restore backward compatibility in [`pkg/output/formatter.go`](file:///Users/brad/Projects/gh-pr-pro/pkg/output/formatter.go).

---

### Step 3: Run Package Tests

Ensure unit tests in `pkg/output` pass:

```bash
go test -v -race ./pkg/output/...
```
