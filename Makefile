.PHONY: test test-cover test-race test-watch install completion

# Run tests with gotestsum
test:
	gotestsum ./...

# Run tests with coverage report
test-cover:
	gotestsum -- -cover ./...

# Run tests with race detector
test-race:
	gotestsum -- -race ./...

# Run tests in watch mode
test-watch:
	gotestsum --watch ./...

# Run tests with verbose output
test-v:
	gotestsum -- -v ./...

# Run tests with race detector and coverage
test-all:
	gotestsum -- -race -cover ./...

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

# Install the assetcap command
install:
	go install ./cmd/main.go
