.PHONY: test test-cover test-cover-detail test-race test-watch test-all test-all-detail install completion lint lint-fix

# Run tests with gotestsum
test:
	gotestsum --format testdox ./...

# Run tests with coverage report (summary only)
test-cover:
	gotestsum -- -coverprofile=coverage.out ./... && \
	grep -v "testutil" coverage.out > coverage.filtered.out && \
	go tool cover -func=coverage.filtered.out | grep "total:" && \
	rm coverage.out coverage.filtered.out

# Run tests with detailed coverage report
test-cover-detail:
	gotestsum -- -coverprofile=coverage.out ./... && \
	grep -v "testutil" coverage.out > coverage.filtered.out && \
	go tool cover -func=coverage.filtered.out && \
	rm coverage.out coverage.filtered.out

# Run tests with race detector
test-race:
	gotestsum -- -race ./...

# Run tests in watch mode
test-watch:
	gotestsum --watch ./...

# Run tests with verbose output
test-v:
	gotestsum -- -v ./...

# Run tests with race detector and coverage (summary only)
test-all:
	gotestsum -- -race -coverprofile=coverage.out ./... && \
	grep -v "testutil" coverage.out > coverage.filtered.out && \
	go tool cover -func=coverage.filtered.out | grep "total:" && \
	rm coverage.out coverage.filtered.out

# Run tests with race detector and detailed coverage
test-all-detail:
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

# Install golangci-lint if not present
install-lint:
	@which golangci-lint > /dev/null || \
		(curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.55.2)

# Run linters
lint: install-lint
	golangci-lint run ./...

# Run linters and fix issues where possible
lint-fix: install-lint
	golangci-lint run --fix ./...

# Install the assetcap command
install:
	go install ./cmd/main.go
