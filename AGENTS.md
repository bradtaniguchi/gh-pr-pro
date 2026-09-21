# AGENTS.md — Agent Guidance for `gh-pr-pro`

This document defines critical architectural invariants, error policies, and agent workflows for `gh-pr-pro`. For package architecture and contribution setup, see [`CONTRIBUTING.md`](CONTRIBUTING.md).

---

## 1. Quality Gate & Verification

Always verify changes using the canonical pre-commit quality gate:

```bash
make check
```
*(Runs module tidy check, `golangci-lint`, `govulncheck`, race-detected unit tests, README audit, and schema validation).*

---

## 2. Key Invariants & Architectural Rules

### Flag Scoping Rules
- **Global Flags** (`--repo`/`-R`, `--no-cache`, `--verbose`/`-v`, `--page-size`) are registered as persistent flags on `RootCmd`.
- **Metric/Filter/Output Flags** (`--past`, `--since`, `--group-by`, `--json`, etc.) must **only** attach to metric domains (`time`, `quality`, `code`, `team`), `overview`, and `export`.
- **Cache Commands** (`cache list`, `cache clean`, `cache path`) must **never** accept metric or filtering flags (enforced in [`pkg/cmd/cache_test.go`](pkg/cmd/cache_test.go)).

### GraphQL & Network Resilience
- **Delta Fetching:** Always calculate updates using delta timestamps (`last_fetched` in local cache).
- **Graceful Offline Fallback:** Never fail hard on network errors or rate limits if cached data exists; fall back to cached records in [`FetchAndProcessPRs`](pkg/cmd/root.go).
- **Backend Execution Ceiling:** Keep query page size conservative (max 100) to stay well under GitHub's 10-second GraphQL execution ceiling and prevent `HTTP 504 Gateway Timeout` errors on large repositories.
- **Adaptive Halving:** Always preserve the adaptive page-size halving fallback in [`pkg/api/client.go`](pkg/api/client.go) on timeout errors.

### Calculations & Zero-Panic Policy
- All duration calculations in [`pkg/metrics/calculator.go`](pkg/metrics/calculator.go) are represented as `float64` seconds.
- Handle `nil` timestamps gracefully (e.g., unmerged PRs or drafts without `ready_for_review_at`).
- **Never call `panic()`** in production code. Return wrapped errors (`fmt.Errorf("...: %w", err)`).

---

## 3. Specialized Agent Skills

Use the dedicated agent skills in [`.agents/skills/`](.agents/skills/) for complex workflows:
- [`add-metric`](.agents/skills/add-metric/SKILL.md): Scaffold, compute, aggregate, and register new metric subcommands.
- [`readme-api-sync`](.agents/skills/readme-api-sync/SKILL.md): Audit and synchronize `README.md` on any CLI API surface change.
- [`schema-regression-test`](.agents/skills/schema-regression-test/SKILL.md): Audit output serialization formats (JSON, CSV, TSV, Markdown).
- [`release-prep`](.agents/skills/release-prep/SKILL.md): Verify cross-compilation and prepare release tags.
