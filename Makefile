BINARY_NAME ?= gh-pr-pro

.PHONY: all help build install test test-v cover cover-text vet fmt clean check doc

all: fmt vet test build ## Format, vet, test, and build the binary

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the gh-pr-pro binary
	go build -o $(BINARY_NAME) .

install: ## Install binary to $GOPATH/bin
	go install .

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

doc: ## Start local interactive pkgsite documentation web server on localhost:8080
	@echo "Starting pkgsite documentation server at http://localhost:8080 ..."
	go run golang.org/x/pkgsite/cmd/pkgsite@latest -http=localhost:8080 .


vet: ## Run go vet static analysis
	go vet ./...

fmt: ## Format Go code with gofmt
	gofmt -s -w .

check: fmt vet test ## Run formatting, vet, and tests (pre-commit check)

clean: ## Remove build artifacts and coverage files
	rm -f $(BINARY_NAME) coverage.out

