# Schema Contracts & Serialization Specifications

This document defines the exact serialization specifications for all output formats supported by `gh-pr-pro`.

---

## 1. Aggregate Metric CSV Schema

When running any metric command with `--csv` (e.g. `gh pr-pro time merge --csv`):

- **Delimiter**: `,` (ASCII 44)
- **Header Row**:
  ```text
  group_key,count,p50,p75,p90,p95,p99,mean,min,max,unit
  ```
- **Column Specifications**:
  1. `group_key`: string (e.g., `"Total"`, `"2025-01"`, `"@octocat"`, `"L"`)
  2. `count`: integer string (e.g., `"42"`)
  3. `p50`: float formatted to 2 decimals (`%.2f`)
  4. `p75`: float formatted to 2 decimals (`%.2f`)
  5. `p90`: float formatted to 2 decimals (`%.2f`)
  6. `p95`: float formatted to 2 decimals (`%.2f`)
  7. `p99`: float formatted to 2 decimals (`%.2f`)
  8. `mean`: float formatted to 2 decimals (`%.2f`)
  9. `min`: float formatted to 2 decimals (`%.2f`)
  10. `max`: float formatted to 2 decimals (`%.2f`)
  11. `unit`: string (e.g., `"seconds"`, `"count"`, `"percent"`, `"lines"`)

*Note: For duration metrics where `unit == "seconds"`, `p50` through `mean` are converted to decimal hours before formatting.*

---

## 2. Aggregate Metric TSV Schema

When running any metric command with `--tsv` (e.g. `gh pr-pro quality rework --tsv`):

- **Delimiter**: `\t` (Tab, ASCII 9)
- **Header Row**:
  ```text
  group_key	count	p50	p75	p90	p95	p99	mean	min	max	unit
  ```
- Columns follow the exact specifications as CSV above.

---

## 3. Raw Export CSV Schema

When running `gh pr-pro export --csv`:

- **Delimiter**: `,` (ASCII 44)
- **Header Row** (23 columns):
  ```text
  number,title,author,state,is_draft,created_at,ready_for_review_at,first_reviewed_at,merged_at,closed_at,ttm_hours,ttfr_hours,draft_hours,additions,deletions,changed_files,commit_count,size_category,had_ci_failure,ci_failed_runs,reviews_count,review_roundtrips,labels
  ```
- **Labels**: multiple labels are joined with `;` (semicolon) to prevent CSV delimiter conflicts.
- **Timestamps**: RFC3339 format (`2006-01-02T15:04:05Z`) or empty string `""` if event has not occurred.
- **Durations**: decimal hours formatted with 2 decimal places (`%.2f`) or empty string `""` if event not reached.

---

## 4. Metric JSON Contract

When running any metric command with `--json`:

```json
{
  "domain": "time",
  "metric": "merge",
  "time_window": {
    "since": "2025-01-01T00:00:00Z",
    "until": "2025-02-01T00:00:00Z",
    "past": "30d"
  },
  "summary": {
    "group_key": "Total",
    "count": 15,
    "p50": 14200.5,
    "p75": 28400.0,
    "p90": 45000.0,
    "p95": 60000.0,
    "p99": 75000.0,
    "mean": 21000.2,
    "min": 120.0,
    "max": 80000.0,
    "unit": "seconds",
    "extra_metrics": {}
  },
  "groups": [],
  "prs": []
}
```

- Field names MUST remain `snake_case` in JSON serialization.
- `extra_metrics` is optional; if present, keys depend on domain/metric (e.g. `top_failing` for CI, `approval_rate` for reviews).
- `prs` is omitted unless `--detailed` flag is enabled.

---

## 5. Markdown Table Contract

When running with `--markdown` or `--md`:
- Must begin with header row: `| Group | Count | p50 | p75 | p90 | Mean | Min | Max | Unit |`
- Must include GFM separator line: `|---|---|---|---|---|---|---|---|---|`
- Values formatted with clean spacing.
