# Digital Asset Capitalization Tool

[![Test Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/helmedeiros/f811420c5b31e6c4d54855df77a88527/raw/go-coverage.json)](https://github.com/helmedeiros/digital-asset-capitalization/actions)

A tool to manage digital assets and calculate time allocation for tasks across different assets.

## Overview

The Digital Asset Capitalization Tool helps organizations track and manage digital assets by providing:

- Asset lifecycle management
- Time allocation tracking for tasks
- Documentation management
- Task classification and management

The tool automatically calculates time allocation for tasks in each sprint and helps manage the capitalization of digital assets.

## Features

### Asset Management

Create and manage digital assets with comprehensive tracking:

```bash
# Create a new asset
assetcap assets create --name "Frontend App" --description "Main web application"

# List all assets
assetcap assets list

# Show detailed information about an asset
assetcap assets show --name "Frontend App"

# Update an asset's description
assetcap assets update --name "Frontend App" --description "Updated description"

# Mark asset documentation as updated
assetcap assets documentation update --asset "Frontend App"

# Manage task counts
assetcap assets tasks increment --asset "Frontend App"
assetcap assets tasks decrement --asset "Frontend App"
```

### Task Management

Comprehensive task management with JIRA integration:

```bash
# Fetch tasks from JIRA
assetcap tasks fetch --project "PROJECT" --sprint "Sprint 1"

# Classify tasks for an asset
assetcap tasks classify --project "PROJECT" --sprint "Sprint 1" --platform "jira" [--dry-run] [--apply]

# Show task details
assetcap tasks show --project "PROJECT" --sprint "Sprint 1"
```

The `classify` command supports the following options:

- `--dry-run`: Preview the classification without making any changes
- `--apply`: Write the classifications back to Jira as labels (e.g., cap-maintenance, cap-discovery, cap-development)

### Time Allocation

Automatically calculate time allocation for tasks in sprints:

```bash
# Fetch and classify tasks for a project and sprint
assetcap tasks classify --project "PROJECT" --sprint "Sprint 1"

# View the calculated time allocation
assetcap tasks show --project "PROJECT" --sprint "Sprint 1"
```

The tool:

1. Fetches tasks from JIRA for a specific project and sprint
2. Calculates time allocation based on task completion
3. Generates a formatted output for JIRA's "Time Allocation %" field
4. Supports integration with Google Spreadsheets for team-wide tracking

## Installation

### Prerequisites

- Go 1.21 or later
- Git
- Ollama (for asset enrichment)

### Installing Dependencies

The tool provides a script to install required dependencies:

```bash
# Install dependencies (Ollama, etc.)
./bin/install-deps.sh
```

This script will:

- Install Ollama and its dependencies
- Start the Ollama service
- Pull the required LLaMA model
- Work on both macOS and Linux

### Installing the Tool

```bash
# Clone the repository
git clone https://github.com/helmedeiros/digital-asset-capitalization.git
cd digital-asset-capitalization

# Install the command
make install
```

This will install the `assetcap` command in your Go bin directory (`$GOPATH/bin`). Make sure this directory is in your PATH.

Verify the installation:

```bash
assetcap --version
```

### Shell Completion

The tool supports shell completion for bash, zsh, and fish shells:

```bash
make completion
```

#### Zsh

Add to `~/.zshrc`:

```bash
fpath=(~/.zsh/completion $fpath)
autoload -U compinit && compinit
```

#### Bash

Add to `~/.bashrc`:

```bash
source /path/to/digital-asset-capitalization/completions/assetcap.bash
```

#### Fish

Copy to fish completions:

```bash
cp completions/assetcap.fish ~/.config/fish/completions/
```

## Configuration

1. Create a `teams.json` file with your team structure:

```json
{
  "PROJECT_KEY": {
    "Members": ["Team Member 1", "Team Member 2"],
    "SprintDuration": "2w",
    "WorkingHoursPerDay": 8
  }
}
```

2. Set up your Jira credentials as environment variables:

```bash
export JIRA_BASE_URL="https://your-domain.atlassian.net"
export JIRA_EMAIL="your.email@company.com"
export JIRA_TOKEN="your-api-token"
```

The tool automatically creates a `.assetcap` directory in your home folder to store:

- Asset data (`assets.json`)
- Task data (`tasks.json`)
- Generated documentation (`docs/`)

## Development

### Architecture

The project follows a hexagonal (ports and adapters) architecture pattern:

1. **Domain Layer** (`internal/*/domain/`)

   - Core business logic and entities
   - Domain models and interfaces
   - No external dependencies

2. **Application Layer** (`internal/*/application/`)

   - Use cases and business rules
   - Data flow orchestration
   - External dependency interfaces

3. **Infrastructure Layer** (`internal/*/infrastructure/`)

   - External dependency adapters
   - Persistence and services
   - JIRA integration

4. **Interface Layer** (`assetcap/action/`)
   - CLI interactions
   - Command routing
   - External API

### Testing

Run tests with various options:

```bash
# Basic test run
make test

# Test with coverage report
make test-cover

# Test with race detector
make test-race

# Test in watch mode
make test-watch

# Test with verbose output
make test-v

# Run all tests (with race detector and coverage)
make test-all

# Run benchmarks
make bench
```

Coverage requirements:

- Domain layer: >90%
- Application layer: >80%
- Infrastructure layer: >80%
- Overall coverage: >80%

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
