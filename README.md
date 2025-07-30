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

### Version and Help

Check version information and get help:

```bash
# Show version information
assetcap version

# Show help for any command
assetcap --help
assetcap help
assetcap assets --help
assetcap tasks classify --help

# Show help for specific subcommands
assetcap assets sync --help
assetcap config init --help
assetcap sprint allocate --help
```

### Asset Management

Create and manage digital assets with comprehensive tracking:

```bash
# Create a new asset
assetcap assets create --name "Frontend App" --description "Main web application"

# List all assets
assetcap assets list

# Show detailed information about an asset
assetcap assets show --name "Frontend App"

# Update an asset's description and metadata
assetcap assets update \
  --name "Frontend App" \
  --description "Updated description" \
  --why "Improve user experience" \
  --benefits "Faster loading times" \
  --how "React optimization" \
  --metrics "Page load time < 2s"

# Sync assets from Confluence
assetcap assets sync --space "MZN" --label "cap-asset" [--debug]

# Sync from Confluence with debug output (shows detailed API calls and responses)
assetcap assets sync --space "TECH" --label "cap-asset" --debug

# Multi-space sync (sync from multiple Confluence spaces)
assetcap assets sync --space "MZN,CAP,DOC" --label "cap-asset"

# Sync from all spaces (using wildcard)
assetcap assets sync --space "*" --label "cap-asset"

# Enrich asset fields using LLaMA 3
assetcap assets enrich --name "Frontend App" --field "description"

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

### Multi-Space Sync

AssetCap supports syncing assets from multiple Confluence spaces simultaneously, providing flexibility for organizations with distributed documentation:

```bash
# Sync from single space
assetcap assets sync --space "MZN" --label "cap-asset"

# Sync from multiple specific spaces
assetcap assets sync --space "MZN,CAP,DOC" --label "cap-asset"

# Sync from all accessible spaces
assetcap assets sync --space "*" --label "cap-asset"

# Multi-space sync with debug output
assetcap assets sync --space "MZN,CAP,DOC" --label "cap-asset" --debug
```

**Multi-Space Sync Features:**

- **Single Space**: Use a single space key like `"MZN"`
- **Multiple Spaces**: Use comma-separated space keys like `"MZN,CAP,DOC"`
- **All Spaces**: Use `"*"` to sync from all accessible Confluence spaces
- **Label Filtering**: Apply consistent label filtering across all specified spaces
- **Debug Mode**: Get detailed output for each space being processed

The multi-space sync automatically:
- Processes each space independently
- Handles permissions gracefully (skips inaccessible spaces)
- Maintains consistent asset metadata across spaces
- Provides detailed logging for troubleshooting

### Sync-and-Enrich Workflow

The sync-and-enrich command provides a powerful workflow that combines asset synchronization from Confluence with AI-powered bulk enrichment operations:

```bash
# Basic sync and keyword generation
assetcap assets sync-and-enrich --label "cap-asset" --keywords

# Multi-space sync with field enrichment
assetcap assets sync-and-enrich --space "MZN,CAP,DOC" --label "cap-asset" --fields description --fields benefits

# Complete workflow with all options
assetcap assets sync-and-enrich \
  --space "MZN,CAP" \
  --label "cap-asset" \
  --keywords \
  --fields description \
  --fields why \
  --fields benefits \
  --max-concurrent 3 \
  --debug

# Preview mode (dry run)
assetcap assets sync-and-enrich --label "cap-asset" --keywords --fields description --dry-run
```

**Sync-and-Enrich Features:**

- **Multi-Space Support**: Sync from single space, multiple spaces, or all spaces
- **AI-Powered Keywords**: Bulk generation of keywords for assets using LLaMA 3
- **Field Enrichment**: Bulk enrichment of specific fields (description, why, benefits, how, metrics)
- **Smart Filtering**: Only enriches missing/empty content by default
- **Concurrent Processing**: Configurable parallelism for optimal performance
- **Dry Run Mode**: Preview what will be done without making changes
- **Progress Tracking**: Detailed logging and error handling

**Available Parameters:**

- `--space`: Confluence space key(s). Single: 'MZN', Multiple: 'MZN,CAP,DOC', All: '*' or omit
- `--label`: Confluence label to filter by (required, e.g., 'cap-asset')
- `--keywords`: Generate AI-powered keywords for synced assets
- `--fields`: Fields to enrich using AI (can be repeated for multiple fields)
- `--field-filter`: Filter strategy ('all', 'missing-fields', 'empty-fields')
- `--max-concurrent`: Maximum concurrent AI operations (default: 2)
- `--dry-run`: Show what would be done without making changes
- `--debug`: Enable detailed debug output

**Workflow Process:**

1. **Sync Phase**: Retrieves assets from specified Confluence spaces with label filtering
2. **Filter Phase**: Applies field filters to determine which assets need enrichment
3. **Enrich Phase**: Performs AI-powered keyword generation and field enrichment in parallel
4. **Save Phase**: Updates local asset storage with enriched content

### Team Management

AssetCap provides comprehensive team management functionality to track asset ownership and contributors:

```bash
# Assign a team as the owner of an asset
assetcap assets teams assign --asset "Frontend App" --owner "TeamName"

