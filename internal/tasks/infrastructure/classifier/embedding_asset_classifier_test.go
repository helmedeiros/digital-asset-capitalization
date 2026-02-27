package classifier

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// mockEmbeddingService implements ports.EmbeddingService for testing
type mockEmbeddingService struct {
	embedFn func(texts []string) ([][]float64, error)
	calls   int
}

func (m *mockEmbeddingService) Embed(texts []string) ([][]float64, error) {
	m.calls++
	return m.embedFn(texts)
}

func newTestStore(t *testing.T) *EmbeddingStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "embeddings.json")
	store, err := NewEmbeddingStore(path)
	require.NoError(t, err)
	return store
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{-1.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        []float64{1.0, 1.0, 0.0},
			b:        []float64{1.0, 0.0, 0.0},
			expected: 1.0 / math.Sqrt(2.0),
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
		},
		{
			name:     "mismatched lengths",
			a:        []float64{1.0, 2.0},
			b:        []float64{1.0},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			a:        []float64{0.0, 0.0},
			b:        []float64{1.0, 1.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 1e-9)
		})
	}
}

func TestNormalizeScore(t *testing.T) {
	tests := []struct {
		raw      float64
		expected float64
	}{
		{0.5, 0.0},
		{0.75, 0.5},
		{1.0, 1.0},
		{0.3, 0.0},
		{1.5, 1.0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("raw=%.2f", tt.raw), func(t *testing.T) {
			assert.InDelta(t, tt.expected, normalizeScore(tt.raw), 1e-9)
		})
	}
}

func TestBuildTaskText(t *testing.T) {
	task := &taskdomain.Task{
		Summary:     "Fix carrier comparison",
		Description: "Update the carrier comparison logic",
		Epic:        "Carrier Optimization",
		Project:     "COP",
		Labels:      []string{"cap-development", "cap-asset-carrier-comparison", "urgent"},
	}

	text := buildTaskText(task)
	assert.Contains(t, text, "Task: Fix carrier comparison")
	assert.Contains(t, text, "Update the carrier comparison logic")
	assert.Contains(t, text, "Epic: Carrier Optimization")
	assert.Contains(t, text, "Project: COP")
	assert.Contains(t, text, "cap-development")
	assert.Contains(t, text, "urgent")
	assert.NotContains(t, text, "cap-asset-carrier-comparison", "cap-asset-* labels should be excluded")
}

func TestBuildTaskText_Truncation(t *testing.T) {
	longDesc := ""
	for i := 0; i < 600; i++ {
		longDesc += "x"
	}
	task := &taskdomain.Task{
		Summary:     "Short",
		Description: longDesc,
	}

	text := buildTaskText(task)
	assert.LessOrEqual(t, len(text), maxTaskTextLen)
}

func TestBuildAssetText(t *testing.T) {
	asset := &assetdomain.Asset{
		Name:        "Carrier Comparison",
		Description: "Compares carriers",
		Why:         "Cost savings",
		Benefits:    "Better rates",
		How:         "Uses API integration",
		Metrics:     "Cost per shipment",
		Keywords:    []string{"carrier", "comparison"},
	}

	text := buildAssetText(asset)
	assert.Contains(t, text, "Asset: Carrier Comparison")
	assert.Contains(t, text, "Compares carriers")
	assert.Contains(t, text, "Cost savings")
	assert.Contains(t, text, "Better rates")
	assert.Contains(t, text, "Uses API integration")
	assert.Contains(t, text, "Cost per shipment")
	assert.Contains(t, text, "Keywords: carrier comparison")
}

func TestBuildAssetText_MinimalFields(t *testing.T) {
	asset := &assetdomain.Asset{
		Name: "Simple Asset",
	}

	text := buildAssetText(asset)
	assert.Equal(t, "Asset: Simple Asset", text)
}

func TestFilterRealAssets(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "Carrier Comparison"},
		{Name: "cap-asset-carrier-comparison"},
		{Name: "Payment Processing"},
		{Name: "Cap-Asset-Something"},
	}

	filtered := filterRealAssets(assets)
	require.Len(t, filtered, 2)
	assert.Equal(t, "Carrier Comparison", filtered[0].Name)
	assert.Equal(t, "Payment Processing", filtered[1].Name)
}

