package classifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

func newTestAssets() []*assetdomain.Asset {
	return []*assetdomain.Asset{
		{Name: "Dynamic Pricing", Description: "Real-time price optimization", Keywords: []string{"pricing", "dynamic", "rate"}},
		{Name: "Search Results Page", Description: "Main search results display", Keywords: []string{"SRP", "search", "results"}},
		{Name: "Cabin Markup", Description: "Cabin-level markup management", Keywords: []string{"cabin", "markup", "fare"}},
	}
}

func ollamaHandler(responseJSON string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaResponse{Response: responseJSON}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func TestLLMAssetClassifier_ValidResponse(t *testing.T) {
	llmResponse := `{"asset_name": "Dynamic Pricing", "confidence": 0.92, "reasoning": "Task mentions price optimization which directly relates to Dynamic Pricing"}`
	server := httptest.NewServer(ollamaHandler(llmResponse))
	defer server.Close()

	repo := &mockAssetRepo{assets: newTestAssets()}
	classifier := NewLLMAssetClassifier(server.URL, "llama4", repo)

	task := &taskdomain.Task{
		Key:     "COP-123",
		Summary: "Update dynamic pricing algorithm for peak hours",
		Type:    "Story",
	}

	result, err := classifier.ClassifyTaskAsset(task)
	require.NoError(t, err)
	assert.NotNil(t, result.Asset)
	assert.Equal(t, "Dynamic Pricing", result.Asset.Name)
	assert.InDelta(t, 0.92, result.Confidence, 0.01)
	assert.Contains(t, result.Reason, "LLM:")
}

func TestLLMAssetClassifier_JSONWrappedInProse(t *testing.T) {
	llmResponse := `Based on my analysis, here is the classification:
{"asset_name": "Search Results Page", "confidence": 0.88, "reasoning": "SRP related task"}
I hope this helps!`
	server := httptest.NewServer(ollamaHandler(llmResponse))
	defer server.Close()

	repo := &mockAssetRepo{assets: newTestAssets()}
	classifier := NewLLMAssetClassifier(server.URL, "llama4", repo)

	task := &taskdomain.Task{
		Key:     "COP-456",
		Summary: "AB Mode Comparison on SRP - Desktop Web",
		Type:    "Story",
	}

	result, err := classifier.ClassifyTaskAsset(task)
	require.NoError(t, err)
	assert.NotNil(t, result.Asset)
	assert.Equal(t, "Search Results Page", result.Asset.Name)
	assert.InDelta(t, 0.88, result.Confidence, 0.01)
}

func TestLLMAssetClassifier_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(ollamaHandler("I cannot determine the asset classification."))
	defer server.Close()

	repo := &mockAssetRepo{assets: newTestAssets()}
	classifier := NewLLMAssetClassifier(server.URL, "llama4", repo)

	task := &taskdomain.Task{Key: "COP-789", Summary: "Some task"}

	result, err := classifier.ClassifyTaskAsset(task)
	require.NoError(t, err)
	assert.Nil(t, result.Asset)
	assert.Equal(t, float64(0), result.Confidence)
	assert.Contains(t, result.Reason, "no valid JSON")
}

func TestLLMAssetClassifier_OllamaUnreachable(t *testing.T) {
	repo := &mockAssetRepo{assets: newTestAssets()}
	classifier := NewLLMAssetClassifier("http://localhost:99999", "llama4", repo)

	task := &taskdomain.Task{Key: "COP-001", Summary: "Test task"}

	_, err := classifier.ClassifyTaskAsset(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM classification failed")
}

func TestLLMAssetClassifier_EmptyAssetList(t *testing.T) {
	server := httptest.NewServer(ollamaHandler(`{}`))
	defer server.Close()

	repo := &mockAssetRepo{assets: []*assetdomain.Asset{}}
	classifier := NewLLMAssetClassifier(server.URL, "llama4", repo)

	task := &taskdomain.Task{Key: "COP-002", Summary: "Test task"}

	result, err := classifier.ClassifyTaskAsset(task)
	require.NoError(t, err)
	assert.Nil(t, result.Asset)
	assert.Equal(t, float64(0), result.Confidence)
	assert.Contains(t, result.Reason, "no assets available")
}

func TestLLMAssetClassifier_FuzzyNameMatching(t *testing.T) {
	tests := []struct {
		name          string
		llmAssetName  string
		expectedAsset string
	}{
		{"case insensitive", "dynamic pricing", "Dynamic Pricing"},
		{"partial match - contains", "Pricing", "Dynamic Pricing"},
		{"partial match - asset contains query", "Dynamic Pricing Engine", "Dynamic Pricing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llmResponse := `{"asset_name": "` + tt.llmAssetName + `", "confidence": 0.8, "reasoning": "test"}`
			server := httptest.NewServer(ollamaHandler(llmResponse))
			defer server.Close()

			repo := &mockAssetRepo{assets: newTestAssets()}
			classifier := NewLLMAssetClassifier(server.URL, "llama4", repo)

			task := &taskdomain.Task{Key: "COP-100", Summary: "Test"}
			result, err := classifier.ClassifyTaskAsset(task)
			require.NoError(t, err)
			assert.NotNil(t, result.Asset, "expected asset match for %q", tt.llmAssetName)
			assert.Equal(t, tt.expectedAsset, result.Asset.Name)
		})
	}
}