# Add a contributing team to an asset
assetcap assets teams add-contributor --asset "Frontend App" --team "TeamName"

# Remove a contributing team from an asset
assetcap assets teams remove-contributor --asset "Frontend App" --team "TeamName"

# Show team assignments for a specific asset
assetcap assets teams show --asset "Frontend App"

# List all asset team assignments
assetcap assets teams list

# Automatically sync contributors from JIRA task assignments
assetcap assets sync-contributors

# Sync contributors with filtering options
assetcap assets sync-contributors --project "FN" --sprint "Sprint 42"

# Preview sync without making changes
assetcap assets sync-contributors --dry-run

# Sync for specific asset or team
assetcap assets sync-contributors --asset "Frontend App"
assetcap assets sync-contributors --team "FrontendTeam"
```

**Team Management Features:**

- **Asset Ownership**: Assign primary ownership teams to assets
- **Contributing Teams**: Track teams that contribute to but don't own assets
- **Team Assignment Viewing**: Display team assignments for specific assets or all assets
- **Team Assignment Management**: Add and remove team assignments as needed
- **Automatic Contributor Sync**: Synchronize asset contributors from JIRA task assignments
- **Filtered Synchronization**: Sync contributors with project, sprint, team, or asset filters

**Team Assignment Structure:**

Each asset can have:
- **One Owner Team**: The primary team responsible for the asset
- **Multiple Contributing Teams**: Teams that contribute to the asset but don't own it
- **Clear Hierarchy**: Distinction between ownership and contribution roles

**Contributor Synchronization:**

The `sync-contributors` command automatically analyzes JIRA task assignments to identify which teams are actively working on assets:

- **JIRA Integration**: Analyzes task assignees to determine asset contributors
- **Smart Filtering**: Supports filtering by project, sprint, team, or specific assets
- **Dry Run Mode**: Preview changes before applying them
- **Automatic Updates**: Keeps team assignments current with actual work patterns

This helps with:
- **Accountability**: Clear ownership for asset maintenance and development
- **Collaboration**: Tracking which teams collaborate on assets
- **Resource Planning**: Understanding team workloads and asset responsibilities
- **Documentation**: Maintaining clear team-asset relationships
- **Current State**: Keeping team assignments aligned with actual JIRA work patterns

### Task Management

Comprehensive task management with JIRA integration:

```bash
# Fetch tasks from JIRA for a project and sprint
assetcap tasks fetch --project "PROJECT" --sprint "Sprint 1" --platform "jira"

# Fetch a specific task by key
assetcap tasks fetch --key "FN-1015" --platform "jira"

# Show task details for a project and sprint
assetcap tasks show --project "PROJECT" --sprint "Sprint 1"

# Show tasks filtered by asset
assetcap tasks show --asset "Frontend App"

# Classify tasks for an asset
assetcap tasks classify --project "PROJECT" --sprint "Sprint 1" --platform "jira" [--dry-run] [--apply]

# Inspect a specific task by its key
assetcap tasks inspect --key "FN-1015"

