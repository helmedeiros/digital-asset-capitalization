package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain/ports"
)

// Interpreter implements the AI interpreter using LLaMA
type Interpreter struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

// Config holds configuration for the AI interpreter
type Config struct {
	BaseURL string
	Model   string
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		BaseURL: "http://localhost:11434",
		Model:   "llama3",
	}
}

// NewInterpreter creates a new AI interpreter
func NewInterpreter(config Config) *Interpreter {
	return &Interpreter{
		baseURL:    config.BaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		model:      config.Model,
	}
}

// Interpret converts natural language input to a structured command
func (i *Interpreter) Interpret(ctx context.Context, input string, sessionContext *domain.Context) (*domain.Command, error) {
	prompt := i.buildInterpretationPrompt(input, sessionContext)

	response, err := i.callLLaMA(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to interpret command: %w", err)
	}

	// Parse the response
	var result InterpretationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If JSON parsing fails, try to extract JSON from within the response
		if extractedJSON := i.extractJSONFromResponse(response); extractedJSON != "" {
			if jsonErr := json.Unmarshal([]byte(extractedJSON), &result); jsonErr != nil {
				// Fall back to text parsing if extracted JSON is also invalid
				result = i.parseTextResponse(response)
			}
		} else {
			// Fall back to text parsing
			result = i.parseTextResponse(response)
		}
	}

	// Create command from result
	command, err := domain.NewCommand(
		sessionContext.SessionID,
		input,
		result.Command,
		result.Confidence,
	)
	if err != nil {
		return nil, err
	}

	// Set intent if available
	if result.InterpretedIntent != "" {
		i.setCommandIntent(command, result)
	}

	// Add parameters
	for k, v := range result.Parameters {
		command.AddParameter(k, v)
	}

	// Check if clarification is needed
	if result.RequiresClarification && result.ClarificationPrompt != "" {
		return nil, ports.NewInterpretationError(input, result.ClarificationPrompt, []string{result.Command})
	}

	return command, nil
}

// GetClarification generates a clarifying question when input is ambiguous
func (i *Interpreter) GetClarification(ctx context.Context, ambiguity string, options []string) (string, error) {
	prompt := fmt.Sprintf(`Generate a helpful clarification question for the user.

Ambiguity: %s
Possible options: %s

Generate a concise, friendly question to help clarify what the user wants. 
The question should reference the specific options available.
Keep it to one sentence.

Example: "Did you mean to list all assets or show details for a specific asset?"`,
		ambiguity, strings.Join(options, ", "))

	response, err := i.callLLaMA(ctx, prompt)
	if err != nil {
		return "Could you please clarify what you meant?", nil
	}

	return strings.TrimSpace(response), nil
}

