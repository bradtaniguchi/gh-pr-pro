# Metric Pipeline Implementation Checklist & Reference

Use this checklist to track your implementation progress across every stage when adding a new metric.

---

## 1. Schema Definition (`pkg/metrics/types.go`)
- [ ] Added typed field(s) to `ProcessedPR` struct
- [ ] Applied proper JSON serialization tags (`json:"..."`)
- [ ] Verified units:
  - Latency / Duration: `*float64` or `float64` in **seconds**
  - Count: `int`
  - Status / Category: `string` or `bool`
- [ ] Handled nullable / optional states with pointer types (`*float64`, `*time.Time`)

---

## 2. GraphQL Schema & API (`pkg/api/models.go` & `pkg/api/client.go`)
- [ ] *Optional*: Added fields to `models.go` if GitHub GraphQL data was missing
- [ ] *Optional*: Updated `const prQuery` in `client.go` with explicit pagination boundaries (`first: N`)
- [ ] Verified query syntax against GitHub GraphQL Explorer or CLI

---

## 3. Calculation Logic (`pkg/metrics/calculator.go`)
- [ ] Located `ProcessPRNode(node api.GraphQLPRNode) ProcessedPR`
- [ ] Implemented calculation logic
- [ ] Checked for nil pointer dereferences:
  - [ ] `node.ClosedAt` / `node.MergedAt` nil checks
  - [ ] Timeline event empty checks
  - [ ] Review lists empty checks
- [ ] Filtered bot accounts (`strings.HasSuffix(login, "[bot]")`)
- [ ] Filtered PR author self-actions (`login == node.Author.Login`)
- [ ] Ensured ZERO `panic()` calls exist in code paths

---

## 4. Aggregation & Percentiles (`pkg/metrics/aggregation.go`)
- [ ] Added `case "<metric>":` to `computeMetricStats(...)`
- [ ] Set appropriate `unit` string (`"seconds"`, `"count"`, `"percent"`, etc.)
- [ ] Appended calculated values into `values []float64`
- [ ] *Optional*: Populated `extra["..."]` map for domain-specific extra summary metrics
- [ ] Verified group-by dimensions (`day`, `week`, `month`, `author`, `size`, `label`) partition data properly

---

## 5. CLI Subcommand Registration (`pkg/cmd/<domain>.go`)
- [ ] Declared `var <domain><Metric>Cmd = &cobra.Command{ ... }`
- [ ] Provided concise `Short` text (< 60 chars)
- [ ] Provided informative `Long` description
- [ ] Added realistic `Example` blocks with `--past 30d`, `--group-by`, etc.
- [ ] Routed `RunE` to `RunMetricCommand(cmd, "<domain>", "<metric>")`
- [ ] Registered subcommand inside `init()` via `<domain>Cmd.AddCommand(...)`
- [ ] Preserved Flag Scoping Invariant: did not register domain flags directly on subcommand

---

## 6. Testing (`pkg/metrics/*_test.go` & `pkg/cmd/root_test.go`)
- [ ] Added table-driven unit test in `pkg/metrics/calculator_test.go`
  - [ ] Tested nominal scenario
  - [ ] Tested nil/empty edge cases
  - [ ] Tested bot/self-review filtering
- [ ] Added aggregation test in `pkg/metrics/aggregation_test.go`
  - [ ] Validated correct `p50`, `mean`, and `unit`
- [ ] Passed race detector: `go test -race -v ./pkg/metrics/...`
- [ ] Passed static analysis: `go vet ./...`

---

## 7. Documentation Synchronization (`README.md`)
- [ ] Updated command hierarchy ASCII tree in `README.md`
- [ ] Added subcommand reference section under `### <domain>`
- [ ] Validated with `readme-api-sync` skill: `make audit-readme` (must exit 0)
