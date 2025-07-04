# Digital Asset Capitalization Tool

[![Test Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/helmedeiros/f811420c5b31e6c4d54855df77a88527/raw/go-coverage.json)](https://github.com/helmedeiros/digital-asset-capitalization/actions)

A tool to manage digital assets and calculate time allocation for tasks across different assets.

## Overview

The Digital Asset Capitalization Tool helps organizations track and manage digital assets by providing:

- Asset lifecycle management
- Time allocation tracking for tasks
- Documentation management
- Task classification and management
- AI-powered asset enrichment and keyword generation

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

# Generate keywords for an asset using LLaMA 3
assetcap assets keywords --name "Frontend App"
```

### Asset Keywords

The tool can automatically generate relevant keywords for your assets using LLaMA 3:

```bash
# Generate keywords for an asset
assetcap assets keywords --name "Frontend App"
```

The keyword generation:

- Uses LLaMA 3 to analyze the asset's content
- Generates 5-10 relevant technical keywords
- Considers the asset's name, description, purpose, benefits, and implementation details
- Helps with asset discoverability and task classification
- Keywords are used for matching tasks to assets

The generated keywords are:

- Technical and domain-specific
- Single words or short phrases (2-3 words max)
- Automatically cleaned and normalized
- Stored with the asset for future reference

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

### Sprint Management

List and manage sprints for projects:

```bash
# List sprints for a project in a specific time period
assetcap sprint list --project "FN" --period "Q2 2025"

# List sprints for a project in a specific year
assetcap sprint list --project "FN" --period "2025"

# List sprints for a project in a custom date range
assetcap sprint list --project "FN" --period "2025-04-01:2025-06-30"

# Calculate time allocation for a specific sprint
assetcap sprint allocate --project "FN" --sprint "Sprint Name"
```

The sprint list command supports various period formats:

- **Quarter format**: `Q1 2025`, `Q2 2025`, `Q3 2025`, `Q4 2025`
- **Year format**: `2025` (lists all sprints in that year)
- **Date range format**: `2025-04-01:2025-06-30` (custom start and end dates)

The tool automatically:

1. Fetches all boards for the specified project
2. Retrieves sprints from each board (excluding Kanban boards that don't support sprints)
3. Filters sprints based on the specified time period
4. Displays sprint details including ID, name, dates, and state

## Installation

### Quick Install (Recommended)

**One-line installation:**

```bash
curl -sSfL https://raw.githubusercontent.com/helmedeiros/digital-asset-capitalization/main/install.sh | bash
```

This will automatically:

- Detect your platform (macOS, Linux, Windows)
- Download the latest binary
- Install dependencies (Ollama for AI features)
- Set up the tool in your PATH

**Manual installation:**

1. Download the latest binary for your platform from [GitHub Releases](https://github.com/helmedeiros/digital-asset-capitalization/releases)
2. Extract the archive: `tar -xzf assetcap_*.tar.gz` (or unzip for Windows)
3. Move the binary to your PATH: `sudo mv assetcap /usr/local/bin/`
4. Install dependencies: `curl -sSfL https://raw.githubusercontent.com/helmedeiros/digital-asset-capitalization/main/bin/install-deps.sh | bash`

**Package Managers:**

```bash
# macOS with Homebrew
brew install helmedeiros/tap/assetcap

# Linux with Snap (coming soon)
# snap install assetcap

# Windows with Chocolatey (coming soon)
# choco install assetcap
```

**Verify Installation:**

```bash
assetcap --version
assetcap --help
```

### Initial Setup

After installation, initialize your configuration:

```bash
# Interactive setup
assetcap config init

# Non-interactive setup (requires environment variables)
export JIRA_BASE_URL="https://your-domain.atlassian.net"
export JIRA_EMAIL="your.email@company.com"
export JIRA_TOKEN="your-api-token"
assetcap config init --non-interactive
```

### Shell Completion

Enable shell completion for a better CLI experience:

```bash
# For zsh users
echo 'eval "$(assetcap completion zsh)"' >> ~/.zshrc

# For bash users
echo 'eval "$(assetcap completion bash)"' >> ~/.bashrc

# For fish users
assetcap completion fish > ~/.config/fish/completions/assetcap.fish
```

## Development Setup

**For developers who want to build from source:**

### Prerequisites

- Go 1.21 or later
- Git
- Make

### Building from Source

```bash
# Clone the repository
git clone https://github.com/helmedeiros/digital-asset-capitalization.git
cd digital-asset-capitalization

# Install dependencies
go mod download

# Build and install
make install

# Install dependencies for AI features
./bin/install-deps.sh
```

### Development Commands

```bash
# Run tests
make test

# Run tests with coverage
make test-cover

# Run linter
make lint

# Generate shell completions
make completion
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