func TestEmbeddingAssetClassifier_NilTask(t *testing.T) {
	svc := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		return nil, nil
	}}
	store := newTestStore(t)
	classifier := NewEmbeddingAssetClassifier(svc, &mockAssetRepo{assets: nil}, store)

	_, err := classifier.ClassifyTaskAsset(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task cannot be nil")
}

func TestEmbeddingAssetClassifier_EmptyTasks(t *testing.T) {
	svc := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		return nil, nil
	}}
	store := newTestStore(t)
	classifier := NewEmbeddingAssetClassifier(svc, &mockAssetRepo{assets: nil}, store)

	results, err := classifier.ClassifyTasksAssets([]*taskdomain.Task{})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestEmbeddingAssetClassifier_NoAssets(t *testing.T) {
	svc := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		return nil, nil
	}}
	store := newTestStore(t)
	classifier := NewEmbeddingAssetClassifier(svc, &mockAssetRepo{assets: []*assetdomain.Asset{}}, store)

	task := &taskdomain.Task{Key: "COP-1", Summary: "Test task"}
	results, err := classifier.ClassifyTasksAssets([]*taskdomain.Task{task})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Nil(t, results[0].Asset)
	assert.Contains(t, results[0].Reason, "no assets available")
}

func TestEmbeddingAssetClassifier_MatchesBestAsset(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "Payment Processing", Description: "Handles payments"},
		{Name: "Carrier Comparison", Description: "Compares shipping carriers"},
		{Name: "User Authentication", Description: "Login and auth"},
	}

	// Simulate embeddings where task is closest to Carrier Comparison (index 1)
	assetVecs := [][]float64{
		{0.1, 0.9, 0.0}, // Payment
		{0.9, 0.1, 0.0}, // Carrier (closest to task)
		{0.0, 0.0, 1.0}, // Auth
	}
	taskVec := [][]float64{{0.8, 0.2, 0.0}} // Similar to Carrier

	callCount := 0
	svc := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		callCount++
		if callCount == 1 {
			// Asset embeddings
			return assetVecs, nil
		}
		// Task embeddings
		return taskVec, nil
	}}

	store := newTestStore(t)
	classifier := NewEmbeddingAssetClassifier(svc, &mockAssetRepo{assets: assets}, store)

	task := &taskdomain.Task{Key: "COP-1", Summary: "Update carrier comparison rates"}
	results, err := classifier.ClassifyTasksAssets([]*taskdomain.Task{task})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Carrier Comparison", results[0].Asset.Name)
	assert.Greater(t, results[0].Confidence, 0.0)
	assert.Contains(t, results[0].Reason, "embedding similarity")
}

func TestEmbeddingAssetClassifier_BatchClassification(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "Asset A"},
		{Name: "Asset B"},
	}

	assetVecs := [][]float64{
		{1.0, 0.0},
		{0.0, 1.0},
	}
	taskVecs := [][]float64{
		{0.9, 0.1}, // closer to Asset A
		{0.1, 0.9}, // closer to Asset B
	}

	callCount := 0
	svc := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		callCount++
		if callCount == 1 {
			return assetVecs, nil
		}
		return taskVecs, nil
	}}

	store := newTestStore(t)
	classifier := NewEmbeddingAssetClassifier(svc, &mockAssetRepo{assets: assets}, store)

	tasks := []*taskdomain.Task{
		{Key: "T-1", Summary: "Work on A"},
		{Key: "T-2", Summary: "Work on B"},
	}

	results, err := classifier.ClassifyTasksAssets(tasks)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "Asset A", results[0].Asset.Name)
	assert.Equal(t, "Asset B", results[1].Asset.Name)
}

