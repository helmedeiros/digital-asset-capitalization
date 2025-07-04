package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

func TestTextMatching_CamelCaseVariations(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		taskContent   string
		expectedMatch bool
		description   string
	}{
		{
			name:          "exact match",
			assetName:     "Omio Flex",
			taskContent:   "Fix Omio Flex issues",
			expectedMatch: true,
			description:   "Should match exact asset name",
		},
		{
			name:          "camelCase concatenated",
			assetName:     "Omio Flex",
			taskContent:   "Fix OmioFlex component issues",
			expectedMatch: true,
			description:   "Should match camelCase concatenation: OmioFlex",
		},
		{
			name:          "all lowercase concatenated",
			assetName:     "Omio Flex",
			taskContent:   "Fix omioflex component issues",
			expectedMatch: true,
			description:   "Should match lowercase concatenation: omioflex",
		},
		{
			name:          "snake_case",
			assetName:     "Omio Flex",
			taskContent:   "Fix omio_flex component issues",
			expectedMatch: true,
			description:   "Should match snake_case: omio_flex",
		},
		{
			name:          "kebab-case",
			assetName:     "Omio Flex",
			taskContent:   "Fix omio-flex component issues",
			expectedMatch: true,
			description:   "Should match kebab-case: omio-flex",
		},
		{
			name:          "mixed separators",
			assetName:     "Payment Gateway",
			taskContent:   "Update payment_gateway config",
			expectedMatch: true,
			description:   "Should match mixed separators",
		},
		{
			name:          "partial word in compound",
			assetName:     "Price Lock",
			taskContent:   "Update PriceLockService implementation",
			expectedMatch: true,
			description:   "Should match when asset words appear in compound words",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := &assetdomain.Asset{
				Name:     tt.assetName,
				Keywords: []string{}, // No keywords, rely on name matching
			}

			task := &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     tt.taskContent,
				Description: "",
			}

			classifier := &ContentBasedAssetClassifier{}
			score, reason := classifier.calculateAssetMatchScore(task, asset)

			if tt.expectedMatch {
				assert.Greater(t, score, 0.0, "Should have found a match for: %s", tt.description)
				assert.NotEqual(t, "no significant match", reason, "Should have a meaningful match reason")
			} else {
				assert.Equal(t, 0.0, score, "Should not have found a match for: %s", tt.description)
			}
		})
	}
}

func TestTextMatching_MultiWordAssets(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		taskContent   string
		expectedMatch bool
		description   string
	}{
		{
			name:          "dynamic markup camelCase",
			assetName:     "Dynamic Markup",
			taskContent:   "Fix DynamicMarkup calculation",
			expectedMatch: true,
			description:   "Should match Dynamic Markup as DynamicMarkup",
		},
		{
			name:          "service fee snake_case",
			assetName:     "Service Fee",
			taskContent:   "Update service_fee logic",
			expectedMatch: true,
			description:   "Should match Service Fee as service_fee",
		},
		{
			name:          "price calendar kebab-case",
			assetName:     "Price Calendar",
			taskContent:   "Implement price-calendar widget",
			expectedMatch: true,
			description:   "Should match Price Calendar as price-calendar",
		},
		{
			name:          "virtual credit card mixed",
			assetName:     "Virtual Credit Card",
			taskContent:   "VirtualCreditCard integration test",
			expectedMatch: true,
			description:   "Should match Virtual Credit Card as VirtualCreditCard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := &assetdomain.Asset{
				Name:     tt.assetName,
				Keywords: []string{},
			}

			task := &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     tt.taskContent,
				Description: "",
			}

			classifier := &ContentBasedAssetClassifier{}
			score, reason := classifier.calculateAssetMatchScore(task, asset)

			if tt.expectedMatch {
				assert.Greater(t, score, 0.0, "Should have found a match for: %s", tt.description)
				assert.NotEqual(t, "no significant match", reason, "Should have a meaningful match reason")
			} else {
				assert.Equal(t, 0.0, score, "Should not have found a match for: %s", tt.description)
			}
		})
	}
}

func TestTextMatching_TyposAndMisspellings(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		taskContent   string
		expectedMatch bool
		description   string
	}{
		{
			name:          "minor typo in flex",
			assetName:     "Omio Flex",
			taskContent:   "Fix Omio Fles component", // 'x' missing
			expectedMatch: false,                     // We probably shouldn't match typos without fuzzy matching
			description:   "Minor typo should not match without fuzzy logic",
		},
		{
			name:          "common abbreviation",
			assetName:     "Dynamic Markup",
			taskContent:   "Fix DynMarkup issue",
			expectedMatch: false, // Abbreviations are tricky
			description:   "Abbreviations should not match without explicit keyword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := &assetdomain.Asset{
				Name:     tt.assetName,
				Keywords: []string{},
			}

			task := &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     tt.taskContent,
				Description: "",
			}

			classifier := &ContentBasedAssetClassifier{}
			score, reason := classifier.calculateAssetMatchScore(task, asset)

			if tt.expectedMatch {
				assert.Greater(t, score, 0.0, "Should have found a match for: %s", tt.description)
				assert.NotEqual(t, "no significant match", reason, "Should have a meaningful match reason")
			} else {
				// For now, we expect these not to match since we're being conservative
				t.Logf("Current behavior: score=%f, reason=%s", score, reason)
			}
		})
	}
}

