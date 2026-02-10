# AssetCap - Digital Asset Capitalization Tool

A Go-based CLI tool for managing digital assets and calculating time allocation for tasks across different assets with AI-powered features.

## 🚨 CRITICAL: Binary Usage

**NEVER use `./main` - it's an outdated binary!**

Always use the latest binary with one of these approaches:

### Option 1: Build-then-run (Recommended)
```bash
# Always build first, then run
make build && ./assetcap [commands]

# Or use the convenient wrapper
./bin/run-assetcap.sh [commands]
```

### Option 2: Direct build-run
```bash
# Build and run in one command
make build-run ARGS="assets teams --help"
make build-run ARGS="assets show --name 'Asset Name'"
```

### Option 3: Install and use globally
```bash
# Install to GOPATH (less preferred for development)
make install
assetcap [commands]
```

## Project Structure

This project follows hexagonal (ports and adapters) architecture:

- **Domain Layer** (`internal/*/domain/`) - Core business logic and entities
- **Application Layer** (`internal/*/application/`) - Use cases and business rules  
- **Infrastructure Layer** (`internal/*/infrastructure/`) - External dependency adapters
- **Interface Layer** (`cmd/`) - CLI interactions and command routing

## Key Commands

### Development Workflow

**Always follow this 3-step process:**
1. **Build first**: `make build`
2. **Run latest binary**: `./assetcap [commands]`
3. **For testing**: Use `./bin/run-assetcap.sh` wrapper

```bash
# Build and test
make build          # Build latest binary
make test           # Run tests
make test-cover     # Run tests with coverage
make install        # Build and install globally

# Quality checks (run before pushing)
make lint           # Run linter
make lint-fix       # Run linter and auto-fix
make pre-push       # Full quality gate (required before push)

# Development tools
go mod tidy         # Clean dependencies
make build-run ARGS="command"  # Build and run in one step
```

### Application Commands

**Remember to use `./assetcap` after `make build` or use build-run patterns:**

```bash
# Configuration
./assetcap config init         # Initialize configuration
./assetcap config show         # Show current config
./assetcap config validate     # Validate configuration
./assetcap config sync-team --project "PROJECT"  # Sync team members from JIRA

# Asset management
./assetcap assets create --name "Asset Name" --description "Description"
./assetcap assets list
./assetcap assets sync --space "MZN" --label "cap-asset"
./assetcap assets enrich --name "Asset Name" --field "description"
./assetcap assets keywords --name "Asset Name"

# Bulk asset sync and enrichment (NEW)
./assetcap assets sync-and-enrich --label "cap-asset" --keywords --fields description --fields benefits
./assetcap assets sync-and-enrich --space "MZN,CAP" --label "cap-asset" --keywords --dry-run
./assetcap assets sync-and-enrich --label "cap-asset" --fields description --fields why --fields benefits --max-concurrent 3

# Team management (latest features)
./assetcap assets teams assign --asset "Asset Name" --owner "TeamName"
./assetcap assets teams add-contributor --asset "Asset Name" --team "TeamName"
./assetcap assets teams show --asset "Asset Name"
./assetcap assets teams list

# Task management
./assetcap tasks fetch --project "PROJECT" --sprint "Sprint Name" --platform "jira"
./assetcap tasks classify --project "PROJECT" --sprint "Sprint Name" --platform "jira"
./assetcap tasks show --project "PROJECT" --sprint "Sprint Name"

# Sprint management
./assetcap sprint list --project "PROJECT" --period "Q1 2025"
./assetcap sprint allocate --project "PROJECT" --sprint "Sprint Name"

# Using build-run pattern (alternative)
make build-run ARGS="assets teams --help"
make build-run ARGS="assets show --name 'Asset Name'"
```

## Configuration

The tool uses:
- `.assetcap/` directory for data storage (assets.json, tasks.json)
- `teams.json` for team configuration
- Environment variables for JIRA credentials:
  - `JIRA_BASE_URL`
  - `JIRA_EMAIL` 
  - `JIRA_TOKEN`

## Dependencies

- **AI Features**: Uses LLaMA 3 via Ollama for asset enrichment and keyword generation
- **JIRA Integration**: REST API integration for task fetching and classification
- **Confluence Integration**: For asset synchronization from documentation

## Testing

- Domain layer: >90% coverage required
- Application layer: >80% coverage required  
- Infrastructure layer: >80% coverage required
- Overall coverage: >70% required

