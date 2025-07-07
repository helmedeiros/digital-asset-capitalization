# AssetCap: Asset Capitalization CLI Application Specification

## Overview

**AssetCap** is a terminal-based Go application designed to manage software asset capitalization. It supports labeling, tracking, and cost allocation for software projects and tasks, and adheres to rules compliant with German government standards and internal asset evaluation frameworks.

---

## Objectives

- Classify development tasks by work type: discovery, development, maintenance
- Identify and manage capitalizable assets
- Attribute development time and cost per asset
- Support auditors with structured, exportable reports
- Enrich asset metadata using Confluence and LLMs (via Ollama)
- Provide sprint-based time allocation calculations
- Integrate with JIRA for task management and classification

---

## High-Level Architecture

- **Language:** Go
- **CLI Framework:** [urfave/cli](https://github.com/urfave/cli)
- **Data Persistence:** Local JSON cache files (`.assetcap/assets.json`, `.assetcap/tasks.json`)
- **LLM Integration:** Ollama + LLaMA 3
- **External Systems:**
  - JIRA API (task data and classification)
  - Confluence API (asset documentation)
- **Architecture Pattern:** Hexagonal (ports and adapters)

---

## CLI Command Structure

```bash
assetcap
│
├── version          # Show version information
├── completion       # Generate shell completion scripts
│   ├── bash
│   ├── zsh
│   └── fish
├── config           # Manage configuration settings
│   ├── init
│   ├── show
│   └── validate
├── assets           # Manage digital assets
│   ├── create
│   ├── list
│   ├── show
│   ├── update
│   ├── sync
│   ├── enrich
│   ├── keywords
│   ├── documentation
│   │   └── update
│   └── tasks
│       ├── increment
│       └── decrement
├── tasks            # Manage tasks from various platforms
│   ├── fetch
│   ├── show
│   ├── classify
│   ├── inspect
│   └── migrate
└── sprint           # Manage sprint-related operations
    ├── list
    └── allocate
```

---

## Key Features

### 1. Asset Management

- **Create/Update Assets**: `assets create`, `assets update`
- **Asset Discovery**: `assets list`, `assets show`
- **Confluence Integration**: `assets sync` (from Confluence pages)
- **AI Enhancement**: `assets enrich` (field-specific using LLM)
- **Keyword Generation**: `assets keywords` (AI-powered)
- **Documentation Tracking**: `assets documentation update`
- **Task Count Management**: `assets tasks increment/decrement`

### 2. Task Classification and Management

- **JIRA Integration**: `tasks fetch` (from JIRA projects/sprints)
- **AI Classification**: `tasks classify` (label tasks by work type)
- **Task Inspection**: `tasks inspect` (detailed task view)
- **Data Migration**: `tasks migrate` (upgrade data formats)
- **Asset Filtering**: `tasks show --asset` (filter by asset)

### 3. Sprint Management

- **Sprint Discovery**: `sprint list` (with time period filtering)
- **Time Allocation**: `sprint allocate` (legacy and sprint-bounded)
- **Manual Overrides**: Support for manual time allocation adjustments

### 4. Configuration Management

- **Setup**: `config init` (interactive and non-interactive)
- **Validation**: `config validate` (check configuration)
- **Status**: `config show` (display current settings)

### 5. Shell Integration

- **Completion**: `completion bash/zsh/fish` (shell completion scripts)
- **Help System**: Comprehensive help at all command levels

---

## Asset JSON Structure

```json
{
  "id": "92a86f1ec6ef5875",
  "name": "omio-flex",
  "description": "Flexible booking management system",
  "why": "Improve customer satisfaction with flexible booking options",
  "benefits": "Increased customer retention and booking flexibility",
  "how": "API-based booking modification system",
  "metrics": "Customer satisfaction score > 8.5",
  "created_at": "2025-03-21T12:45:07.317742+01:00",
  "updated_at": "2025-03-21T12:45:07.317742+01:00",
  "last_doc_update_at": "2025-03-21T12:45:07.317742+01:00",
  "associated_task_count": 0,
  "version": 1,
  "keywords": ["refund", "cancel", "flexibility", "booking"],
  "doc_link": "https://confluence.company.com/pages/viewpage.action?pageId=123456"
}
```

---

## Task JSON Structure

```json
{
  "key": "FN-1015",
  "type": "Story",
  "summary": "Implement flexible booking cancellation",
  "description": "Allow customers to cancel bookings with flexible penalties",
  "status": "Done",
  "project": "FN",
  "sprint": ["Sprint 42", "Sprint 43"],
  "epic": "Flexible Booking",
  "work_type": "cap-development",
  "priority": "High",
  "platform": "jira",
  "labels": ["cap-development", "omio-flex"],
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-20T16:45:00Z",
  "version": 1
}
```

---

## Classification Logic

### Work Type Classification

- **Discovery (`cap-discovery`)**: Spikes, research, POCs, analysis tasks
- **Development (`cap-development`)**: New features, APIs, within 6 months of rollout
- **Maintenance (`cap-maintenance`)**: Bug fixes, maintenance tasks after rollout

### Asset Association

1. **Keyword Matching**: Match task title/description with asset keywords
2. **Epic/Component Matching**: Use JIRA epic and component fields
3. **Manual Classification**: User-defined overrides and classifications
4. **Fallback**: `cap-asset-not-applicable` for unmatched tasks

### Business Rules

- New functionality before 100% rollout → `cap-development`
- Bug fixes after rollout → `cap-maintenance`
- Research and spikes → `cap-discovery`
- Tasks with asset keywords → Associated with matching asset

---

## Sprint Time Allocation

### Legacy Calculation

- Equal distribution of time across sprint tasks
- Based on story points and task completion

### Sprint-Bounded Calculation

- Respects sprint date boundaries
- Allocates time only within sprint duration
- Handles cross-sprint tasks appropriately

### Manual Overrides

- JSON format: `{"ISSUE-1": 6, "ISSUE-2": 36}`
- Override specific task time allocations
- Preserve automated calculations for other tasks

---

## Configuration

### Environment Variables

```bash
JIRA_BASE_URL="https://company.atlassian.net"
JIRA_EMAIL="user@company.com"
JIRA_TOKEN="api-token"
```

### Teams Configuration (`teams.json`)

```json
{
  "PROJECT_KEY": {
    "Members": ["Team Member 1", "Team Member 2"],
    "SprintDuration": "2w",
    "WorkingHoursPerDay": 8
  }
}
```

### Local Storage

- **Assets**: `.assetcap/assets.json`
- **Tasks**: `.assetcap/tasks.json`
- **Configuration**: `.assetcap/config.json`
- **Backups**: `.assetcap/backups/`

---

## LLM Integration

### Asset Enrichment

```text
You are enriching a metadata field called "{{FIELD_NAME}}" based on the content of a Confluence page.

- Output only the field content
- One plain-text paragraph
- No markdown, no headings
- Do not hallucinate
- Use only content present in the source
```

### Keyword Generation

- Generates 5-10 relevant technical keywords
- Considers asset metadata and documentation
- Used for task-to-asset matching
- Automatically cleaned and normalized

---

## Error Handling Strategy

- User-friendly error messages for CLI operations
- JSON schema validation on asset/task inputs
- Graceful degradation for external service failures
- Retry logic for API calls with exponential backoff
- Comprehensive logging for debugging

---

## Testing Strategy

### Unit Tests

- Domain logic validation
- Classification algorithm testing
- Asset/task parsing and validation
- Time allocation calculations

### Integration Tests

- JIRA API integration
- Confluence API integration
- Local storage operations
- End-to-end command execution

### CLI Tests

- Command parsing and validation
- Output formatting verification
- Error handling scenarios
- Help text accuracy

---

## Data Migration

### Sprint Data Migration

- Migrates comma-separated sprint strings to arrays
- Supports dry-run mode for preview
- Creates automatic backups
- Rollback capability for failed migrations
- Statistics reporting for migration progress

### Migration Commands

```bash
# Preview migration
assetcap tasks migrate --dry-run

# Show migration statistics
assetcap tasks migrate --stats

# Run migration with backup
assetcap tasks migrate

# Rollback migration
assetcap tasks migrate --rollback
```

---

## Future Considerations

- Switch local JSON to SQLite for improved performance
- Add audit logging (user actions and timestamps)
- Web dashboard for asset and task visualization
- Advanced reporting and analytics
- Multi-tenant support for large organizations
- Integration with additional project management tools
