.PHONY: test test-cover test-cover-detail test-race test-watch test-all test-all-detail install install-tools install-lint install-gotestsum completion lint lint-fix check test-cover-gate pre-push build build-run

# Build the binary
build:
	@echo "Building assetcap binary..."
	@go build -o assetcap cmd/main.go cmd/console.go cmd/deployment_commands.go
	@echo "✅ Built assetcap binary"

# Build and run assetcap with arguments
build-run: build
	@./assetcap $(ARGS)

# Run all checks (lint + test)
check: lint test

# Run coverage check with 70% threshold enforcement
test-cover-gate:
	@./scripts/coverage-gate.sh

# Run pre-push checks (includes coverage gate)
pre-push: lint test-cover-gate

# Run tests with gotestsum
test: install-gotestsum
	gotestsum --format testdox ./...

# Run tests with coverage report (summary only)
test-cover: install-gotestsum
	gotestsum -- -coverprofile=coverage.out ./... && \
	grep -v "testutil" coverage.out > coverage.filtered.out && \
	go tool cover -func=coverage.filtered.out | grep "total:" && \
	rm coverage.out coverage.filtered.out

# Run tests with detailed coverage report
test-cover-detail: install-gotestsum
	gotestsum -- -coverprofile=coverage.out ./... && \
	grep -v "testutil" coverage.out > coverage.filtered.out && \
	go tool cover -func=coverage.filtered.out && \
	rm coverage.out coverage.filtered.out

# Run tests with race detector
test-race: install-gotestsum
	gotestsum -- -race ./...

# Run tests in watch mode
test-watch: install-gotestsum
	gotestsum --watch ./...

# Run tests with verbose output
test-v: install-gotestsum
	gotestsum -- -v ./...

# Run tests with race detector and coverage (summary only)
test-all: install-gotestsum
	gotestsum -- -race -coverprofile=coverage.out ./... && \
	grep -v "testutil" coverage.out > coverage.filtered.out && \
	go tool cover -func=coverage.filtered.out | grep "total:" && \
	rm coverage.out coverage.filtered.out

# Run tests with race detector and detailed coverage
test-all-detail: install-gotestsum
	gotestsum -- -race -coverprofile=coverage.out ./... && \
	grep -v "testutil" coverage.out > coverage.filtered.out && \
	go tool cover -func=coverage.filtered.out && \
	rm coverage.out coverage.filtered.out

# Generate and install shell completion scripts
completion:
	@echo "Installing shell completions..."
	@mkdir -p completions
	@mkdir -p ~/.zsh/completion
	@assetcap completion zsh > completions/_assetcap
	@assetcap completion bash > completions/assetcap.bash
	@assetcap completion fish > completions/assetcap.fish
	@cp completions/_assetcap ~/.zsh/completion/
	@echo "Zsh completion installed to ~/.zsh/completion/_assetcap"
	@echo "Add the following to your ~/.zshrc if not already present:"
	@echo "  fpath=(~/.zsh/completion \$$fpath)"
	@echo "  autoload -U compinit && compinit"
	@echo ""
	@echo "Bash completion saved to completions/assetcap.bash"
	@echo "To use it, add this to your ~/.bashrc:"
	@echo "  source $(PWD)/completions/assetcap.bash"
	@echo ""
	@echo "Fish completion saved to completions/assetcap.fish"
	@echo "To use it, copy to the fish completions directory:"
	@echo "  cp completions/assetcap.fish ~/.config/fish/completions/"

# Install every dev tool the Makefile depends on. Use this on fresh
# clones so `make pre-push` works end-to-end.
install-tools: install-lint install-gotestsum

# Install gotestsum if not present. Used by `make test*` targets.
install-gotestsum:
	@command -v gotestsum >/dev/null 2>&1 || \
		go install gotest.tools/gotestsum@latest

# Install golangci-lint if not present, or if the installed version is
# below the v2.x required by .golangci.yml. CI pins the same version
# in .github/workflows/ci.yml — keep these two in sync.
GOLANGCI_LINT_VERSION := v2.10.1
install-lint:
	@if ! command -v golangci-lint >/dev/null 2>&1 || \
		! golangci-lint version 2>&1 | grep -q "version v2"; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
			sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	fi

# Run linters
lint: install-lint
	golangci-lint run ./...

# Run linters and fix issues where possible
lint-fix: install-lint
	golangci-lint run --fix ./...

# Install the assetcap command
install:
	go install ./cmd/main.go
