# Digital Asset Capitalization Tool

A tool to manage digital assets and calculate time allocation for tasks across different assets.

## What is this?

This tool helps track and manage digital assets in your organization, including:

1. Asset lifecycle management
2. Time allocation tracking for tasks
3. Documentation management
4. Task classification and management

The tool automatically calculates the time allocation for tasks in each sprint and helps manage the capitalization of digital assets.

## Installation

To install the `assetcap` command globally:

```bash
# Clone the repository
git clone https://github.com/helmedeiros/digital-asset-capitalization.git
cd digital-asset-capitalization

# Install the command
make install
```

This will install the `assetcap` command in your Go bin directory (`$GOPATH/bin`). Make sure this directory is in your PATH.

You can verify the installation by running:

```bash
assetcap --version
```

## Shell Completion

The tool supports shell completion for bash, zsh, and fish shells. To install completions:

```bash
make completion
```

### Zsh

The completion script will be installed to `~/.zsh/completion/_assetcap`. Add these lines to your `~/.zshrc`:

```bash
fpath=(~/.zsh/completion $fpath)
autoload -U compinit && compinit
```

### Bash

The completion script will be saved to `completions/assetcap.bash`. Add this line to your `~/.bashrc`:

```bash
source /path/to/digital-asset-capitalization/completions/assetcap.bash
```

### Fish

The completion script will be saved to `completions/assetcap.fish`. Copy it to the fish completions directory:

```bash
cp completions/assetcap.fish ~/.config/fish/completions/
```

## Features

### Asset Management

The tool provides comprehensive asset management capabilities through the CLI:

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

### Time Allocation

This tool automatically calculates the time it took the developer
to complete the task in the sprint in % of the total sprint time. The tool:

1. Fetches tasks from JIRA for a specific project and sprint
2. Calculates time allocation based on task completion
3. Generates a formatted output that can be copied to JIRA's "Time Allocation %" field
4. Supports integration with Google Spreadsheets for team-wide tracking

To use the time allocation feature:

```bash
# Fetch and classify tasks for a project and sprint
assetcap tasks classify --project "PROJECT" --sprint "Sprint 1"

# View the calculated time allocation
assetcap tasks show --project "PROJECT" --sprint "Sprint 1"
```

### Task Management

The tool provides comprehensive task management capabilities:

```bash
# Fetch tasks from JIRA
assetcap tasks fetch --project "PROJECT" --sprint "Sprint 1"

# Classify tasks for an asset
assetcap tasks classify --project "PROJECT" --sprint "Sprint 1"

# Show task details
assetcap tasks show --project "PROJECT" --sprint "Sprint 1"
```

## Architecture

The project follows a hexagonal (ports and adapters) architecture pattern, which provides several benefits:

### Hexagonal Architecture

The codebase is organized into distinct layers:

1. **Domain Layer** (`internal/*/domain/`)

   - Contains the core business logic and entities
   - Defines the domain models and interfaces
   - No dependencies on external frameworks or libraries
   - Includes both assets and tasks domain models

2. **Application Layer** (`internal/*/application/`)

   - Implements use cases and business rules
   - Orchestrates the flow of data and domain objects
   - Defines ports (interfaces) for external dependencies
   - Handles both asset and task operations

3. **Infrastructure Layer** (`internal/*/infrastructure/`)

   - Implements the adapters for external dependencies
   - Handles persistence, external services, and frameworks
   - Conforms to the interfaces defined in the application layer
   - Includes JIRA integration and local storage

4. **Interface Layer** (`assetcap/action/`)
   - Handles user interface and command-line interactions
   - Routes commands to appropriate application services
   - Provides a clean API for external users

### Persistent Ancillaries

The tool uses persistent ancillaries to maintain state between runs:

1. **Asset Storage**

   - Assets are stored in `.assetcap/assets.json`
   - JSON-based storage for easy inspection and backup
   - Thread-safe operations with proper file locking

2. **Configuration**

   - Team configurations in `teams.json`
   - Environment variables for sensitive data
   - Template-based configuration for easy setup

3. **Documentation**
   - Generated documentation stored in `.assetcap/docs/`
   - Supports multiple output formats (Confluence, Markdown)
   - Version-controlled documentation templates

### Benefits of the Architecture

1. **Separation of Concerns**

   - Clear boundaries between layers
   - Easy to understand and maintain
   - Independent testing of each layer

2. **Dependency Inversion**

   - Core business logic is independent of external concerns
   - Easy to swap implementations (e.g., different storage backends)
   - Better testability through interface-based design

3. **Flexibility**
   - Easy to add new features without modifying existing code
   - Simple to integrate with new external services
   - Clear upgrade paths for future enhancements

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
       "Members": ["Team Member 1", "Team Member 2"],
       "SprintDuration": "2w",
       "WorkingHoursPerDay": 8
     }
   }
   ```
4. Set up your Jira credentials as environment variables:
   ```bash
   export JIRA_BASE_URL="https://your-domain.atlassian.net"
   export JIRA_EMAIL="your.email@company.com"
   export JIRA_TOKEN="your-api-token"
   ```
5. The tool will automatically create a `.assetcap` directory in your home folder to store:
   - Asset data (`assets.json`)
   - Task data (`tasks.json`)
   - Generated documentation (`docs/`)

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
make bench
```

### Test Coverage Requirements

The project maintains high test coverage requirements:

- Domain layer: >90% coverage
- Application layer: >80% coverage
- Infrastructure layer: >80% coverage
- Overall coverage target: >80%

### Test Utilities

The project includes test utilities in `internal/*/application/usecase/testutil/`:

- Mock implementations for repositories
- Test helpers for common scenarios
- Utilities for setting up test data

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
