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

// batchStubAssetClassifier is a hand-rolled stub for the batch
// classification path. The single-task stubAssetClassifier in
// inherit_from_parent_test.go returns nil from ClassifyTasksAssets,
// which doesn't exercise runBatchLLMClassification's match loop.
type batchStubAssetClassifier struct {
	batchResults []*ports.AssetClassificationResult
	batchErr     error
}

func (b *batchStubAssetClassifier) ClassifyTaskAsset(*taskdomain.Task) (*ports.AssetClassificationResult, error) {
	return nil, nil
}
func (b *batchStubAssetClassifier) ClassifyTasksAssets([]*taskdomain.Task) ([]*ports.AssetClassificationResult, error) {
	return b.batchResults, b.batchErr
}

func TestRunBatchLLMClassification_AttachesByTaskKey(t *testing.T) {
	tasks := []*taskdomain.Task{{Key: "FN-1"}, {Key: "FN-2"}, {Key: "FN-3"}}
	results := []*ports.ComprehensiveClassificationResult{
		{Task: tasks[0]},
		{Task: tasks[1]},
		{Task: tasks[2]},
	}

	// Returns a result for FN-1 and FN-3 (FN-2 missing) plus a stray nil
	// entry to prove the nil filter in the lookup loop is exercised.
	chain := &ComprehensiveClassificationChainWithInheritance{
		llmClassifier: &batchStubAssetClassifier{
			batchResults: []*ports.AssetClassificationResult{
				{Task: tasks[0], Asset: &assetdomain.Asset{Name: "Search"}},
				nil,
				{Task: tasks[2], Asset: &assetdomain.Asset{Name: "Payments"}},
				{Task: nil, Asset: &assetdomain.Asset{Name: "Should be skipped"}},
			},
		},
	}

	chain.runBatchLLMClassification(tasks, results)
	require.NotNil(t, results[0].LLMAsset)
	assert.Equal(t, "Search", results[0].LLMAsset.Asset.Name)
	assert.Nil(t, results[1].LLMAsset, "FN-2 has no LLM result so it stays nil")
	require.NotNil(t, results[2].LLMAsset)
	assert.Equal(t, "Payments", results[2].LLMAsset.Asset.Name)
}

func TestRunBatchLLMClassification_ErrorIsLoggedAndResultsUntouched(t *testing.T) {
	tasks := []*taskdomain.Task{{Key: "FN-1"}}
	results := []*ports.ComprehensiveClassificationResult{{Task: tasks[0]}}

	chain := &ComprehensiveClassificationChainWithInheritance{
		llmClassifier: &batchStubAssetClassifier{batchErr: errors.New("model busy")},
	}
	chain.runBatchLLMClassification(tasks, results)
	assert.Nil(t, results[0].LLMAsset, "errored classifier must leave results untouched")
}