func TestLLMAssetClassifier_ConfidenceClamping(t *testing.T) {
	tests := []struct {
		name               string
		confidence         float64
		expectedConfidence float64
	}{
		{"above 1.0", 1.5, 1.0},
		{"below 0", -0.5, 0.0},
		{"normal", 0.75, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llmResp, _ := json.Marshal(llmClassificationResponse{
				AssetName:  "Dynamic Pricing",
				Confidence: tt.confidence,
				Reasoning:  "test",
			})
			server := httptest.NewServer(ollamaHandler(string(llmResp)))
			defer server.Close()

			repo := &mockAssetRepo{assets: newTestAssets()}
			classifier := NewLLMAssetClassifier(server.URL, "llama4", repo)

			task := &taskdomain.Task{Key: "COP-200", Summary: "Test"}
			result, err := classifier.ClassifyTaskAsset(task)
			require.NoError(t, err)
			assert.InDelta(t, tt.expectedConfidence, result.Confidence, 0.01)
		})
	}
}

func TestLLMAssetClassifier_NilTask(t *testing.T) {
	repo := &mockAssetRepo{assets: newTestAssets()}
	classifier := NewLLMAssetClassifier("http://localhost:11434", "llama4", repo)

	_, err := classifier.ClassifyTaskAsset(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task cannot be nil")
}

func TestLLMAssetClassifier_NoneAsset(t *testing.T) {
	llmResponse := `{"asset_name": "none", "confidence": 0.0, "reasoning": "No matching asset found"}`
	server := httptest.NewServer(ollamaHandler(llmResponse))
	defer server.Close()

	repo := &mockAssetRepo{assets: newTestAssets()}
	classifier := NewLLMAssetClassifier(server.URL, "llama4", repo)

	task := &taskdomain.Task{Key: "COP-300", Summary: "Team standup meeting"}

	result, err := classifier.ClassifyTaskAsset(task)
	require.NoError(t, err)
	assert.Nil(t, result.Asset)
	assert.Equal(t, float64(0), result.Confidence)
	assert.Contains(t, result.Reason, "no matching asset")
}

func TestLLMAssetClassifier_NoMatchingAssetForSuggestedName(t *testing.T) {
	llmResponse := `{"asset_name": "Nonexistent Asset", "confidence": 0.9, "reasoning": "test"}`
	server := httptest.NewServer(ollamaHandler(llmResponse))
	defer server.Close()

	repo := &mockAssetRepo{assets: newTestAssets()}
	classifier := NewLLMAssetClassifier(server.URL, "llama4", repo)

	task := &taskdomain.Task{Key: "COP-400", Summary: "Test"}

	result, err := classifier.ClassifyTaskAsset(task)
	require.NoError(t, err)
	assert.Nil(t, result.Asset)
	assert.Equal(t, float64(0), result.Confidence)
	assert.Contains(t, result.Reason, "no matching asset found")
}

func TestLLMAssetClassifier_ClassifyTasksAssets(t *testing.T) {
	llmResponse := `{"asset_name": "Dynamic Pricing", "confidence": 0.85, "reasoning": "pricing task"}`
	server := httptest.NewServer(ollamaHandler(llmResponse))
	defer server.Close()

	repo := &mockAssetRepo{assets: newTestAssets()}
	classifier := NewLLMAssetClassifier(server.URL, "llama4", repo)

	tasks := []*taskdomain.Task{
		{Key: "COP-500", Summary: "Update pricing"},
		{Key: "COP-501", Summary: "Fix pricing bug"},
	}

	results, err := classifier.ClassifyTasksAssets(tasks)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.NotNil(t, r.Asset)
		assert.Equal(t, "Dynamic Pricing", r.Asset.Name)
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"pure JSON", `{"key": "value"}`, `{"key": "value"}`},
		{"JSON in prose", `Here is the result: {"key": "value"} done.`, `{"key": "value"}`},
		{"nested JSON", `{"outer": {"inner": "val"}}`, `{"outer": {"inner": "val"}}`},
		{"no JSON", "just text", ""},
		{"unclosed brace", `{"key": "value"`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractJSON(tt.input))
		})
	}
}

func TestFuzzyMatchAsset(t *testing.T) {
	assets := newTestAssets()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"exact match", "Dynamic Pricing", "Dynamic Pricing"},
		{"case insensitive", "dynamic pricing", "Dynamic Pricing"},
		{"contains in asset", "Pricing", "Dynamic Pricing"},
		{"asset name contained in input", "Dynamic Pricing Engine V2", "Dynamic Pricing"},
		{"no match", "Nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fuzzyMatchAsset(tt.input, assets)
			if tt.expected == "" {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected, result.Name)
			}
		})
	}
}

func TestLLMAssetClassifier_BuildPrompt(t *testing.T) {
	repo := &mockAssetRepo{assets: newTestAssets()}
	c := NewLLMAssetClassifier("http://localhost:11434", "llama4", repo).(*LLMAssetClassifier)

	task := &taskdomain.Task{
		Key:         "COP-123",
		Summary:     "Update pricing algorithm",
		Description: "Detailed description of the task",
		Type:        "Story",
		Epic:        "COP-100",
		Labels:      []string{"backend", "pricing"},
	}

	prompt := c.buildPrompt(task, newTestAssets())

	assert.Contains(t, prompt, "COP-123")
	assert.Contains(t, prompt, "Update pricing algorithm")
	assert.Contains(t, prompt, "Dynamic Pricing")
	assert.Contains(t, prompt, "Search Results Page")
	assert.Contains(t, prompt, "Cabin Markup")
	assert.Contains(t, prompt, "asset_name")
	assert.Contains(t, prompt, "confidence")
}

func TestLLMAssetClassifier_OllamaServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	repo := &mockAssetRepo{assets: newTestAssets()}
	classifier := NewLLMAssetClassifier(server.URL, "llama4", repo)

	task := &taskdomain.Task{Key: "COP-600", Summary: "Test task"}

	_, err := classifier.ClassifyTaskAsset(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM classification failed")
}
