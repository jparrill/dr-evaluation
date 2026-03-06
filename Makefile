BINARY   := bin/dr-evaluation
GOFLAGS  := -trimpath
LDFLAGS  := -s -w

.PHONY: all build test test-verbose test-cover clean fmt vet lint help

all: fmt vet test build ## Run fmt, vet, test, and build

build: ## Build the binary into bin/
	@mkdir -p bin
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINARY) .

test: ## Run unit tests
	go test ./... -count=1

test-verbose: ## Run unit tests with verbose output
	go test ./... -v -count=1

test-cover: ## Run tests with coverage report
	go test ./... -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML report: go tool cover -html=coverage.out"

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out

fmt: ## Format source code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (requires golangci-lint installed)
	golangci-lint run ./...

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