func TestEmbeddingAssetClassifier_CachesAssetEmbeddings(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "Cached Asset", Description: "Testing cache"},
	}

	assetVec := [][]float64{{0.5, 0.5}}
	taskVec := [][]float64{{0.5, 0.5}}

	callCount := 0
	svc := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		callCount++
		if callCount == 1 {
			return assetVec, nil
		}
		return taskVec, nil
	}}

	store := newTestStore(t)
	classifier := NewEmbeddingAssetClassifier(svc, &mockAssetRepo{assets: assets}, store)

	task := &taskdomain.Task{Key: "T-1", Summary: "test"}

	// First call: computes asset embeddings
	_, err := classifier.ClassifyTasksAssets([]*taskdomain.Task{task})
	require.NoError(t, err)
	assert.Equal(t, 2, callCount) // 1 for assets + 1 for tasks

	// Second call: should reuse cached asset embeddings
	// Reset to a new classifier but same store
	callCount = 0
	svc2 := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		callCount++
		return taskVec, nil
	}}

	classifier2 := NewEmbeddingAssetClassifier(svc2, &mockAssetRepo{assets: assets}, store)
	_, err = classifier2.ClassifyTasksAssets([]*taskdomain.Task{task})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "should only embed tasks, not assets (cached)")
}

func TestEmbeddingAssetClassifier_EmbedServiceError(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "Test Asset"},
	}

	svc := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		return nil, fmt.Errorf("connection refused")
	}}

	store := newTestStore(t)
	classifier := NewEmbeddingAssetClassifier(svc, &mockAssetRepo{assets: assets}, store)

	task := &taskdomain.Task{Key: "T-1", Summary: "test"}
	_, err := classifier.ClassifyTasksAssets([]*taskdomain.Task{task})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compute asset embeddings")
}

func TestEmbeddingAssetClassifier_SingleTask(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "Only Asset"},
	}

	assetVec := [][]float64{{1.0, 0.0}}
	taskVec := [][]float64{{0.9, 0.1}}

	callCount := 0
	svc := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		callCount++
		if callCount == 1 {
			return assetVec, nil
		}
		return taskVec, nil
	}}

	store := newTestStore(t)
	classifier := NewEmbeddingAssetClassifier(svc, &mockAssetRepo{assets: assets}, store)

	task := &taskdomain.Task{Key: "T-1", Summary: "single task"}
	result, err := classifier.ClassifyTaskAsset(task)
	require.NoError(t, err)
	assert.Equal(t, "Only Asset", result.Asset.Name)
}

func TestFindBestMatch(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "A"},
		{Name: "B"},
		{Name: "C"},
	}

	vectors := [][]float64{
		{1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0},
		{0.0, 0.0, 1.0},
	}

	query := []float64{0.0, 0.9, 0.1}
	best, score := findBestMatch(query, assets, vectors)

	assert.Equal(t, "B", best.Name)
	assert.Greater(t, score, 0.0)
}

func TestFindBestMatch_EmptyVectors(t *testing.T) {
	assets := []*assetdomain.Asset{{Name: "A"}}
	vectors := [][]float64{{}}

	best, _ := findBestMatch([]float64{1.0}, assets, vectors)
	assert.Nil(t, best)
}

func TestBuildAssetTextWithHistory(t *testing.T) {
	asset := &assetdomain.Asset{
		Name:        "Carrier Comparison",
		Description: "Compares carriers",
		Keywords:    []string{"carrier"},
	}
	history := map[string][]string{
		"cap-asset-carrier-comparison": {"Fix carrier rates", "Update comparison UI"},
	}
	asset.ID = "cap-asset-carrier-comparison"

	text := buildAssetTextWithHistory(asset, history)
	assert.Contains(t, text, "Asset: Carrier Comparison")
	assert.Contains(t, text, "Compares carriers")
	assert.Contains(t, text, "Keywords: carrier")
	assert.Contains(t, text, "Related tasks: Fix carrier rates; Update comparison UI")
}

func TestBuildAssetTextWithHistory_NoHistory(t *testing.T) {
	asset := &assetdomain.Asset{
		Name:        "Payment Processing",
		Description: "Handles payments",
	}

	withNil := buildAssetTextWithHistory(asset, nil)
	withoutHistory := buildAssetText(asset)
	assert.Equal(t, withoutHistory, withNil, "nil history map should produce same result as buildAssetText")
}

