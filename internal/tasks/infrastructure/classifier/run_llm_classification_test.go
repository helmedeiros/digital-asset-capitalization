package classifier

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

func TestRunLLMClassification_DisabledOrNilIsNoop(t *testing.T) {
	t.Run("llmEnabled false short-circuits even with classifier set", func(t *testing.T) {
		chain := &ComprehensiveClassificationChainWithInheritance{
			llmClassifier: &stubAssetClassifier{
				result: &ports.AssetClassificationResult{Asset: &assetdomain.Asset{Name: "X"}},
			},
			llmEnabled: false,
		}
		result := &ports.ComprehensiveClassificationResult{}
		chain.runLLMClassification(&taskdomain.Task{Key: "FN-1"}, result)
		assert.Nil(t, result.LLMAsset, "disabled flag must skip the call entirely")
	})

	t.Run("nil llmClassifier short-circuits even when enabled", func(t *testing.T) {
		chain := &ComprehensiveClassificationChainWithInheritance{
			llmEnabled: true,
			// llmClassifier intentionally nil
		}
		result := &ports.ComprehensiveClassificationResult{}
		chain.runLLMClassification(&taskdomain.Task{Key: "FN-1"}, result)
		assert.Nil(t, result.LLMAsset)
	})
}

func TestRunLLMClassification_ErrorIsLoggedAndResultUnchanged(t *testing.T) {
	chain := &ComprehensiveClassificationChainWithInheritance{
		llmClassifier: &stubAssetClassifier{err: errors.New("model offline")},
		llmEnabled:    true,
	}
	result := &ports.ComprehensiveClassificationResult{}
	chain.runLLMClassification(&taskdomain.Task{Key: "FN-1"}, result)
	assert.Nil(t, result.LLMAsset, "errored classifier must not populate LLMAsset")
}

func TestRunLLMClassification_HappyPathPopulatesLLMAsset(t *testing.T) {
	expected := &ports.AssetClassificationResult{
		Asset:      &assetdomain.Asset{Name: "Payments"},
		Confidence: 0.7,
		Reason:     "llm classification",
	}
	chain := &ComprehensiveClassificationChainWithInheritance{
		llmClassifier: &stubAssetClassifier{result: expected},
		llmEnabled:    true,
	}
	result := &ports.ComprehensiveClassificationResult{}
	chain.runLLMClassification(&taskdomain.Task{Key: "FN-1"}, result)
	require.NotNil(t, result.LLMAsset)
	assert.Same(t, expected, result.LLMAsset)
}
