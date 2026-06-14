package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// stubAssetClassifier is a hand-rolled stub so the tests can pin
// behaviour without the testify mock setup ceremony. The chain's
// classifier ports only call ClassifyTaskAsset / ClassifyTask in this
// path, so we don't need to implement the batch methods.
type stubAssetClassifier struct {
	result *ports.AssetClassificationResult
	err    error
}

func (s *stubAssetClassifier) ClassifyTaskAsset(*taskdomain.Task) (*ports.AssetClassificationResult, error) {
	return s.result, s.err
}
func (s *stubAssetClassifier) ClassifyTasksAssets([]*taskdomain.Task) ([]*ports.AssetClassificationResult, error) {
	return nil, nil
}

type stubWorkTypeClassifier struct {
	wt  taskdomain.WorkType
	err error
}

func (s *stubWorkTypeClassifier) ClassifyTask(*taskdomain.Task) (taskdomain.WorkType, error) {
	return s.wt, s.err
}
func (s *stubWorkTypeClassifier) ClassifyTasks([]*taskdomain.Task) (map[string]taskdomain.WorkType, error) {
	return nil, nil
}

func TestInheritFromParent_ParentNotInLookup(t *testing.T) {
	chain := &ComprehensiveClassificationChainWithInheritance{
		taskLookup: map[string]*taskdomain.Task{},
	}
	task := &taskdomain.Task{Key: "FN-2", Epic: "FN-999"}

	gotAsset, gotWT, gotReason := chain.inheritFromParent(task)
	assert.Nil(t, gotAsset)
	assert.Equal(t, taskdomain.WorkType(""), gotWT)
	assert.Equal(t, "", gotReason)
}

func TestInheritFromParent_DiscoveryLabelMaps(t *testing.T) {
	parent := &taskdomain.Task{
		Key:    "FN-1",
		Labels: []string{"cap-asset-search", "cap-discovery"},
	}
	chain := &ComprehensiveClassificationChainWithInheritance{
		taskLookup: map[string]*taskdomain.Task{"FN-1": parent},
	}
	task := &taskdomain.Task{Key: "FN-2", Epic: "FN-1"}

	gotAsset, gotWT, gotReason := chain.inheritFromParent(task)
	require.NotNil(t, gotAsset)
	require.NotNil(t, gotAsset.Asset)
	assert.Equal(t, "Search", gotAsset.Asset.Name)
	assert.Equal(t, taskdomain.WorkTypeDiscovery, gotWT)
	assert.Contains(t, gotReason, "FN-1")
}

func TestInheritFromParent_FallsBackToDirectClassification(t *testing.T) {
	parent := &taskdomain.Task{
		Key:     "FN-1",
		Summary: "Parent",
		// No cap-* labels — forces the fallback branch where the chain
		// calls assetClassifier/workTypeClassifier directly on the parent.
	}
	stubAsset := &stubAssetClassifier{
		result: &ports.AssetClassificationResult{
			Task:       parent,
			Asset:      &assetdomain.Asset{Name: "Payments"},
			Confidence: 0.5,
			Reason:     "matched payments",
		},
	}
	stubWT := &stubWorkTypeClassifier{wt: taskdomain.WorkTypeMaintenance}

	chain := &ComprehensiveClassificationChainWithInheritance{
		assetClassifier:    stubAsset,
		workTypeClassifier: stubWT,
		taskLookup:         map[string]*taskdomain.Task{"FN-1": parent},
	}
	task := &taskdomain.Task{Key: "FN-2", Epic: "FN-1"}

	gotAsset, gotWT, gotReason := chain.inheritFromParent(task)
	require.NotNil(t, gotAsset)
	require.NotNil(t, gotAsset.Asset)
	assert.Equal(t, "Payments", gotAsset.Asset.Name)
	// Confidence is scaled by 0.8 on the inherited path.
	assert.InDelta(t, 0.4, gotAsset.Confidence, 0.0001)
	assert.Contains(t, gotAsset.Reason, "inherited from parent task FN-1")
	assert.Equal(t, taskdomain.WorkTypeMaintenance, gotWT)
	assert.Contains(t, gotReason, "FN-1")
}

func TestInheritFromParent_ClassifierErrorsLeaveInheritsZeroed(t *testing.T) {
	parent := &taskdomain.Task{Key: "FN-1", Summary: "Parent"}
	chain := &ComprehensiveClassificationChainWithInheritance{
		assetClassifier:    &stubAssetClassifier{err: assert.AnError},
		workTypeClassifier: &stubWorkTypeClassifier{err: assert.AnError},
		taskLookup:         map[string]*taskdomain.Task{"FN-1": parent},
	}
	task := &taskdomain.Task{Key: "FN-2", Epic: "FN-1"}

	gotAsset, gotWT, gotReason := chain.inheritFromParent(task)
	// Asset/work-type stay zero because both classifiers errored. The
	// reason is still populated since the parent existed.
	assert.Nil(t, gotAsset)
	assert.Equal(t, taskdomain.WorkType(""), gotWT)
	assert.Contains(t, gotReason, "FN-1")
}