## Architecture Guidelines

**IMPORTANT: Always follow hexagonal (ports and adapters) architecture pattern for all implementations:**

### Layer Responsibilities

1. **Domain Layer** (`internal/*/domain/`)
   - Contains core business entities and value objects
   - Defines ports (interfaces) for external dependencies
   - Pure business logic with NO external dependencies
   - Example: `Asset`, `Task`, `Sprint` entities and repository interfaces

2. **Application Layer** (`internal/*/application/`)
   - Contains use cases and application services
   - Orchestrates domain objects and calls ports
   - Implements business workflows
   - Example: `CreateAssetUseCase`, `ClassifyTasksUseCase`

3. **Infrastructure Layer** (`internal/*/infrastructure/`)
   - Implements adapters for external systems
   - Contains concrete implementations of domain ports
   - Handles persistence, HTTP clients, file I/O
   - Example: `JiraTaskRepository`, `JsonAssetRepository`

4. **Interface Layer** (`cmd/`)
   - CLI command handlers and routing
   - Dependency injection and wiring
   - User input/output handling

### Implementation Rules

- **Domain**: Never import from application or infrastructure layers
- **Application**: Can import domain, but never infrastructure directly
- **Infrastructure**: Can import domain and application interfaces
- **Interface**: Can import all layers for dependency injection

### When Adding New Features

**IMPORTANT: Always create a new branch for feature development:**

1. **Create feature branch** from latest main/master:
   ```bash
   git checkout main
   git pull origin main
   git checkout -b feature/descriptive-feature-name
   ```

2. Start with domain entities and ports
3. Create application use cases
4. Implement infrastructure adapters
5. Wire everything in the interface layer

### Git Workflow Rules

- **Always** create a new branch for any feature development
- Branch naming convention: `feature/descriptive-name` or `fix/issue-description`
- Ensure branch is based on latest main/master branch
- Never commit directly to main/master branch
- **NEVER** include co-author information in commit messages for this project
- **NEVER** include icons, markdown, or quality assurance information in commit messages
- Commit messages should focus on: what the commit solves, why the decisions were made, and how the problem was solved

### Before Pushing Feature Branch

**MANDATORY: Always check for updates, run quality checks, and rebase before pushing:**

```bash
# 1. Check for updates on main/master
git checkout main
git pull origin main

# 2. Switch back to feature branch
git checkout feature/your-feature-name

# 3. Rebase feature branch on latest main
git rebase main

# 4. Resolve any conflicts if they exist
# Edit conflicted files, then:
git add .
git rebase --continue

# 5. CRITICAL: Run quality checks before pushing
make pre-push

# 6. Push feature branch (force push if rebased)
git push origin feature/your-feature-name --force-with-lease
```

### Quality Gates (Enforced by Git Hooks)

**Pre-push hook automatically runs `make pre-push` which includes:**

- **Linting**: `golangci-lint` checks for code quality issues
- **Coverage Gate**: Enforces 70% minimum test coverage threshold
- **Test Execution**: Runs all tests to ensure nothing is broken

**Available Make Commands:**

```bash
# Quality checks
make lint              # Run linters
make lint-fix          # Run linters and auto-fix issues
make test              # Run tests with gotestsum
make test-cover        # Run tests with coverage report
make test-cover-gate   # Run coverage with 70% threshold check
make pre-push          # Full quality gate (lint + coverage gate)

# Development
make install           # Build and install assetcap binary
make completion        # Generate shell completions

# Testing variants
make test-race         # Run tests with race detector
make test-watch        # Run tests in watch mode
make test-all          # Run tests with race detector and coverage
```

**Coverage Requirements:**
- Overall project: 70% minimum (enforced by git hook)
- Domain layer: >90% (architectural requirement)
- Application layer: >80% (architectural requirement)
- Infrastructure layer: >80% (architectural requirement)

**Why this matters:**
- Prevents merge conflicts in pull requests
- Keeps feature branch up-to-date with latest changes
- Maintains clean, linear git history
- **Ensures code quality through automated gates**
- **Blocks push if tests fail or coverage drops**
- Reduces integration issues

## Important Notes

- **Binary Usage**: The source code has the latest team management features that old binaries lack
- **Build First**: Building ensures you get all recent changes and prevents "command not found" errors
- **Permission System**: The project explicitly denies usage of `./main` binary through configuration
- **Development Wrapper**: Use `./bin/run-assetcap.sh` for convenient testing during development

