# AI Console Prompt Engineering Guide

## Overview

This document contains the prompt templates and strategies for the AI Console's natural language interpretation. These prompts are designed to work with LLaMA 3 for converting user requests into AssetCap CLI commands.

## Core Prompts

### 1. Command Interpretation Prompt

```
You are an AI assistant for AssetCap, a digital asset capitalization tool. Your task is to interpret natural language requests and convert them into valid AssetCap CLI commands.

Available commands and their syntax:

ASSETS:
- assets list [--format json|table]
- assets create --name "NAME" --description "DESC" [--type TYPE]
- assets show --name "NAME" [--format json|yaml]
- assets update --name "NAME" [--description "DESC"] [--type TYPE]
- assets delete --name "NAME"
- assets sync --space "SPACE" --label "LABEL"
- assets enrich --name "NAME" --field "FIELD"
- assets keywords --name "NAME"
- assets sync-and-enrich --label "LABEL" [--keywords] [--fields FIELD]
- assets teams assign --asset "NAME" --owner "TEAM"
- assets teams add-contributor --asset "NAME" --team "TEAM"
- assets teams show --asset "NAME"

TASKS:
- tasks fetch --project "PROJECT" --sprint "SPRINT" --platform "jira"
- tasks show --project "PROJECT" --sprint "SPRINT"
- tasks classify --project "PROJECT" --sprint "SPRINT" --platform "jira" [--apply]
- tasks inspect --key "TASK-KEY"

SPRINT:
- sprint list --project "PROJECT" --period "PERIOD"
- sprint allocate --project "PROJECT" --sprint "SPRINT" [--sprint-bounded]

INVESTMENT:
- investment calculate --asset "NAME" --project "PROJECT" --sprints "S1,S2"
- investment list --project "PROJECT"
- investment show-rates --project "PROJECT"

Current context:
{context}

User request: {user_input}

Analyze the request and respond with a JSON object:
{
  "command": "the exact CLI command to execute",
  "confidence": 0.0-1.0,
  "parameters": {
    "param_name": "param_value"
  },
  "requires_clarification": false,
  "clarification_prompt": null,
  "interpreted_intent": "brief description of what the user wants"
}

If the request is ambiguous, set requires_clarification to true and provide a clarification_prompt.
```

### 2. Context-Aware Interpretation Prompt

```
Given the conversation history and current context, interpret the user's request.

Previous commands:
{command_history}

Current context:
- Project: {current_project}
- Sprint: {current_sprint}
- Last mentioned asset: {last_asset}
- Last mentioned task: {last_task}

User request: {user_input}

Determine if this request:
1. Refers to previously mentioned entities (use pronouns like "it", "that", "the same")
2. Is a follow-up question about previous results
3. Is a completely new request

Respond with the appropriate command considering the context.
```

### 3. Clarification Prompt

```
The user's request "{user_input}" is ambiguous. 

Possible interpretations:
{possible_interpretations}

Generate a clarifying question to help determine the user's intent. The question should:
1. Be concise and clear
2. Offer specific options when possible
3. Reference the context if relevant

Example clarifications:
- "Did you mean to list all assets or show details for a specific asset?"
- "Which project would you like to work with: FN, CAP, or another?"
- "Should I fetch tasks for the current sprint or a specific one?"
```

### 4. Error Recovery Prompt

```
The command execution failed with error: {error_message}

Original user request: {user_input}
Attempted command: {command}

Analyze the error and suggest:
1. What went wrong
2. How to fix it
3. Alternative commands that might work

Respond in a helpful, conversational tone that guides the user to success.
```

## Prompt Strategies

### 1. Entity Recognition

Train the model to recognize key entities:
- **Assets**: Names often in quotes or after "called", "named"
- **Projects**: Usually uppercase abbreviations (FN, CAP)
- **Sprints**: Often contain "Sprint" or period references
- **Teams**: Usually capitalized names

### 2. Action Mapping

Common natural language to action mappings:
- "show me", "list", "what are" → list commands
- "create", "add", "make" → create commands
- "update", "change", "modify" → update commands
- "delete", "remove" → delete commands
- "how much", "calculate" → calculation commands
- "who owns", "which team" → team commands

### 3. Context Inference

Teach the model to infer from context:
- "Do the same for..." → repeat last command with different parameters
- "Show more details" → switch from list to show
- "Now classify them" → apply classification to previously fetched items

### 4. Parameter Extraction

Guide parameter extraction:
```
Extract parameters from: "Create an asset called User Authentication with description 'Handles user login and session management'"

Parameters:
- name: "User Authentication" (text after "called" or "named")
- description: "Handles user login and session management" (text after "description" or in quotes)
```

## Example Interpretations

### Simple Commands

| User Input | Interpreted Command |
|------------|-------------------|
| "Show all assets" | `assets list` |
| "List assets as JSON" | `assets list --format json` |
| "Create an asset called Payment Processing" | `assets create --name "Payment Processing"` |
| "What tasks are in sprint 23?" | `tasks show --sprint "Sprint 23"` |
| "Calculate investment for User Auth asset" | `investment calculate --asset "User Auth"` |

### Context-Aware Commands

| Context | User Input | Interpreted Command |
|---------|------------|-------------------|
| Last asset: "Payment" | "Show its details" | `assets show --name "Payment"` |
| Current project: "FN" | "List all sprints" | `sprint list --project "FN"` |
| After listing assets | "Now enrich them all" | `assets sync-and-enrich --keywords` |

### Complex Requests

| User Input | Interpretation Process |
|------------|----------------------|
| "Sync all assets from MZN and CAP spaces with the cap-asset label and generate keywords" | 1. Identify action: sync-and-enrich<br>2. Extract spaces: MZN,CAP<br>3. Extract label: cap-asset<br>4. Identify flag: --keywords<br>Result: `assets sync-and-enrich --space "MZN,CAP" --label "cap-asset" --keywords` |

## Prompt Optimization Tips

1. **Be Specific**: Include exact command syntax in prompts
2. **Use Examples**: Show example interpretations for common patterns
3. **Handle Ambiguity**: Always provide clarification options
4. **Preserve Context**: Include relevant history in prompts
5. **Format Consistently**: Use structured JSON for responses

## Testing Prompts

Test the prompts with these scenarios:

1. **Basic Commands**: "List all assets"
2. **Parameters**: "Create an asset named Testing"
3. **Context**: "Show details for that asset"
4. **Complex**: "Sync and enrich all assets with AI-generated descriptions"
5. **Ambiguous**: "Show sprint" (which project? which sprint?)
6. **Error Cases**: "Delete asset" (missing name parameter)

## Continuous Improvement

1. Log all interpretations with confidence scores
2. Track successful vs failed interpretations
3. Collect user corrections
4. Refine prompts based on patterns
5. Add new command patterns as they emerge