package classifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	assetports "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// LLMAssetClassifier implements asset classification using an LLM (Ollama) for semantic reasoning
type LLMAssetClassifier struct {
	baseURL   string
	model     string
	assetRepo assetports.AssetRepository
	client    *http.Client
}

// NewLLMAssetClassifier creates a new LLM-based asset classifier
func NewLLMAssetClassifier(baseURL, model string, assetRepo assetports.AssetRepository) ports.AssetClassifier {
	return &LLMAssetClassifier{
		baseURL:   baseURL,
		model:     model,
		assetRepo: assetRepo,
		client:    &http.Client{},
	}
}

// ollamaRequest represents the request payload for Ollama's /api/generate endpoint
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaResponse represents the response from Ollama's /api/generate endpoint
type ollamaResponse struct {
	Response string `json:"response"`
}

// llmClassificationResponse represents the structured JSON response expected from the LLM
type llmClassificationResponse struct {
	AssetName  string  `json:"asset_name"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// ClassifyTaskAsset determines which asset a task belongs to using LLM reasoning
func (c *LLMAssetClassifier) ClassifyTaskAsset(task *taskdomain.Task) (*ports.AssetClassificationResult, error) {
	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}

	assets, err := c.assetRepo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assets: %w", err)
	}

	if len(assets) == 0 {
		return &ports.AssetClassificationResult{
			Task:       task,
			Asset:      nil,
			Confidence: 0,
			Reason:     "no assets available for LLM classification",
		}, nil
	}

	prompt := c.buildPrompt(task, assets)

	response, err := c.callOllama(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM classification failed: %w", err)
	}

	return c.parseResponse(response, task, assets), nil
}

// ClassifyTasksAssets classifies multiple tasks using the LLM
func (c *LLMAssetClassifier) ClassifyTasksAssets(tasks []*taskdomain.Task) ([]*ports.AssetClassificationResult, error) {
	results := make([]*ports.AssetClassificationResult, 0, len(tasks))
	for _, task := range tasks {
		result, err := c.ClassifyTaskAsset(task)
		if err != nil {
			return nil, fmt.Errorf("failed to classify task %s: %w", task.Key, err)
		}
		results = append(results, result)
	}
	return results, nil
}

// buildPrompt constructs the LLM prompt with task and asset context
func (c *LLMAssetClassifier) buildPrompt(task *taskdomain.Task, assets []*assetdomain.Asset) string {
	var sb strings.Builder

	sb.WriteString("You are a software asset classifier. Given a JIRA task and a list of digital assets, determine which asset the task belongs to.\n\n")

	// Task context
	sb.WriteString("## Task\n")
	sb.WriteString(fmt.Sprintf("- Key: %s\n", task.Key))
	sb.WriteString(fmt.Sprintf("- Summary: %s\n", task.Summary))
	if task.Description != "" {
		desc := task.Description
		if len(desc) > 500 {
			desc = desc[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("- Description: %s\n", desc))
	}
	sb.WriteString(fmt.Sprintf("- Type: %s\n", task.Type))
	if task.Epic != "" {
		sb.WriteString(fmt.Sprintf("- Epic: %s\n", task.Epic))
	}
	if len(task.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("- Labels: %s\n", strings.Join(task.Labels, ", ")))
	}

	// Assets context
	sb.WriteString("\n## Available Assets\n")
	for i, asset := range assets {
		sb.WriteString(fmt.Sprintf("\n### Asset %d: %s\n", i+1, asset.Name))
		if asset.Description != "" {
			desc := asset.Description
			if len(desc) > 300 {
				desc = desc[:300] + "..."
			}
			sb.WriteString(fmt.Sprintf("- Description: %s\n", desc))
		}
		if len(asset.Keywords) > 0 {
			sb.WriteString(fmt.Sprintf("- Keywords: %s\n", strings.Join(asset.Keywords, ", ")))
		}
	}

	// Response format
	sb.WriteString("\n## Instructions\n")
	sb.WriteString("Analyze the task and determine which asset it most likely belongs to. Consider the task's summary, description, type, epic, and labels against each asset's name, description, and keywords.\n\n")
	sb.WriteString("Respond with ONLY a JSON object in this exact format (no additional text):\n")
	sb.WriteString(`{"asset_name": "Exact Asset Name", "confidence": 0.85, "reasoning": "Brief explanation"}`)
	sb.WriteString("\n\nIf no asset matches, use confidence 0.0 and asset_name \"none\".\n")
	sb.WriteString("Confidence should be between 0.0 and 1.0.\n")

	return sb.String()
}

// callOllama sends the prompt to the Ollama API and returns the response text
func (c *LLMAssetClassifier) callOllama(prompt string) (string, error) {
	reqBody := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return ollamaResp.Response, nil
}

// parseResponse extracts the classification from the LLM response text
func (c *LLMAssetClassifier) parseResponse(response string, task *taskdomain.Task, assets []*assetdomain.Asset) *ports.AssetClassificationResult {
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		log.Printf("LLM returned no valid JSON for task %s", task.Key)
		return &ports.AssetClassificationResult{
			Task:       task,
			Asset:      nil,
			Confidence: 0,
			Reason:     "LLM response contained no valid JSON",
		}
	}

	var llmResp llmClassificationResponse
	if err := json.Unmarshal([]byte(jsonStr), &llmResp); err != nil {
		log.Printf("Failed to parse LLM JSON for task %s: %v", task.Key, err)
		return &ports.AssetClassificationResult{
			Task:       task,
			Asset:      nil,
			Confidence: 0,
			Reason:     "failed to parse LLM response JSON",
		}
	}

	// Clamp confidence
	if llmResp.Confidence < 0 {
		llmResp.Confidence = 0
	}
	if llmResp.Confidence > 1.0 {
		llmResp.Confidence = 1.0
	}

	// Match asset by name (fuzzy)
	if strings.EqualFold(llmResp.AssetName, "none") || llmResp.AssetName == "" {
		return &ports.AssetClassificationResult{
			Task:       task,
			Asset:      nil,
			Confidence: 0,
			Reason:     fmt.Sprintf("LLM: no matching asset (%s)", llmResp.Reasoning),
		}
	}

	matchedAsset := fuzzyMatchAsset(llmResp.AssetName, assets)
	if matchedAsset == nil {
		return &ports.AssetClassificationResult{
			Task:       task,
			Asset:      nil,
			Confidence: 0,
			Reason:     fmt.Sprintf("LLM suggested '%s' but no matching asset found", llmResp.AssetName),
		}
	}

	return &ports.AssetClassificationResult{
		Task:       task,
		Asset:      matchedAsset,
		Confidence: llmResp.Confidence,
		Reason:     fmt.Sprintf("LLM: %s", llmResp.Reasoning),
	}
}

// extractJSON finds and returns the first JSON object in the text
func extractJSON(text string) string {
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}

	return ""
}

// fuzzyMatchAsset finds the best matching asset by name (case-insensitive, contains)
func fuzzyMatchAsset(name string, assets []*assetdomain.Asset) *assetdomain.Asset {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	// Exact match (case-insensitive)
	for _, asset := range assets {
		if strings.EqualFold(asset.Name, nameLower) {
			return asset
		}
	}

	// Contains match
	for _, asset := range assets {
		assetLower := strings.ToLower(asset.Name)
		if strings.Contains(assetLower, nameLower) || strings.Contains(nameLower, assetLower) {
			return asset
		}
	}

	return nil
}
