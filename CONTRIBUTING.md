# Contributing to `gh-pr-pro`

Thank you for your interest in contributing to `gh-pr-pro`! This document covers setup, development workflows, testing strategies, project architecture, and instructions for adding new metric domains.

This document largely assumes you are leveraging AI coding tools to help contribute, for which this repository is designed to support to the best of its ability. That said, this document targets both human and AI powered development.

---

## Prerequisites & Tooling

To develop and test `gh-pr-pro`, ensure you have the following installed:

- **Go**: `1.22+` (Go `1.24+` recommended). Verify with `go version`.
- **GitHub CLI (`gh`)**: `v2.20.0+`. Verify with `gh version`.
- **golangci-lint**: `v2.0+` recommended for static analysis (`brew install golangci-lint` or see [golangci-lint installation](https://golangci-lint.run/welcome/install/)).
- **GitHub Account Authentication & Scopes**:
  ```bash
  # Check current auth status and granted token scopes
  gh auth status

  # Alternatively, inspect raw HTTP OAuth scope headers
  gh api / -i | grep -i x-oauth-scopes
  ```
  Ensure your `gh` token has `repo` and `read:org` scopes. If missing, request additional scopes using:
  ```bash
  gh auth refresh -s repo -s read:org
  ```

---

## Local Development Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/bradtaniguchi/gh-pr-pro.git
   cd gh-pr-pro
   ```

2. **Download Go module dependencies**:
   ```bash
   go mod download
   go mod tidy
   ```

3. **Install the local repository as a `gh` extension**:
   ```bash
   # Install directly from the local directory (creates a symlink)
   gh extension install .
   ```

   Verify the extension is installed and detected:
   ```bash
   gh extension list
   gh pr-pro --help
   ```

4. **Initialize Git pre-commit hooks** (recommended):
   ```bash
   make init-hooks
   ```
   This configures Git's native `core.hooksPath` to use `.githooks/pre-commit`, running `make check` (module tidy check, static analysis, pinned vulnerability scan, race-detected tests, README audit, and schema validation) before each commit.

---

## Development Workflow

A `Makefile` is included to streamline common build, test, and formatting tasks:

```bash
# Display all available Make targets
make help

# Configure project git pre-commit hooks (.githooks)
make init-hooks

# Tidy dependencies, lint, test, audit README & schemas, and build the binary
make all

# Run comprehensive static analysis with golangci-lint
make lint

# Run the pinned govulncheck vulnerability scanner
make vulncheck

# Format Go code with golangci-lint / gofumpt
make fmt

# Fast build of the extension binary
make build

# Install binary to $GOPATH/bin
make install

# Pre-commit verification (tidy + lint + vulncheck + test + audit-readme + audit-schema)
make check

# Audit README synchronization against CLI API surface
make audit-readme

# Audit output serialization format schemas
make audit-schema

# Verify cross-compilation across all 5 target platforms (Linux, macOS, Windows)
make check-cross-compile
```

You can test commands locally against any public repository without changing directory:

```bash
# 1. Run the extension directly via GitHub CLI
gh pr-pro time merge --past 7d -R cli/cli

# 2. Or invoke the compiled binary directly during prototyping
./gh-pr-pro quality ci --past 14d -R cli/cli --json
```

> [!TIP]
> Use `-R <owner>/<repo>` (e.g., `-R cli/cli` or `-R golang/go`) when testing commands without needing to be inside a local git clone of that repository.

---

## Testing Strategy

### Running Unit Tests

We use Go's standard table-driven testing pattern across all metric calculators, aggregation engines, filter matrices, and output formatters:

```bash
# Run all unit tests with race detection
make test

# Run tests with verbose table-driven subtest output
make test-v

# Generate and open an interactive HTML coverage report
make cover

# Print terminal coverage summary
make cover-text
```

You can also run native Go test commands directly:

```bash
go test -v -race -count=1 ./...
```

### Live Testing with GitHub CLI

Test various metric domains and flag combinations against real public repositories:

```bash
# 1. Lifecycle & Latency commands
gh pr-pro time merge --past 30d --group-by month -R cli/cli
gh pr-pro time review --past 14d --group-by reviewer -R cli/cli
gh pr-pro time draft --past 60d --group-by size -R cli/cli

# 2. Quality & CI health commands
gh pr-pro quality ci --past 30d --group-by week -R cli/cli
gh pr-pro quality rework --past 60d --author "@octocat" -R cli/cli

# 3. Code complexity & sizing
gh pr-pro code size --past 30d --format markdown -R cli/cli
gh pr-pro code commits --past 90d --group-by month -R cli/cli

# 4. Multi-format exports
gh pr-pro export --past 7d --csv -R cli/cli
gh pr-pro overview --past 30d --json -R cli/cli
```

### Cache Inspection & Management

`gh-pr-pro` stores cached PRs under `~/.cache/gh-pr-pro/`.

- To list cache entries: `gh pr-pro cache list`
- To inspect cache directory: `gh pr-pro cache path`
- To test without cache: pass `--no-cache`.
- To clear cache via CLI:
  ```bash
  gh pr-pro cache clean cli/cli   # clear single repo
  gh pr-pro cache clean --all     # clear everything
  ```

---

## Project Architecture & Directory Layout

```text
gh-pr-pro/
├── .agents/
│   └── skills/
│       ├── readme-api-sync/        # AI skill to review & sync README on API surface changes
│       ├── add-metric/             # AI skill to scaffold & integrate new metric subcommands
│       ├── schema-regression-test/ # AI skill to audit & verify output schemas (JSON/CSV/TSV/MD)
│       └── release-prep/           # AI skill for cross-compile verification, notes & tagging
├── .githooks/
│   └── pre-commit                 # Zero-dependency pre-commit verification hook
├── .github/
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md          # Bug report issue template
│   │   └── feature_request.md     # Feature request issue template
│   ├── dependabot.yml             # Automated dependency and GitHub Actions update config
│   ├── pull_request_template.md   # PR verification checklist template
│   └── workflows/
│       ├── ci.yml                 # Continuous integration (lint, test matrix, build verification)
│       ├── release.yml            # Automated release packaging via cli/gh-extension-precompile
│       └── security.yml           # Scheduled weekly govulncheck dependency vulnerability scan
├── CODE_OF_CONDUCT.md             # Contributor Covenant Code of Conduct
├── SECURITY.md                    # Security vulnerability disclosure policy
├── main.go                       # Application entry point
├── pkg/
│   ├── api/                      # GitHub GraphQL client & pagination
│   │   ├── client.go             # GraphQL client using go-gh API
│   │   └── models.go             # Raw GraphQL schema structs & queries
│   ├── cache/                    # Local persistent cache manager
│   │   └── cache.go              # File-based JSON caching (~/.cache/gh-pr-pro/)
│   ├── cmd/                      # Cobra CLI command definitions
│   │   ├── root.go               # Global flags & repository resolution
│   │   ├── time.go               # time domain (merge, review, draft, pickup, idle)
│   │   ├── quality.go            # quality domain (ci, rework, conflicts, reverts)
│   │   ├── code.go               # code domain (size, commits)
│   │   ├── team.go               # team domain (throughput, reviews, unreviewed)
│   │   ├── overview.go           # overview composite scorecard
│   │   ├── export.go             # export raw dataset streamer
│   │   └── cache.go              # cache management commands
│   ├── metrics/                  # Metric processing & math engine
│   │   ├── types.go              # ProcessedPR and metric schemas
│   │   ├── calculator.go         # Raw GraphQL node -> ProcessedPR transformer
│   │   ├── aggregation.go        # Grouping & dimension bucketing
│   │   └── stats.go              # Percentiles (p50..p99), means, and formatting
│   └── output/                   # Multi-format rendering engine
│       └── formatter.go          # Text tables, JSON, CSV, TSV, and Markdown
├── README.md                     # User documentation & CLI reference
├── CONTRIBUTING.md               # Development & testing guide
└── AGENTS.md                     # AI agent architecture and workflow instructions
```

---

Follow the [`add-metric`](file:///Users/brad/Projects/gh-pr-pro/.agents/skills/add-metric/SKILL.md) skill workflow when contributing a new metric:

1. **Scaffold Boilerplate**:
   ```bash
   go run .agents/skills/add-metric/scripts/scaffold_metric.go --domain <domain> --metric <subcommand>
   ```

2. **Update Domain Structs (`pkg/metrics/types.go`)**:
   Add any new calculated fields to `ProcessedPR`.

3. **Add Calculation Logic (`pkg/metrics/calculator.go`)**:
   Extract and compute the new metric from raw GraphQL timeline items, reviews, or commits.

4. **Update Aggregator (`pkg/metrics/aggregation.go`)**:
   Add the metric to `computeMetricStats(...)` so grouping and percentiles are automatically generated.

5. **Register Cobra Command (`pkg/cmd/<domain>.go`)**:
   Add the subcommand using `RunMetricCommand("<domain>", "<subcommand>")`.

6. **Verify Wiring**:
   ```bash
   go run .agents/skills/add-metric/scripts/scaffold_metric.go --check <subcommand>
   ```

7. **Write Unit Tests (`pkg/metrics/*_test.go`)**:
   Add test coverage for calculation and grouping.

8. **Update Documentation & Verify with `readme-api-sync` Skill**:
   Run the `readme-api-sync` skill (`.agents/skills/readme-api-sync/SKILL.md`) or `make audit-readme` to ensure `README.md` command trees and flag tables are updated.

---

## Code Style & Quality Guidelines

- **Formatting**: Format all Go code using `gofmt` or `goimports`:
  ```bash
  gofmt -s -w .
  ```
- **Static Analysis**: Run `go vet` before opening PRs:
  ```bash
  go vet ./...
  ```
- **Error Handling**: Always wrap errors with context using `fmt.Errorf("...: %w", err)`.
- **Zero Panic Policy**: Never use `panic()` in production code paths. Propagate errors up to Cobra's `RunE`.

---

## Code of Conduct & Security Policy

- **Code of Conduct**: We are committed to providing a welcoming, inclusive, and harassment-free environment. All contributors and participants are expected to uphold our [Code of Conduct](CODE_OF_CONDUCT.md) (adapted from Contributor Covenant v2.1).
- **Reporting Vulnerabilities**: Please do not report security issues via public GitHub issues. Follow the vulnerability disclosure instructions in [SECURITY.md](SECURITY.md).

---

## Submitting Pull Requests

1. Initialize git pre-commit hooks (if not done yet): `make init-hooks`.
2. Create a feature branch: `git checkout -b feat/my-new-metric`.
3. Ensure pre-commit checks pass: `make check`.
4. Push to your fork and submit a Pull Request against `main` using the provided [pull request template](.github/pull_request_template.md).

---

## Release & Publishing Process

Maintainers follow this workflow to publish new versions of `gh-pr-pro` as a GitHub CLI extension:

### 1. Ensure Repository Discoverability
Ensure the repository has the `gh-extension` topic so it is discoverable via `gh extension search` and `gh extension browse`:
```bash
gh repo edit bradtaniguchi/gh-pr-pro --add-topic gh-extension
```

### 2. Pre-Release Verification Gate
Before releasing, ensure the local working tree is clean and up to date:
```bash
# Ensure working tree is clean and on main
git checkout main && git pull origin main
```

Optionally run the full verification gate locally prior to triggering the release:
```bash
# Run full CI quality checks (tidy check, linting, vulnerability scan, tests with race detector)
make check

# Verify output schemas (JSON/CSV/TSV/Markdown)
make audit-schema

# Verify cross-compilation across all 5 OS/architecture targets (Darwin, Linux, Windows)
make check-cross-compile
```

### 3. Determine Semantic Version
Releases adhere to [Semantic Versioning](https://semver.org/) (`vMAJOR.MINOR.PATCH`):
- **PATCH** (`v0.1.1`): Bug fixes, internal refactors, documentation updates.
- **MINOR** (`v0.2.0`): New metric commands, new flags, new export formats.
- **MAJOR** (`v1.0.0`): Breaking CLI changes or schema alterations.

Generate release notes from commits since the last tag:
```bash
./.agents/skills/release-prep/scripts/generate_release_notes.sh
```

---

### 4. Publish Release

#### Method A: Automated Release Dispatch (Recommended for Maintainers)

Maintainers can trigger the automated release workflow directly from the command line or GitHub web UI. The workflow runs on [`.github/workflows/release.yml`](.github/workflows/release.yml):

1. **Maintainer Authorization**: The workflow queries the GitHub Collaborator API to enforce that the executing user has maintainer (`admin` or `write`) permissions on the repository. Unauthorized invocations fail immediately.
2. **Automated Quality & Compilation Gates in CI**: Automatically runs `make tidy-check`, unit tests with race detection, schema contract validation (`make audit-schema`), README synchronization check (`make audit-readme`), and cross-platform compilation across all 5 OS/architecture targets.
3. **Build & Publish Assets**: Invokes `cli/gh-extension-precompile@v2` to cross-compile binary releases, tag the release on GitHub, generate release notes, and upload multi-platform archive assets.

**Triggering via GitHub CLI (`gh`)**:
```bash
# Dispatch automated release for v0.2.0
gh workflow run release.yml -f tag=v0.2.0

# Optional: Run in dry-run mode (runs all checks and compilation without publishing)
gh workflow run release.yml -f tag=v0.2.0 -f dry_run=true
```

**Triggering via GitHub Web UI**:
1. Navigate to the repository's **Actions** tab.
2. Select the **Release** workflow from the left sidebar.
3. Click **Run workflow**, enter the release tag (e.g. `v0.2.0`), and submit.

#### Method B: Manual Git Tag & Push

Alternatively, maintainers can create and push an annotated git tag manually:
```bash
# Create annotated tag
git tag -a v0.2.0 -m "Release v0.2.0"

# Push tag to trigger release workflow
git push origin v0.2.0
```

---

### 5. Verify Installation
Once the GitHub Actions workflow completes, test installing and running the published release:
```bash
# Install or upgrade the extension from remote
gh extension install bradtaniguchi/gh-pr-pro

# Verify binary execution
gh pr-pro --help

# Verify extension is listed with its remote repository and published version tag
gh extension list | grep gh-pr-pro
```
