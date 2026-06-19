package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// togglerComprehensiveClassifier embeds MockComprehensiveTaskClassifier
// and additionally satisfies LLMToggler so the previewClassificationsWithRetry
// withLLM branch can flip the toggle on/off via the deferred reset.
type togglerComprehensiveClassifier struct {
	MockComprehensiveTaskClassifier
	enabledCalls []bool
}

func (t *togglerComprehensiveClassifier) SetLLMEnabled(enabled bool) {
	t.enabledCalls = append(t.enabledCalls, enabled)
}

// LLM-mode preview rendering branches. The Execute table-driven tests
// cover the heuristic-only path; this file pins the withLLM=true
// rendering paths: LLMToggler set+reset, LLM result alongside
// heuristic result, LLM-only result, agreement vs disagreement
// branches, and the LLM-comparison summary footer.

func TestClassifyTasksUseCase_previewClassificationsWithRetry_LLMTogglerSetAndReset(t *testing.T) {
	t.Parallel()
	classifier := &togglerComprehensiveClassifier{}
	uc := NewClassifyTasksUseCase(nil, nil, classifier, nil, testutil.NewMockAssetService(), nil)

	tasks := []*domain.Task{{Key: "T-1", Summary: "S"}}
	results := []*ports.ComprehensiveClassificationResult{{
		Task:     tasks[0],
		WorkType: domain.WorkTypeDevelopment,
		Asset:    nil,
	}}
	classifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)

	require.NoError(t, uc.previewClassificationsWithRetry(tasks, true, true))
	// Toggle should fire once with true (entering) and once with false (deferred reset).
	assert.Equal(t, []bool{true, false}, classifier.enabledCalls)
	classifier.AssertExpectations(t)
}

func TestClassifyTasksUseCase_previewClassificationsWithRetry_LLMAgreementSummary(t *testing.T) {
	t.Parallel()
	classifier := &togglerComprehensiveClassifier{}
	uc := NewClassifyTasksUseCase(nil, nil, classifier, nil, testutil.NewMockAssetService(), nil)

	asset := assetClassRes("PaymentGateway", 0.9, "label match")
	tasks := []*domain.Task{{Key: "T-1", Summary: "S", Type: "Task", Status: "Done"}}
	// Heuristic and LLM agree on the same asset.
	results := []*ports.ComprehensiveClassificationResult{{
		Task:           tasks[0],
		WorkType:       domain.WorkTypeDevelopment,
		Asset:          asset,
		LLMAsset:       asset,
		WorkTypeReason: "matched dev keywords",
	}}
	classifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)

	require.NoError(t, uc.previewClassificationsWithRetry(tasks, true, true))
	classifier.AssertExpectations(t)
}

func TestClassifyTasksUseCase_previewClassificationsWithRetry_LLMDisagreementSummary(t *testing.T) {
	t.Parallel()
	classifier := &togglerComprehensiveClassifier{}
	uc := NewClassifyTasksUseCase(nil, nil, classifier, nil, testutil.NewMockAssetService(), nil)

	tasks := []*domain.Task{{Key: "T-1", Summary: "S", Type: "Task", Status: "Done", Epic: "EPIC-1", Labels: []string{"a", "b"}}}
	results := []*ports.ComprehensiveClassificationResult{{
		Task:     tasks[0],
		WorkType: domain.WorkTypeDevelopment,
		// Heuristic picks PaymentGateway, LLM picks Checkout → disagreement.
		Asset:          assetClassRes("PaymentGateway", 0.9, "match A"),
		LLMAsset:       assetClassRes("Checkout", 0.7, "match B"),
		WorkTypeReason: "reason",
	}}
	classifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)

	require.NoError(t, uc.previewClassificationsWithRetry(tasks, true, true))
	classifier.AssertExpectations(t)
}

func TestClassifyTasksUseCase_previewClassificationsWithRetry_LLMOnlyResultNoHeuristic(t *testing.T) {
	t.Parallel()
	// Heuristic returned no asset (Asset is nil); LLM did. Hits the
	// "No assignment found" heuristic branch AND the LLM rendering
	// branch in the same result.
	classifier := &togglerComprehensiveClassifier{}
	uc := NewClassifyTasksUseCase(nil, nil, classifier, nil, testutil.NewMockAssetService(), nil)

	tasks := []*domain.Task{{Key: "T-1", Summary: "S", Type: "Task", Status: "Done"}}
	results := []*ports.ComprehensiveClassificationResult{{
		Task:     tasks[0],
		WorkType: domain.WorkTypeDevelopment,
		Asset:    nil,
		LLMAsset: assetClassRes("Checkout", 0.7, "from LLM"),
	}}
	classifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)

	require.NoError(t, uc.previewClassificationsWithRetry(tasks, true, true))
	classifier.AssertExpectations(t)
}

func TestClassifyTasksUseCase_previewClassificationsWithRetry_LLMHadNoAsset(t *testing.T) {
	t.Parallel()
	// LLMAsset is non-nil but its inner Asset is nil — exercises the
	// "[LLM] Asset: No assignment found" branch.
	classifier := &togglerComprehensiveClassifier{}
	uc := NewClassifyTasksUseCase(nil, nil, classifier, nil, testutil.NewMockAssetService(), nil)

	tasks := []*domain.Task{{Key: "T-1", Summary: "S"}}
	results := []*ports.ComprehensiveClassificationResult{{
		Task:     tasks[0],
		WorkType: domain.WorkTypeDevelopment,
		Asset:    assetClassRes("PaymentGateway", 0.9, "match"),
		LLMAsset: &ports.AssetClassificationResult{
			Asset:      nil,
			Confidence: 0.0,
			Reason:     "",
		},
	}}
	classifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)

	require.NoError(t, uc.previewClassificationsWithRetry(tasks, true, true))
	classifier.AssertExpectations(t)
}

// assetClassRes builds an AssetClassificationResult wrapping a new
// asset with the given name + confidence + reason. Tests use this
// to avoid repeating the domain.NewAsset boilerplate.
func assetClassRes(name string, confidence float64, reason string) *ports.AssetClassificationResult {
	asset, _ := assetsdomain.NewAsset(name, "Test description")
	return &ports.AssetClassificationResult{
		Asset:      asset,
		Confidence: confidence,
		Reason:     reason,
	}
}
