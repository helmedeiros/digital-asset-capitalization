.PHONY: test test-cover test-race test-watch install

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

# Install the assetcap command
install:
	go install ./cmd/main.go