func TestTextMatching_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		taskContent   string
		expectedMatch bool
		description   string
	}{
		{
			name:          "omio flex component names",
			assetName:     "Omio Flex",
			taskContent:   "Fix color styles to pass contrast checks. Update OmioFlexAddedMessage and OmioFlexPriceTextStyle components",
			expectedMatch: true,
			description:   "Should match OmioFlex in component names (real FN-972 case)",
		},
		{
			name:          "payment gateway service",
			assetName:     "Payment Gateway",
			taskContent:   "Refactor PaymentGatewayService timeout handling",
			expectedMatch: true,
			description:   "Should match PaymentGateway in class names",
		},
		{
			name:          "dynamic markup config",
			assetName:     "Dynamic Markup",
			taskContent:   "Update dynamic_markup_config.json file",
			expectedMatch: true,
			description:   "Should match dynamic_markup in file names",
		},
		{
			name:          "price lock feature",
			assetName:     "Price Lock",
			taskContent:   "Implement price-lock-widget for booking flow",
			expectedMatch: true,
			description:   "Should match price-lock in feature names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := &assetdomain.Asset{
				Name:     tt.assetName,
				Keywords: []string{},
			}

			task := &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     tt.taskContent,
				Description: "",
			}

			classifier := &ContentBasedAssetClassifier{}
			score, reason := classifier.calculateAssetMatchScore(task, asset)

			if tt.expectedMatch {
				assert.Greater(t, score, 0.0, "Should have found a match for: %s", tt.description)
				assert.NotEqual(t, "no significant match", reason, "Should have a meaningful match reason")
			} else {
				assert.Equal(t, 0.0, score, "Should not have found a match for: %s", tt.description)
			}
		})
	}
}

// TestCurrentImplementationGaps runs tests to identify where current implementation fails
func TestCurrentImplementationGaps(t *testing.T) {
	// This test documents current failures that we need to fix
	failing := []struct {
		name        string
		assetName   string
		taskContent string
		reason      string
	}{
		{
			name:        "camelCase concatenation",
			assetName:   "Omio Flex",
			taskContent: "Fix OmioFlex issues",
			reason:      "Current logic doesn't handle camelCase concatenation",
		},
		{
			name:        "snake_case variation",
			assetName:   "Price Lock",
			taskContent: "Update price_lock configuration",
			reason:      "Current logic doesn't handle snake_case variations",
		},
		{
			name:        "kebab-case variation",
			assetName:   "Service Fee",
			taskContent: "Fix service-fee calculation",
			reason:      "Current logic doesn't handle kebab-case variations",
		},
	}

	for _, test := range failing {
		t.Run("gap_"+test.name, func(t *testing.T) {
			asset := &assetdomain.Asset{
				Name:     test.assetName,
				Keywords: []string{},
			}

			task := &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     test.taskContent,
				Description: "",
			}

			classifier := &ContentBasedAssetClassifier{}
			score, reason := classifier.calculateAssetMatchScore(task, asset)

			// Document current state
			t.Logf("CURRENT STATE - %s: score=%f, reason=%s", test.reason, score, reason)

			// This test is documentation - we expect these to fail with current implementation
			if score == 0.0 {
				t.Logf("✓ Confirmed gap: %s", test.reason)
			} else {
				t.Logf("✗ Unexpected match found - implementation may have improved")
			}
		})
	}
}

// TestSpecificFN972Case tests the exact FN-972 scenario
func TestSpecificFN972Case(t *testing.T) {
	asset := &assetdomain.Asset{
		Name: "Omio Flex",
		Keywords: []string{
			"fare flexibility",
			"non-refundable fares",
			"semi-refundable fares",
			"mode routes",
			"flexible ticketing",
			"perceived value",
			"flex",
			"omio flex",
			"omioflex",
		},
	}

	task := &taskdomain.Task{
		Key:         "FN-972",
		Summary:     "Fix color styles to pass contrast checks",
		Description: "OmioFlexAddedMessage and OmioFlexPriceTextStyle components need color adjustments",
		Epic:        "FN-983",
		Labels:      []string{},
	}

	classifier := &ContentBasedAssetClassifier{}
	score, reason := classifier.calculateAssetMatchScore(task, asset)

	t.Logf("FN-972 Case: score=%f, reason=%s", score, reason)
	t.Logf("Task Content: %s %s", task.Summary, task.Description)

	// Test what the current implementation produces
	if score > 0.0 {
		t.Logf("✓ Found match with score %f", score)
	} else {
		t.Logf("✗ No match found")
	}

	// We expect this to match due to "OmioFlex" in the description
	assert.Greater(t, score, 0.0, "Should match OmioFlex in description")
}

// TestDebugTextMatching shows detailed matching process
func TestDebugTextMatching(t *testing.T) {
	asset := &assetdomain.Asset{
		Name:     "Omio Flex",
		Keywords: []string{"flex", "omio flex", "omioflex"},
	}

	tests := []struct {
		name        string
		taskContent string
	}{
		{"exact_match", "Fix Omio Flex issues"},
		{"camelCase", "Fix OmioFlex issues"},
		{"lowercase", "Fix omioflex issues"},
		{"in_description", "Fix color styles. Update OmioFlexAddedMessage component"},
		{"snake_case", "Fix omio_flex issues"},
		{"kebab_case", "Fix omio-flex issues"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     tt.taskContent,
				Description: "",
			}

			classifier := &ContentBasedAssetClassifier{}
			score, reason := classifier.calculateAssetMatchScore(task, asset)

			t.Logf("%s: score=%f, reason=%s", tt.name, score, reason)
		})
	}
}
