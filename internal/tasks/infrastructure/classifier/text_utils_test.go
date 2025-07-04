package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextMatcher_MatchesAssetName(t *testing.T) {
	tm := NewTextMatcher()

	tests := []struct {
		name      string
		content   string
		assetName string
		expected  bool
	}{
		// Exact matches
		{"exact match", "Fix Omio Flex issues", "Omio Flex", true},
		{"exact match case insensitive", "fix omio flex issues", "Omio Flex", true},

		// camelCase variations
		{"camelCase concatenated", "Fix OmioFlex component", "Omio Flex", true},
		{"camelCase in middle", "Update OmioFlexAddedMessage component", "Omio Flex", true},
		{"camelCase lowercase", "fix omioflex issues", "Omio Flex", true},

		// snake_case variations
		{"snake_case", "Fix omio_flex component", "Omio Flex", true},
		{"snake_case in filename", "update omio_flex_config.json", "Omio Flex", true},

		// kebab-case variations
		{"kebab-case", "Fix omio-flex component", "Omio Flex", true},
		{"kebab-case in URL", "api/omio-flex/status", "Omio Flex", true},

		// Multi-word assets
		{"dynamic markup camelCase", "Fix DynamicMarkup calculation", "Dynamic Markup", true},
		{"service fee snake_case", "Update service_fee logic", "Service Fee", true},
		{"price calendar kebab-case", "Implement price-calendar widget", "Price Calendar", true},
		{"virtual credit card PascalCase", "VirtualCreditCard integration", "Virtual Credit Card", true},

		// Negative cases
		{"partial word", "Fix flex issues", "Omio Flex", false},
		{"different asset", "Fix Payment Gateway", "Omio Flex", false},
		{"empty content", "", "Omio Flex", false},
		{"empty asset name", "Fix issues", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.MatchesAssetName(tt.content, tt.assetName)
			assert.Equal(t, tt.expected, result, "Content: %s, Asset: %s", tt.content, tt.assetName)
		})
	}
}

func TestTextMatcher_MatchesKeyword(t *testing.T) {
	tm := NewTextMatcher()

	tests := []struct {
		name     string
		content  string
		keyword  string
		expected bool
	}{
		// Single word keywords
		{"exact keyword", "Fix flex issues", "flex", true},
		{"keyword in camelCase", "Fix FlexComponent", "flex", true},
		{"keyword case insensitive", "Fix FLEX issues", "flex", true},

		// Multi-word keywords
		{"multi-word exact", "fare flexibility options", "fare flexibility", true},
		{"multi-word camelCase", "FareFlexibility settings", "fare flexibility", true},
		{"multi-word snake_case", "fare_flexibility_config", "fare flexibility", true},
		{"multi-word kebab-case", "fare-flexibility-widget", "fare flexibility", true},

		// Negative cases
		{"partial keyword", "fl issues", "flex", false},
		{"different keyword", "payment gateway", "flex", false},
		{"empty content", "", "flex", false},
		{"empty keyword", "Fix issues", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.MatchesKeyword(tt.content, tt.keyword)
			assert.Equal(t, tt.expected, result, "Content: %s, Keyword: %s", tt.content, tt.keyword)
		})
	}
}

func TestTextMatcher_generateNameVariations(t *testing.T) {
	tm := NewTextMatcher()

	tests := []struct {
		name        string
		input       string
		expectedLen int
		mustContain []string
	}{
		{
			name:        "two words",
			input:       "omio flex",
			expectedLen: 7, // Original + 6 variations
			mustContain: []string{"omio flex", "omioFlex", "OmioFlex", "omio_flex", "omio-flex", "omioflex", "OMIOFLEX"},
		},
		{
			name:        "three words",
			input:       "virtual credit card",
			expectedLen: 7,
			mustContain: []string{"virtual credit card", "virtualCreditCard", "VirtualCreditCard", "virtual_credit_card", "virtual-credit-card", "virtualcreditcard", "VIRTUALCREDITCARD"},
		},
		{
			name:        "single word",
			input:       "flex",
			expectedLen: 1, // Only original for single words
			mustContain: []string{"flex"},
		},
		{
			name:        "already camelCase",
			input:       "omioFlex",
			expectedLen: 7,
			mustContain: []string{"omio flex", "omioFlex", "OmioFlex", "omio_flex", "omio-flex", "omioflex", "OMIOFLEX"},
		},
		{
			name:        "already snake_case",
			input:       "omio_flex",
			expectedLen: 7,
			mustContain: []string{"omio flex", "omioFlex", "OmioFlex", "omio_flex", "omio-flex", "omioflex", "OMIOFLEX"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variations := tm.generateNameVariations(tt.input)

			assert.GreaterOrEqual(t, len(variations), tt.expectedLen, "Should generate at least %d variations", tt.expectedLen)

			for _, expected := range tt.mustContain {
				assert.Contains(t, variations, expected, "Should contain variation: %s", expected)
			}

			// Check for duplicates
			seen := make(map[string]bool)
			for _, v := range variations {
				assert.False(t, seen[v], "Should not have duplicate: %s", v)
				seen[v] = true
			}
		})
	}
}