// AnalyzeIntent performs deeper analysis of user intent
func (i *Interpreter) AnalyzeIntent(ctx context.Context, input string) (*domain.CommandIntent, error) {
	prompt := fmt.Sprintf(`Analyze the user's intent from this input: "%s"

Determine:
1. Action type (create, read, update, delete, list, sync, help, other)
2. Resource type (asset, task, sprint, investment, config, context, unknown)
3. Target (specific entity name if mentioned)

Respond with JSON:
{
  "action": "action_type",
  "resource": "resource_type",
  "target": "specific_target_or_empty"
}`, input)

	response, err := i.callLLaMA(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var result struct {
		Action   string `json:"action"`
		Resource string `json:"resource"`
		Target   string `json:"target"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse intent: %w", err)
	}

	intent := &domain.CommandIntent{
		Action:   i.mapActionType(result.Action),
		Resource: i.mapResourceType(result.Resource),
		Target:   result.Target,
	}

	return intent, nil
}

// buildInterpretationPrompt creates the prompt for command interpretation
func (i *Interpreter) buildInterpretationPrompt(input string, context *domain.Context) string {
	contextInfo := i.buildContextInfo(context)

	return fmt.Sprintf(`You are an AI assistant for AssetCap, a digital asset capitalization tool. Your task is to interpret natural language requests and convert them into valid AssetCap CLI commands.

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
- assets teams assign --asset "NAME" --team "TEAM" (assign team ownership)
- assets teams add-contributor --asset "NAME" --team "TEAM" (add contributing team)
- assets teams remove-contributor --asset "NAME" --team "TEAM" (remove contributing team)
- assets teams show --asset "NAME" (show team assignments for asset)
- assets teams list (list all team assignments)

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

CONFIG:
- config init (initialize configuration)
- config show (show current configuration)  
- config validate (validate configuration)
- config sync-team --project "PROJECT" (sync team members from JIRA)

Current context:
%s

User request: %s

CRITICAL: Team query disambiguation:
1. TEAM MEMBERS (people working on projects) → use CONFIG commands
2. TEAM ASSIGNMENTS (which teams own assets) → use ASSETS TEAMS commands

REQUIRED mappings:
- "team members for project X" → config sync-team --project "X" 
- "team members for X project" → config sync-team --project "X"
- "show team members for X" → config sync-team --project "X"
- "all team members for X" → config sync-team --project "X"
- "list team assignments" → assets teams list
- "show teams" (without project) → assets teams list
- "who owns asset X" → assets teams show --asset "X"

Examples:
- "show me team members for AD" = config sync-team --project "AD"
- "team members for FN project" = config sync-team --project "FN"
- "show teams" = assets teams list

Analyze the request and respond with ONLY a valid JSON object (no explanations or extra text):
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

Important: Return ONLY the JSON object above, no markdown code blocks, no explanations.
If the request is ambiguous, set requires_clarification to true and provide a clarification_prompt.`, contextInfo, input)
}

// buildContextInfo builds context information for the prompt
func (i *Interpreter) buildContextInfo(context *domain.Context) string {
	var info []string

	if context.CurrentProject != "" {
		info = append(info, fmt.Sprintf("- Current project: %s", context.CurrentProject))
	}
	if context.CurrentSprint != "" {
		info = append(info, fmt.Sprintf("- Current sprint: %s", context.CurrentSprint))
	}
	if context.CurrentSpace != "" {
		info = append(info, fmt.Sprintf("- Current space: %s", context.CurrentSpace))
	}
	if len(context.RecentAssets) > 0 {
		info = append(info, fmt.Sprintf("- Recent assets: %v", context.RecentAssets))
	}
	if len(context.RecentTasks) > 0 {
		info = append(info, fmt.Sprintf("- Recent tasks: %v", context.RecentTasks))
	}

	if len(info) == 0 {
		return "No context available"
	}

	return strings.Join(info, "\n")
}

// callLLaMA makes a request to the LLaMA API
func (i *Interpreter) callLLaMA(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model":  i.model,
		"prompt": prompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", i.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return strings.TrimSpace(result.Response), nil
}

// extractJSONFromResponse tries to extract JSON from a text response
func (i *Interpreter) extractJSONFromResponse(text string) string {
	// Look for JSON between ``` markers
	lines := strings.Split(text, "\n")
	var jsonLines []string
	inJSON := false
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for code block markers
		if strings.HasPrefix(trimmed, "```json") {
			inCodeBlock = true
			inJSON = true
			continue
		} else if strings.HasPrefix(trimmed, "```") && inCodeBlock {
			break
		}

		// Look for direct JSON patterns
		if !inJSON && (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) {
			inJSON = true
		}

		if inJSON {
			jsonLines = append(jsonLines, line)
			// Stop at closing brace if not in code block
			if !inCodeBlock && (strings.HasSuffix(trimmed, "}") || strings.HasSuffix(trimmed, "},")) {
				break
			}
		}
	}

	if len(jsonLines) > 0 {
		return strings.Join(jsonLines, "\n")
	}

	return ""
}

// parseTextResponse attempts to parse a plain text response
func (i *Interpreter) parseTextResponse(text string) InterpretationResult {
	// Simple heuristic parsing for non-JSON responses
	result := InterpretationResult{
		Command:    strings.TrimSpace(text),
		Confidence: 0.5, // Lower confidence for text parsing
		Parameters: make(map[string]interface{}),
	}

	// Try to extract command from common patterns
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "assets ") || strings.HasPrefix(line, "tasks ") ||
			strings.HasPrefix(line, "sprint ") || strings.HasPrefix(line, "investment ") {
			result.Command = line
			result.Confidence = 0.7
			break
		}
	}

	return result
}

// setCommandIntent sets the command intent based on interpretation result
func (i *Interpreter) setCommandIntent(cmd *domain.Command, result InterpretationResult) {
	// Parse command to determine action and resource
	parts := strings.Fields(result.Command)
	if len(parts) < 2 {
		return
	}

	resource := i.mapResourceType(parts[0])
	action := domain.CommandTypeOther

	if len(parts) > 1 {
		switch parts[1] {
		case "list":
			action = domain.CommandTypeList
		case "create":
			action = domain.CommandTypeCreate
		case "show", "inspect":
			action = domain.CommandTypeRead
		case "update":
			action = domain.CommandTypeUpdate
		case "delete":
			action = domain.CommandTypeDelete
		case "sync", "sync-and-enrich":
			action = domain.CommandTypeSync
		}
	}

	target := ""
	if name, ok := result.Parameters["name"].(string); ok {
		target = name
	}

	cmd.SetIntent(action, resource, target)
}

// mapActionType maps string to CommandType
func (i *Interpreter) mapActionType(action string) domain.CommandType {
	switch strings.ToLower(action) {
	case "create":
		return domain.CommandTypeCreate
	case "read", "show", "inspect":
		return domain.CommandTypeRead
	case "update":
		return domain.CommandTypeUpdate
	case "delete":
		return domain.CommandTypeDelete
	case "list":
		return domain.CommandTypeList
	case "sync":
		return domain.CommandTypeSync
	case "help":
		return domain.CommandTypeHelp
	default:
		return domain.CommandTypeOther
	}
}

// mapResourceType maps string to ResourceType
func (i *Interpreter) mapResourceType(resource string) domain.ResourceType {
	switch strings.ToLower(resource) {
	case "asset", "assets":
		return domain.ResourceTypeAsset
	case "task", "tasks":
		return domain.ResourceTypeTask
	case "sprint", "sprints":
		return domain.ResourceTypeSprint
	case "investment", "investments":
		return domain.ResourceTypeInvestment
	case "config", "configuration":
		return domain.ResourceTypeConfig
	case "context":
		return domain.ResourceTypeContext
	default:
		return domain.ResourceTypeUnknown
	}
}

// InterpretationResult represents the result of command interpretation
type InterpretationResult struct {
	Command               string                 `json:"command"`
	Confidence            float64                `json:"confidence"`
	Parameters            map[string]interface{} `json:"parameters"`
	RequiresClarification bool                   `json:"requires_clarification"`
	ClarificationPrompt   string                 `json:"clarification_prompt"`
	InterpretedIntent     string                 `json:"interpreted_intent"`
}