# Migrate sprint data format (for data migration)
assetcap tasks migrate [--file "path/to/tasks.json"] [--dry-run] [--stats] [--rollback]
```

The `classify` command supports the following options:

- `--dry-run`: Preview the classification without making any changes
- `--apply`: Write the classifications back to Jira as labels (e.g., cap-maintenance, cap-discovery, cap-development)

The `migrate` command helps upgrade task data format:

- `--dry-run`: Preview migration without making changes
- `--stats`: Show migration statistics without running migration
- `--rollback`: Rollback previous migration using backup file
- `--file`: Specify custom tasks.json file path (default: .assetcap/tasks.json)

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

# Using short aliases (same as above)
assetcap sprint list -p "FN" -t "Q2 2025"

# List sprints for a project in a specific year
assetcap sprint list --project "FN" --period "2025"

# List sprints for a project in a custom date range
assetcap sprint list --project "FN" --period "2025-04-01:2025-06-30"

# Calculate time allocation for a specific sprint (legacy calculation)
assetcap sprint allocate --project "FN" --sprint "Sprint Name"

# Using short aliases
assetcap sprint allocate -p "FN" -s "Sprint Name"

# Calculate time allocation with sprint-bounded calculation
assetcap sprint allocate --project "FN" --sprint "Sprint Name" --sprint-bounded

# Using short aliases with sprint-bounded calculation
assetcap sprint allocate -p "FN" -s "Sprint Name" -sb

# Calculate with manual overrides
assetcap sprint allocate --project "FN" --sprint "Sprint Name" \
  --override '{"ISSUE-1": 6, "ISSUE-2": 36}'

# Using short aliases with manual overrides
assetcap sprint allocate -p "FN" -s "Sprint Name" \
  -o '{"ISSUE-1": 6, "ISSUE-2": 36}' -sb
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

### Configuration Management

Manage application configuration and settings:

```bash
# Initialize configuration interactively
assetcap config init

# Initialize configuration non-interactively (using environment variables)
assetcap config init --non-interactive \
  --jira-url "https://company.atlassian.net" \
  --jira-email "user@company.com" \
  --jira-token "api-token"

# Non-interactive initialization with individual flags
assetcap config init --non-interactive --jira-url "https://company.atlassian.net"
assetcap config init --non-interactive --jira-email "user@company.com"
assetcap config init --non-interactive --jira-token "api-token"

# Show current configuration
assetcap config show

# Validate current configuration
assetcap config validate

# Synchronize team members from JIRA for a project
assetcap config sync-team --project "FN"

# Using short alias
assetcap config sync-team -p "AD"
```

The configuration commands help you:

- Set up JIRA integration credentials
- Create and manage teams.json configuration
- Validate configuration before running other commands
- View current configuration status (with masked sensitive data)
- Synchronize team members from JIRA projects to local teams.json configuration

**Config Init Flags:**

- `--non-interactive`: Run without prompts (requires environment variables or flags)
- `--jira-url`: Specify JIRA base URL directly
- `--jira-email`: Specify JIRA email directly
- `--jira-token`: Specify JIRA API token directly

**Note**: When using `--non-interactive`, you can either set environment variables (`JIRA_BASE_URL`, `JIRA_EMAIL`, `JIRA_TOKEN`) or use the flag options above.

**Team Synchronization:**

The `config sync-team` command automatically extracts active team members from JIRA projects and updates your local teams.json configuration. It identifies team members by analyzing recent issue assignments rather than all assignable users, providing more accurate team membership.

```bash
# Sync team members for FN project
assetcap config sync-team --project "FN"

# Sync team members for AD project  
assetcap config sync-team --project "AD"

# Using short alias
assetcap config sync-team -p "PROJECT_KEY"
```

The sync process:
- Queries JIRA for recent issues in the specified project
- Extracts unique assignees from those issues
- Filters to active users only
- Updates the teams.json file with the current team membership
- Shows a summary of added/removed members

### Shell Completion

Generate shell completion scripts for better CLI experience:

```bash
# Generate bash completion
assetcap completion bash

# Generate zsh completion
assetcap completion zsh

# Generate fish completion
assetcap completion fish
```

Install completion scripts:

```bash
# For zsh users
echo 'eval "$(assetcap completion zsh)"' >> ~/.zshrc

# For bash users
echo 'eval "$(assetcap completion bash)"' >> ~/.bashrc

# For fish users
assetcap completion fish > ~/.config/fish/completions/assetcap.fish
```

## Advanced Usage Examples

### Complex Task Classification Workflow

```bash
# 1. Fetch tasks for a specific project and sprint
assetcap tasks fetch --project "FN" --sprint "Sprint 42" --platform "jira"

# 2. Preview classification before applying (dry-run)
assetcap tasks classify --project "FN" --sprint "Sprint 42" --platform "jira" --dry-run

# 3. Apply classifications to JIRA (adds labels like cap-development, cap-maintenance)
assetcap tasks classify --project "FN" --sprint "Sprint 42" --platform "jira" --apply

# 4. Inspect specific tasks for detailed information
assetcap tasks inspect --key "FN-1015"
assetcap tasks inspect --key "FN-1023"

# 5. Show tasks filtered by asset
assetcap tasks show --asset "Frontend App"
```

### Sprint Time Allocation Scenarios

```bash
# Basic time allocation (legacy method)
assetcap sprint allocate --project "FN" --sprint "Sprint 42"