func TestBuildTaskTextWithEpics(t *testing.T) {
	epicNames := map[string]string{
		"COP-2": "Carrier Rate Optimization",
	}

	task := &taskdomain.Task{
		Key:     "COP-10",
		Summary: "Fix rate comparison",
		Epic:    "COP-2",
		Project: "COP",
	}

	text := buildTaskTextWithEpics(task, epicNames)
	assert.Contains(t, text, "Task: Fix rate comparison")
	assert.Contains(t, text, "Epic: Carrier Rate Optimization", "epic key should be resolved to name")
	assert.NotContains(t, text, "Epic: COP-2")
}

func TestBuildTaskTextWithEpics_TaskKeyRefs(t *testing.T) {
	epicNames := map[string]string{
		"COP-39": "AB 10% Voucher test on SRP banner",
	}

	task := &taskdomain.Task{
		Key:     "COP-50",
		Summary: "Re-run COP-39 experiment with new config",
		Project: "COP",
	}

	text := buildTaskTextWithEpics(task, epicNames)
	assert.Contains(t, text, "Re-run AB 10% Voucher test on SRP banner experiment with new config",
		"task key references in summary should be resolved")
	assert.NotContains(t, text, "COP-39")
}

func TestBuildTaskTextWithEpics_NilMap(t *testing.T) {
	task := &taskdomain.Task{
		Key:     "COP-10",
		Summary: "Fix something",
		Epic:    "COP-2",
	}

	withNil := buildTaskTextWithEpics(task, nil)
	withoutEpics := buildTaskText(task)
	assert.Equal(t, withoutEpics, withNil, "nil epicNames map should produce same result as buildTaskText")
}

func TestFilterRealAssets_Empty(t *testing.T) {
	filtered := filterRealAssets([]*assetdomain.Asset{})
	assert.Empty(t, filtered)
	assert.NotNil(t, filtered, "should return empty slice, not nil")
}

func TestFindBestMatchWithEpicBoost(t *testing.T) {
	// Two assets with very close scores — epic boost should tip the winner
	assets := []*assetdomain.Asset{
		{ID: "asset-a", Name: "Asset A"},
		{ID: "asset-b", Name: "Asset B"},
	}
	assetVecs := [][]float64{
		{0.80, 0.20}, // Asset A
		{0.78, 0.22}, // Asset B — slightly lower similarity
	}
	query := []float64{0.80, 0.20} // closest to Asset A without boost

	epicAssetHint := map[string]string{
		"COP-5": "asset-b", // hint says this epic belongs to Asset B
	}

	svc := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		return nil, nil
	}}
	store := newTestStore(t)
	c := &EmbeddingAssetClassifier{
		embeddingService: svc,
		store:            store,
		epicAssetHint:    epicAssetHint,
	}

	task := &taskdomain.Task{Key: "COP-10", Epic: "COP-5"}
	best, _ := c.findBestMatchWithEpicBoost(query, assets, assetVecs, task)
	assert.Equal(t, "Asset B", best.Name, "epic boost should change winner when scores are close")
}

func TestFindBestMatchWithEpicBoost_NoHint(t *testing.T) {
	assets := []*assetdomain.Asset{
		{ID: "asset-a", Name: "Asset A"},
		{ID: "asset-b", Name: "Asset B"},
	}
	assetVecs := [][]float64{
		{0.9, 0.1},
		{0.1, 0.9},
	}
	query := []float64{0.9, 0.1} // closest to Asset A

	svc := &mockEmbeddingService{embedFn: func(texts []string) ([][]float64, error) {
		return nil, nil
	}}
	store := newTestStore(t)
	c := &EmbeddingAssetClassifier{
		embeddingService: svc,
		store:            store,
		epicAssetHint:    nil, // no hints
	}

	task := &taskdomain.Task{Key: "COP-10", Epic: "COP-5"}
	best, _ := c.findBestMatchWithEpicBoost(query, assets, assetVecs, task)
	assert.Equal(t, "Asset A", best.Name, "without epic hint, should fall back to pure cosine similarity")
}
