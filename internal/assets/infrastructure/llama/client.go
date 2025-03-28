package llama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Client represents an Ollama API client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Config holds the configuration for the Ollama client
type Config struct {
	BaseURL string
}

// DefaultConfig returns a default configuration for the Ollama client
func DefaultConfig() Config {
	baseURL := os.Getenv("OLLAMA_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return Config{
		BaseURL: baseURL,
	}
}

// NewClient creates a new Ollama client
func NewClient(config Config) (*Client, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("OLLAMA_API_URL environment variable must be set")
	}

	return &Client{
		baseURL:    config.BaseURL,
		httpClient: &http.Client{},
	}, nil
}

// EnrichContent sends content to Ollama for enrichment
func (c *Client) EnrichContent(content string, field string) (string, error) {
	prompt := fmt.Sprintf(`You are enriching a field of a capitalisable software asset based on internal documentation from Confluence.

Your task is to carefully analyze the page content and generate a new, clean version of the following field:
**%s**

Guidelines:
- Be strictly factual and neutral - avoid any marketing language or subjective claims
- Focus on concrete features, functionality, and technical aspects
- Remove any promotional language, buzzwords, or emotional appeals
- Keep it concise and audit-friendly
- Do NOT include any introduction or explanation.
- Do NOT format with Markdown or headings.
- Output should be a single, plain-text paragraph.
- The output must be suitable for inclusion in a JSON field.
- Do not include phrases like "we aim to", "we want to", "we hope to", etc.
- Do not include subjective benefits or user experience claims
- Stick to what the feature actually does, not what it's intended to achieve
- Output only the content of the new field, as a single plain-text paragraph.
- Do NOT include any headings, titles, or markdown formatting.
- Do NOT explain what you're doing.
- Do NOT prefix with phrases like "Here's the updated version..."
- Output only the new content of the field.

Here is the content of the page (extracted from Confluence):

---
%s
---

Now write a new value for the field: **%s**`, field, content, field)

	requestBody := map[string]interface{}{
		"model":  "llama2",
		"prompt": prompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Response == "" {
		return "", fmt.Errorf("no response from Ollama")
	}

	return result.Response, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	// No resources to clean up since we're using the default http.Client
	return nil
}