# Sprint-bounded calculation (respects sprint dates)
assetcap sprint allocate --project "FN" --sprint "Sprint 42" --sprint-bounded

# With manual overrides for specific issues
assetcap sprint allocate --project "FN" --sprint "Sprint 42" \
  --override '{"FN-1015": 6, "FN-1023": 36, "FN-1024": 12}'

# List sprints with different time period formats
assetcap sprint list --project "FN" --period "Q1 2025"        # Quarter
assetcap sprint list --project "FN" --period "2025"           # Full year
assetcap sprint list --project "FN" --period "2025-01-01:2025-03-31"  # Custom range
```

### Data Migration Examples

```bash
# Check migration statistics without making changes
assetcap tasks migrate --stats

# Check migration statistics for a specific file
assetcap tasks migrate --file "/path/to/custom/tasks.json" --stats

# Preview what the migration would do
assetcap tasks migrate --dry-run

# Preview migration for a specific file
assetcap tasks migrate --file "/path/to/custom/tasks.json" --dry-run

# Run migration with backup (recommended)
assetcap tasks migrate

# Migrate specific file
assetcap tasks migrate --file "/path/to/custom/tasks.json"

# Rollback if something goes wrong
assetcap tasks migrate --rollback

# Rollback specific file migration
assetcap tasks migrate --file "/path/to/custom/tasks.json" --rollback
```

### Asset Management Workflows

```bash
# Complete asset creation and enrichment workflow
assetcap assets create --name "Payment Gateway" --description "Secure payment processing system"

# Sync from Confluence with debug output
assetcap assets sync --space "TECH" --label "cap-asset" --debug

# Enrich different fields using AI
assetcap assets enrich --name "Payment Gateway" --field "description"
assetcap assets enrich --name "Payment Gateway" --field "benefits"
assetcap assets enrich --name "Payment Gateway" --field "metrics"

# Generate keywords for better task matching
assetcap assets keywords --name "Payment Gateway"

# Update comprehensive asset information
assetcap assets update \
  --name "Payment Gateway" \
  --description "Secure payment processing system with fraud detection" \
  --why "Reduce payment fraud and improve user trust" \
  --benefits "15% reduction in fraudulent transactions" \
  --how "Machine learning-based fraud detection algorithms" \
  --metrics "Fraud detection rate > 95%, false positive rate < 2%"

# Track task associations
assetcap assets tasks increment --asset "Payment Gateway"
assetcap assets documentation update --asset "Payment Gateway"
```

### Multi-Space and Bulk Operations

```bash
# Complete multi-space sync and enrichment workflow
assetcap assets sync-and-enrich \
  --space "MZN,CAP,DOC" \
  --label "cap-asset" \
  --keywords \
  --fields description \
  --fields benefits \
  --fields metrics \
  --max-concurrent 3 \
  --debug

# Bulk keyword generation for existing assets
assetcap assets sync-and-enrich --space "*" --label "cap-asset" --keywords --dry-run

# Multi-space sync with selective field enrichment
assetcap assets sync-and-enrich \
  --space "TECH,DOCS" \
  --label "cap-asset" \
  --fields description \
  --field-filter "empty-fields" \
  --max-concurrent 2
```

### Team Management Workflows

```bash
# Complete team assignment workflow
assetcap assets teams assign --asset "Payment Gateway" --owner "BackendTeam"
assetcap assets teams add-contributor --asset "Payment Gateway" --team "SecurityTeam"
assetcap assets teams add-contributor --asset "Payment Gateway" --team "DevOpsTeam"

# Sync contributors based on actual JIRA work
assetcap assets sync-contributors --project "FN" --sprint "Sprint 42" --dry-run
assetcap assets sync-contributors --project "FN" --sprint "Sprint 42"

# Comprehensive team management review
assetcap assets teams list
assetcap assets teams show --asset "Payment Gateway"

# Sync contributors for specific team's assets
assetcap assets sync-contributors --team "BackendTeam" --max-results 500
```

### Configuration and Troubleshooting

```bash
# Setup configuration step by step
assetcap config init --non-interactive \
  --jira-url "https://company.atlassian.net" \
  --jira-email "team@company.com" \
  --jira-token "your-api-token"

# Verify configuration is working
assetcap config validate

# Check current settings (sensitive data masked)
assetcap config show

# Test JIRA connectivity by fetching a single task
assetcap tasks fetch --key "TEST-1" --platform "jira"
```

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

### Shell Completion Setup

After installation, enable shell completion for a better CLI experience (see Shell Completion section above for more details).

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
