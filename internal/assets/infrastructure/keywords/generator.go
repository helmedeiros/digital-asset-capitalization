package keywords

import (
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// LlamaClient defines the interface for LLaMA operations
type LlamaClient interface {
	EnrichContent(content, field string, asset *domain.Asset) (string, error)
	Close() error
}

// Generator handles keyword generation for assets
type Generator struct {
	llamaClient LlamaClient
}

// NewGenerator creates a new keyword generator
func NewGenerator(llamaClient LlamaClient) *Generator {
	return &Generator{
		llamaClient: llamaClient,
	}
}

// GenerateKeywords generates keywords for an asset based on its content
func (g *Generator) GenerateKeywords(asset *domain.Asset) ([]string, error) {
	// Combine relevant asset fields for keyword generation
	content := fmt.Sprintf(`Asset Name: %s
Description: %s
Why: %s
Benefits: %s
How: %s
Metrics: %s`,
		asset.Name,
		asset.Description,
		asset.Why,
		asset.Benefits,
		asset.How,
		asset.Metrics,
	)

	// Create a prompt for keyword generation
	prompt := fmt.Sprintf(`You are a professional technical writer helping to generate keywords for a software asset.

Asset Content:
%s

Please generate a list of relevant keywords for this asset. Guidelines:
1. Generate 5-10 keywords that best represent the asset's purpose and functionality
2. Use technical terms and domain-specific vocabulary
3. Include both broad and specific terms
4. Avoid generic terms like "software", "system", "application"
5. Use single words or short phrases (2-3 words max)
6. Separate keywords with commas
7. Do not include any explanations or additional text
8. Do not include any formatting or special characters
9. Do not include any metadata or labels
10. Do not include any marketing terms or buzzwords

Keywords:`, content)

	// Get response from LLaMA
	response, err := g.llamaClient.EnrichContent(prompt, "keywords", asset)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keywords: %w", err)
	}

	// Process the response to extract keywords
	keywords := processKeywords(response)
	return keywords, nil
}

// processKeywords processes the LLaMA response to extract keywords
func processKeywords(response string) []string {
	// Split by commas and clean up each keyword
	rawKeywords := strings.Split(response, ",")
	keywords := make([]string, 0, len(rawKeywords))

	for _, keyword := range rawKeywords {
		// Clean up the keyword
		keyword = strings.TrimSpace(keyword)
		keyword = strings.ToLower(keyword)

		// Skip empty keywords
		if keyword == "" {
			continue
		}

		// Remove any remaining special characters
		keyword = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' {
				return r
			}
			return -1
		}, keyword)

		// Skip if keyword contains only special characters
		if strings.TrimSpace(keyword) == "" {
			continue
		}

		// Skip if keyword is too short or too long
		// We'll count hyphens as word separators, so we need to check the actual word length
		words := strings.Split(keyword, "-")
		isValid := true
		for _, word := range words {
			if len(word) < 2 {
				isValid = false
				break
			}
		}
		if !isValid || len(keyword) > 25 {
			continue
		}

		keywords = append(keywords, keyword)
	}

	return keywords
}
