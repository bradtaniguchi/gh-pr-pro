BINARY_NAME ?= gh-pr-pro

.PHONY: all help build install test test-v cover cover-text vet fmt fmt-check tidy-check clean check doc audit-readme audit-schema check-cross-compile

##@ Build & Execution

all: fmt-check vet test audit-readme audit-schema build ## Format, vet, test, audit, and build binary

build: ## Build the gh-pr-pro binary
	go build -o $(BINARY_NAME) .

install: ## Install binary to $GOPATH/bin
	go install .

check-cross-compile: ## Verify cross-compilation across all 5 target platforms
	./.agents/skills/release-prep/scripts/verify_cross_compile.sh

##@ Testing & Coverage

test: ## Run all unit tests with race detector
	go test -race -count=1 ./...

test-v: ## Run all unit tests with verbose output
	go test -v -race -count=1 ./...

cover: ## Run tests and open interactive HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

cover-text: ## Run tests and print coverage summary to terminal
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

##@ Quality & Pre-commit

check: fmt-check vet test audit-readme audit-schema ## Run full pre-commit check (fmt, vet, test, audits)

fmt: ## Format Go code with gofmt
	gofmt -s -w .

fmt-check: ## Verify Go code formatting without modifying files
	@UNFORMATTED=$$(gofmt -s -l .); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "The following files require formatting:"; \
		echo "$$UNFORMATTED"; \
		echo "Please run 'make fmt' locally to format them."; \
		exit 1; \
	fi

vet: ## Run go vet static analysis
	go vet ./...

tidy-check: ## Verify module dependencies (go mod tidy) are in sync
	@go mod tidy
	@if ! git diff --exit-code go.mod go.sum > /dev/null 2>&1; then \
		echo "go.mod or go.sum is out of sync. Please run 'go mod tidy' locally."; \
		exit 1; \
	fi

##@ Agent Skills & Audits

audit-readme: ## Verify README.md is in sync with CLI API surface
	go run .agents/skills/readme-api-sync/scripts/audit_readme.go

audit-schema: ## Verify output formats comply with schema contracts
	go run .agents/skills/schema-regression-test/scripts/verify_schemas.go

##@ Documentation & Utilities

doc: ## Start local interactive pkgsite documentation web server on localhost:8080
	@echo "Starting pkgsite documentation server at http://localhost:8080 ..."
	go run golang.org/x/pkgsite/cmd/pkgsite@latest -http=localhost:8080 .

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

clean: ## Remove build artifacts and coverage files
	rm -f $(BINARY_NAME) coverage.out