## Key Features

1. **Asset Management**: Create, update, sync, and enrich digital assets
2. **Task Classification**: AI-powered classification of tasks to assets
3. **Time Allocation**: Calculate sprint time allocation for capitalization
4. **Documentation Sync**: Sync assets from Confluence spaces with multi-space support
5. **Keyword Generation**: AI-generated keywords for better task matching
6. **Bulk Asset Enrichment**: Sync and enrich multiple assets with AI-powered content and keywords
7. **Intelligent Filtering**: Only enrich missing content to preserve existing data
8. **Concurrent Processing**: Configurable parallel AI operations for efficiency

## Sync-and-Enrich Workflow (NEW)

The `sync-and-enrich` command provides a powerful workflow that combines asset synchronization from Confluence with AI-powered bulk enrichment:

### Features
- **Multi-Space Sync**: Support for single space, multiple spaces, or all spaces
- **AI-Powered Keywords**: Bulk generation of keywords for assets using LLaMA 3
- **Field Enrichment**: Bulk enrichment of specific fields (description, why, benefits, how, metrics)
- **Smart Filtering**: Only enriches missing/empty content by default
- **Concurrent Processing**: Configurable parallelism for optimal performance
- **Dry Run Mode**: Preview what will be done without making changes
- **Progress Tracking**: Detailed logging and error handling

### Usage Examples

```bash
# Basic sync and keyword generation
./assetcap assets sync-and-enrich --label "cap-asset" --keywords

# Multi-space sync with field enrichment
./assetcap assets sync-and-enrich --space "MZN,CAP,DOC" --label "cap-asset" --fields description --fields benefits

# Full workflow with all options
./assetcap assets sync-and-enrich \
  --space "MZN,CAP" \
  --label "cap-asset" \
  --keywords \
  --fields description \
  --fields why \
  --fields benefits \
  --max-concurrent 3 \
  --debug

# Preview mode (dry run)
./assetcap assets sync-and-enrich --label "cap-asset" --keywords --fields description --dry-run
```

### Parameters
- `--space`: Confluence space key(s). Single: 'MZN', Multiple: 'MZN,CAP,DOC', All: '*' or omit
- `--label`: Confluence label to filter by (required, e.g., 'cap-asset')
- `--keywords`: Generate AI-powered keywords for synced assets
- `--fields`: Fields to enrich using AI (can be repeated for multiple fields)
- `--field-filter`: Filter strategy ('all', 'missing-fields', 'empty-fields')
- `--max-concurrent`: Maximum concurrent AI operations (default: 2)
- `--dry-run`: Show what would be done without making changes
- `--debug`: Enable detailed debug output

## AI Console Mode (In Development)

The AssetCap AI Console provides natural language interaction with the tool, allowing users to execute commands conversationally:

### Overview
- **Natural Language**: Use plain English instead of CLI syntax
- **Context Awareness**: Remembers previous commands and entities
- **Smart Interpretation**: AI-powered command translation using LLaMA 3
- **Interactive Help**: Ask questions about capabilities

### Architecture
The AI Console follows the same hexagonal architecture:
- **Domain Layer** (`internal/console/domain/`) - Command and context entities
- **Application Layer** (`internal/console/application/`) - Console service and use cases
- **Infrastructure Layer** (`internal/console/infrastructure/`) - AI interpreter and executor
- **Interface Layer** (`cmd/console.go`) - Console command entry point

### Usage (Coming Soon)
```bash
# Start interactive console
./assetcap console

# Example interactions:
> Show me all assets
> Create an asset called Payment Processing
> What tasks are assigned to it?
> Calculate investment for the last 3 sprints
```

### Key Components
- **AI Interpreter**: Converts natural language to CLI commands using LLaMA 3
- **Command Executor**: Runs interpreted commands using existing services
- **Context Store**: Maintains conversation state for follow-up questions
- **Prompt Handler**: Interactive UI with command history

### Implementation Status
- Documentation: ✅ Complete (see `docs/ai-console/`)
- Domain Design: 🔄 In Progress
- Infrastructure: 📋 Planned
- Testing: 📋 Planned

For implementation details, see:
- `docs/ai-console/IMPLEMENTATION.md` - Architecture and design
- `docs/ai-console/PROMPT_ENGINEERING.md` - AI prompt templates
- `docs/ai-console/CONTEXT_GUIDE.md` - Context management