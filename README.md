# Digital Asset Capitalization Tool

A tool to manage digital assets and calculate time allocation for tasks across different assets.

## What is this?

This tool helps track and manage digital assets in your organization, including:

1. Asset lifecycle management
2. Time allocation tracking for tasks
3. Documentation management
4. Contribution type tracking

The tool automatically calculates the time allocation for tasks in each sprint and helps manage the capitalization of digital assets.

## Features

### Asset Management

The tool provides comprehensive asset management capabilities:

- **Asset Creation and Tracking**

  ```go
  asset, err := NewAsset("Frontend App", "Main web application")
  ```

- **Contribution Types**
  Assets support three types of contributions:

  - `discovery`: Initial research and requirements gathering
  - `development`: Implementation of new features
  - `maintenance`: Bug fixes and improvements

  ```go
  asset.AddContributionType("development")
  ```

- **Task Association**
  Track tasks associated with each asset:

  ```go
  asset.IncrementTaskCount() // When adding a task
  asset.DecrementTaskCount() // When removing a task
  ```

- **Documentation Updates**
  Track documentation changes:
  ```go
  asset.UpdateDocumentation()
  ```

### Time Allocation

This tool automatically calculates the time it took the developer
to complete the task in the sprint in % of the total sprint time, as later fill it into "Time Allocation %" in JIRA.

Simply copy-paste the output to the Google Spreadsheet with the split per columns.

## Setup

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Create a `teams.json` file with your team structure:
   ```json
   {
     "PROJECT_KEY": {
       "Members": ["Team Member 1", "Team Member 2"]
     }
   }
   ```
4. Set up your Jira credentials as environment variables:
   ```bash
   export JIRA_BASE_URL="https://your-domain.atlassian.net"
   export JIRA_EMAIL="your.email@company.com"
   export JIRA_TOKEN="your-api-token"
   ```

## Testing

This project uses `gotestsum` for better test output and organization. To run tests, you have several options:

### Basic Test Run

```bash
make test
```

### Test with Coverage Report

```bash
make test-cover
```

### Test with Race Detector

```bash
make test-race
```

### Test in Watch Mode (useful during development)

```bash
make test-watch
```

### Test with Verbose Output

```bash
make test-v
```

### Run All Tests (with race detector and coverage)

```bash
make test-all
```

### Performance Benchmarks

Run benchmarks to measure performance:

```bash
go test -bench=. -benchmem ./internal/assets/model/...
```

Current benchmark results on M1 Pro:

- Asset creation: ~623 ns/op
- Description updates: ~59 ns/op
- Contribution type additions: ~25 ns/op
- Task count operations: ~134 ns/op
- ID generation: ~505 ns/op
- Concurrent operations: ~366 ns/op

### Test Coverage Report

The current test coverage is:

- `assetcap/action`: 88.2%
- `assetcap/config`: 96.0%
- `assetcap`: 18.4%
- `internal/assets/model`: 100%

## Usage

### Asset Management

1. Create a new asset:

```bash
assetcap assets create --name "Frontend" --description "Main web application"
```

2. Add contribution types:

```bash
assetcap assets contribution-type add --asset "Frontend" --type "development"
```

3. Generate documentation:

```bash
assetcap docs generate --asset "Frontend" --platform confluence
```

### Time Allocation

Run the time allocation calculation:

```bash
assetcap tasks allocate --project PROJECT_KEY --sprint "Sprint Name" [-override '{"ISSUE-KEY": hours}']
```

The tool will:

1. Calculate time allocation for each task
2. Update task metadata with allocation percentages
3. Generate a report showing allocation per story and engineer

## Security Note

For better security, you can add these environment variables to your shell's configuration file (e.g., `~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish`). This way, they'll be available in your shell sessions without being stored in any files in the project directory.

## Thread Safety

All operations in the Asset model are thread-safe and can be used concurrently. The implementation uses proper synchronization to ensure data consistency in multi-threaded environments.
