# Jira Time Allocation Calculator

A tool to calculate time allocation for Jira issues based on their status changes and manual adjustments.

## What is this?

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

### Test Coverage Report

The current test coverage is:

- `assetcap/action`: 88.2%
- `assetcap/config`: 96.0%
- `assetcap`: 18.4%

## Usage

Run the tool with:

```bash
go run cmd/main.go -project PROJECT_KEY -sprint "Sprint Name" [-override '{"ISSUE-KEY": hours}']
```

The tool will generate a CSV file with time allocation percentages for each team member.

## Security Note

For better security, you can add these environment variables to your shell's configuration file (e.g., `~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish`). This way, they'll be available in your shell sessions without being stored in any files in the project directory.
