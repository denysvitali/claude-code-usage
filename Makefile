.PHONY: all build test lint clean install release snapshot fmt tidy coverage help

# Binary name
BINARY := claude-usage

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# Version (can be overridden: make build VERSION=1.2.3)
VERSION ?= dev

# Build flags
LDFLAGS := -s -w -X github.com/denysvitali/claude-code-usage/internal/version.Version=$(VERSION)

# Default target
all: lint test build

## build: Build the binary
build:
	$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/claude-usage

## test: Run tests with race detection
test:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## lint: Run golangci-lint
lint:
	golangci-lint run

## fmt: Format code
fmt:
	$(GOFMT) -s -w .
	goimports -w -local github.com/denysvitali/claude-code-usage .

## tidy: Tidy and verify dependencies
tidy:
	$(GOMOD) tidy
	$(GOMOD) verify

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out
	rm -rf dist/

## install: Install binary to GOPATH/bin
install:
	$(GOCMD) install ./cmd/claude-usage

## snapshot: Create a snapshot release (for testing)
snapshot:
	goreleaser release --snapshot --clean

## release: Create a release (requires GITHUB_TOKEN)
release:
	goreleaser release --clean

## coverage: Generate and open coverage report
coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## help: Show this help
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/ /'
