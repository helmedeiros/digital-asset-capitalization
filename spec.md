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

---

## High-Level Architecture
- **Language:** Go
- **CLI Framework:** [spf13/cobra](https://github.com/spf13/cobra)
- **Data Persistence:** Local JSON cache files (extendable to SQLite)
- **LLM Integration:** Ollama + LLaMA 2
- **External Systems:**
  - Jira API (task data)
  - Confluence API (asset documentation)

---

## CLI Command Structure
```bash
assetcap
│
├── asset       # Manage asset metadata
├── tasks       # Classify and tag tasks
├── sprint      # Allocate effort and story points
├── report      # Generate cost and audit reports
├── keywords    # NLP-based keyword generation
├── user        # Manage developer metadata
├── data        # Import/export local data
├── test        # Run classification logic in isolation
```

---

## Key Features
### 1. Asset Management
- `asset create` / `asset update`
- `asset show` / `asset list`
- `asset sync` (from Confluence)
- `asset enrich` (field-specific using LLM)
- `asset keywords set`

### 2. Task Classification
- `tasks classify` (label tasks by work type)
- `tasks suggest-labels` (AI-powered)
- `tasks tag` / `tasks link`

### 3. Sprint Allocation
- `sprint allocate` (distribute time by story points)
- `sprint report`
- `sprint contributions`

### 4. Reporting
- `report generate` (costs per asset)
- `report totals --group-by asset`

### 5. NLP + Keyword Management
- `keywords generate`
- `keywords enrich`

### 6. Developer Cost Attribution
- `user add`
- `user assign`

---

## Asset JSON Structure
```json
{
  "id": "92a86f1ec6ef5875",
  "name": "omio-flex",
  "description": "...",
  "created_at": "2025-03-21T12:45:07.317742+01:00",
  "updated_at": "2025-03-21T12:45:07.317742+01:00",
  "last_doc_update_at": "...",
  "associated_task_count": 0,
  "version": 1,
  "platform": "pricing",
  "status": "development",
  "launch_date": "2024-06-01",
  "is_rolled_out_100": true,
  "keywords": ["refund", "cancel", "flexibility"],
  "doc_link": "https://..."
}
```

---

## Classification Logic
- Use rules to evaluate tasks:
  - If labeled as a spike, research → `cap-discovery`
  - If within 6 months or pre-100% rollout → `cap-development`
  - If adds new API/inventory → `cap-development`
  - Otherwise, if bug/fix past rollout → `cap-maintenance`
- Asset linking:
  - Use keywords in task title/description
  - Fall back to epic/component/tags
  - Else default to `cap-asset-not-applicable`

---

## LLM Prompt (Enrich Asset Field)
```text
You are enriching a metadata field called "{{FIELD_NAME}}" based on the content of a Confluence page.

- Output only the field content
- One plain-text paragraph
- No markdown, no headings
- Do not hallucinate
- Use only content present in the source
```

---

## Error Handling Strategy
- All commands return user-friendly errors
- JSON schema validation on asset/task inputs
- Fallback to manual tagging for low-confidence classification
- Confluence/LLM/API timeouts are caught with retries and warnings

---

## Testing Plan
### Unit Tests
- Asset parsing and enrichment
- Classification logic edge cases
- Time allocation calculations

### Integration Tests
- Jira mock integration
- Confluence API calls
- Asset–task linkage logic

### CLI Tests
- Golden file outputs for report generation
- JSON diff for enrich/update

---

## Future Considerations
- Switch local JSON to SQLite for scaling
- Add audit logging (who enriched, when)
- LLM model selection (switch between local and cloud)
- Web dashboard for browsing asset contributions

