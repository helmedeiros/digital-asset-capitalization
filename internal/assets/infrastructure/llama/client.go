package llama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// DefaultTimeout is the default HTTP timeout for Ollama requests. It is
// generous because LLM inference for large prompts can legitimately take
// several minutes; a shorter value risks aborting valid in-flight requests.
const DefaultTimeout = 5 * time.Minute

// Client represents an Ollama API client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Config holds the configuration for the Ollama client
type Config struct {
	BaseURL string
	Timeout time.Duration
}

// DefaultConfig returns a default configuration for the Ollama client
func DefaultConfig() Config {
	baseURL := os.Getenv("OLLAMA_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return Config{
		BaseURL: baseURL,
		Timeout: DefaultTimeout,
	}
}

// NewClient creates a new Ollama client
func NewClient(config Config) (*Client, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("OLLAMA_API_URL environment variable must be set")
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	return &Client{
		baseURL:    config.BaseURL,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

var (
	htmlTagRE    = regexp.MustCompile("<[^>]*>")
	whitespaceRE = regexp.MustCompile(`\s+`)
)

// cleanHTML removes HTML tags and normalizes whitespace
func cleanHTML(content string) string {
	content = htmlTagRE.ReplaceAllString(content, "")

	content = strings.ReplaceAll(content, "&nbsp;", " ")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&#39;", "'")

	content = whitespaceRE.ReplaceAllString(content, " ")
	return strings.TrimSpace(content)
}

// EnrichContent sends content to Ollama for enrichment
func (c *Client) EnrichContent(content string, field string, asset *domain.Asset) (string, error) {
	cleanedContent := cleanHTML(content)

	log.Printf("Enriching content for field: %s", field)
	log.Printf("Cleaned content: %s", cleanedContent)
	log.Printf("Asset Why: %s", asset.Why)
	log.Printf("Asset How: %s", asset.How)
	log.Printf("Asset Benefits: %s", asset.Benefits)
	log.Printf("Asset Metrics: %s", asset.Metrics)

	prompt := fmt.Sprintf(`You are a professional technical writer helping to enrich a specific field of a software asset based on internal documentation from Confluence.

The asset is about: %s

Current asset fields:
Why: %s
Benefits: %s
How: %s
Metrics: %s

Content from Confluence:
%s

Please generate a clean version of the field "%s" based on the above information.

Guidelines:
1. Generate a single, concise paragraph (maximum 2 sentences) that describes what the asset does
2. Focus only on the core functionality and purpose
3. Use professional, technical language without marketing terms
4. Do not include any formatting, headers, sections, or line breaks
5. Do not include any placeholders or template language
6. Do not mention that you are an AI or that this is a generated response
7. Do not include any metadata or additional information
8. Do not include any subjective benefits or user experience claims
9. Do not include phrases like "we aim to", "we want to", "we hope to", etc.
10. Do not include any bullet points, lists, or sections
11. Do not include any marketing language or promotional content
12. Do not include any future plans or aspirations
13. Do not include any technical implementation details
14. Do not include any metrics or success criteria
15. Do not include any information about the company, team, or organization
16. Do not include any references to user experience or benefits
17. Return only the field content as a single paragraph, nothing else

Field content:`, asset.Name, asset.Why, asset.Benefits, asset.How, asset.Metrics, cleanedContent, field)

	// Add debug logging
	fmt.Printf("\n=== Debug: Content being sent to LLaMA ===\n")
	fmt.Printf("Field: %s\n", field)
	fmt.Printf("Why: %s\n", asset.Why)
	fmt.Printf("How: %s\n", asset.How)
	fmt.Printf("Benefits: %s\n", asset.Benefits)
	fmt.Printf("Metrics: %s\n", asset.Metrics)
	fmt.Printf("Content from Confluence (cleaned):\n%s\n", cleanedContent)
	fmt.Printf("=====================================\n\n")

	requestBody := map[string]interface{}{
		"model":  "llama4",
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

// GenerateEmbeddings generates embedding vectors for the given texts using Ollama's /api/embed endpoint
func (c *Client) GenerateEmbeddings(model string, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	requestBody := map[string]interface{}{
		"model": model,
		"input": texts,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/embed", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama embed returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Embeddings [][]float64 `json:"embeddings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	if len(result.Embeddings) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(result.Embeddings))
	}

	return result.Embeddings, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	// No resources to clean up since we're using the default http.Client
	return nil
}
