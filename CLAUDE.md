# AssetCap - Digital Asset Capitalization Tool

A Go-based CLI tool for managing digital assets and calculating time allocation for tasks across different assets with AI-powered features.

## Project Structure

This project follows hexagonal (ports and adapters) architecture:

- **Domain Layer** (`internal/*/domain/`) - Core business logic and entities
- **Application Layer** (`internal/*/application/`) - Use cases and business rules  
- **Infrastructure Layer** (`internal/*/infrastructure/`) - External dependency adapters
- **Interface Layer** (`cmd/`) - CLI interactions and command routing

## Key Commands

### Development
```bash
# Build and test
make test           # Run tests
make test-cover     # Run tests with coverage
make build          # Build binary
make install        # Build and install

# Development tools
make lint           # Run linter
go mod tidy         # Clean dependencies
```

### Application Commands
```bash
# Configuration
assetcap config init         # Initialize configuration
assetcap config show         # Show current config
assetcap config validate     # Validate configuration
assetcap config sync-team --project "PROJECT"  # Sync team members from JIRA

# Asset management
assetcap assets create --name "Asset Name" --description "Description"
assetcap assets list
assetcap assets sync --space "SPACE" --label "cap-asset"
assetcap assets enrich --name "Asset Name" --field "description"
assetcap assets keywords --name "Asset Name"

# Task management
assetcap tasks fetch --project "PROJECT" --sprint "Sprint Name" --platform "jira"
assetcap tasks classify --project "PROJECT" --sprint "Sprint Name" --platform "jira"
assetcap tasks show --project "PROJECT" --sprint "Sprint Name"

# Sprint management
assetcap sprint list --project "PROJECT" --period "Q1 2025"
assetcap sprint allocate --project "PROJECT" --sprint "Sprint Name"
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
- Overall coverage: >80% required

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
- **Coverage Gate**: Enforces 80% minimum test coverage threshold
- **Test Execution**: Runs all tests to ensure nothing is broken

**Available Make Commands:**

```bash
# Quality checks
make lint              # Run linters
make lint-fix          # Run linters and auto-fix issues
make test              # Run tests with gotestsum
make test-cover        # Run tests with coverage report
make test-cover-gate   # Run coverage with 80% threshold check
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
- Overall project: 80% minimum (enforced by git hook)
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

## Key Features

1. **Asset Management**: Create, update, sync, and enrich digital assets
2. **Task Classification**: AI-powered classification of tasks to assets
3. **Time Allocation**: Calculate sprint time allocation for capitalization
4. **Documentation Sync**: Sync assets from Confluence spaces
5. **Keyword Generation**: AI-generated keywords for better task matching