func TestTextMatcher_splitIntoWords(t *testing.T) {
	tm := NewTextMatcher()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"space separated", "omio flex", []string{"omio", "flex"}},
		{"camelCase", "omioFlex", []string{"omio", "flex"}},
		{"PascalCase", "OmioFlex", []string{"omio", "flex"}},
		{"snake_case", "omio_flex", []string{"omio", "flex"}},
		{"kebab-case", "omio-flex", []string{"omio", "flex"}},
		{"mixed separators", "omio-flex_service", []string{"omio", "flex", "service"}},
		{"single word", "flex", []string{"flex"}},
		{"empty string", "", []string{}},
		{"complex camelCase", "OmioFlexAddedMessage", []string{"omio", "flex", "added", "message"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.splitIntoWords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTextMatcher_CaseConversions(t *testing.T) {
	tm := NewTextMatcher()
	words := []string{"omio", "flex", "service"}

	tests := []struct {
		name     string
		function func([]string) string
		expected string
	}{
		{"camelCase", tm.toCamelCase, "omioFlexService"},
		{"PascalCase", tm.toPascalCase, "OmioFlexService"},
		{"snake_case", tm.toSnakeCase, "omio_flex_service"},
		{"kebab-case", tm.toKebabCase, "omio-flex-service"},
		{"concatenated", tm.toConcatenated, "omioflexservice"},
		{"upper concatenated", tm.toUpperConcatenated, "OMIOFLEXSERVICE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(words)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTextMatcher_RealWorldScenarios(t *testing.T) {
	tm := NewTextMatcher()

	tests := []struct {
		name        string
		content     string
		assetName   string
		shouldMatch bool
		description string
	}{
		{
			name:        "FN-972 scenario",
			content:     "Fix color styles to pass contrast checks. Update OmioFlexAddedMessage and OmioFlexPriceTextStyle components",
			assetName:   "Omio Flex",
			shouldMatch: true,
			description: "Should match OmioFlex in component names",
		},
		{
			name:        "payment gateway service",
			content:     "Refactor PaymentGatewayService timeout handling",
			assetName:   "Payment Gateway",
			shouldMatch: true,
			description: "Should match PaymentGateway in class names",
		},
		{
			name:        "dynamic markup config",
			content:     "Update dynamic_markup_config.json file",
			assetName:   "Dynamic Markup",
			shouldMatch: true,
			description: "Should match dynamic_markup in file names",
		},
		{
			name:        "price lock widget",
			content:     "Implement price-lock-widget for booking flow",
			assetName:   "Price Lock",
			shouldMatch: true,
			description: "Should match price-lock in feature names",
		},
		{
			name:        "service fee calculation",
			content:     "Fix service_fee_calculation logic",
			assetName:   "Service Fee",
			shouldMatch: true,
			description: "Should match service_fee in variable names",
		},
		{
			name:        "virtual credit card test",
			content:     "VirtualCreditCard integration test",
			assetName:   "Virtual Credit Card",
			shouldMatch: true,
			description: "Should match VirtualCreditCard in test names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.MatchesAssetName(tt.content, tt.assetName)
			assert.Equal(t, tt.shouldMatch, result, tt.description)
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		assetName   string
		shouldMatch bool
	}{
		{"convenience asset name match", "Fix OmioFlex issues", "Omio Flex", true},
		{"convenience keyword match", "flexible ticketing", "flexible ticketing", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchesAssetNameEnhanced(tt.content, tt.assetName)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}
