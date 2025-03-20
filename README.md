# Jira Time Allocation Calculator

A tool that calculates the time allocation percentage for tasks in a Jira sprint.

## What is this?

This tool automatically calculates the time it took the developer
to complete the task in the sprint in % of the total sprint time, as later fill it into "Time Allocation %" in JIRA.

Simply copy-paste the output to the Google Spreadsheet with the split per columns.

## Setup

1. Set up your Jira credentials as environment variables in your shell:

   ```bash
   # For bash/zsh
   export JIRA_BASE_URL="https://your-domain.atlassian.net"
   export JIRA_EMAIL="your.email@company.com"
   export JIRA_TOKEN="your-api-token"

   # For fish shell
   set -x JIRA_BASE_URL "https://your-domain.atlassian.net"
   set -x JIRA_EMAIL "your.email@company.com"
   set -x JIRA_TOKEN "your-api-token"
   ```

   You can get your Jira API token from https://id.atlassian.com/manage-profile/security/api-tokens

2. Copy `teams.json.template` to `teams.json` and add your team members:
   ```bash
   cp teams.json.template teams.json
   ```
   > NB! You have to include names with their diacritics as they are in Jira, with ć, á or others depending on the contributor's name, otherwise the tool won't be able to match issues and contributors together.

## How to use it? Example

```bash
./assetcap-calc timeallocation-calc --project "YOUR_PROJECT" --sprint "YOUR_SPRINT" [--override '{"PROJECT-123": 1}']
```

## Security Note

For better security, you can add these environment variables to your shell's configuration file (e.g., `~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish`). This way, they'll be available in your shell sessions without being stored in any files in the project directory.
