package classifier

import (
	"testing"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// mockAssetRepo is a mock implementation of AssetRepository for testing
type mockAssetRepo struct {
	assets []*assetdomain.Asset
}

func (m *mockAssetRepo) FindAll() ([]*assetdomain.Asset, error) {
	return m.assets, nil
}

func (m *mockAssetRepo) FindByName(name string) (*assetdomain.Asset, error) {
	for _, asset := range m.assets {
		if asset.Name == name {
			return asset, nil
		}
	}
	return nil, nil
}

func (m *mockAssetRepo) FindByID(id string) (*assetdomain.Asset, error) {
	for _, asset := range m.assets {
		if asset.ID == id {
			return asset, nil
		}
	}
	return nil, nil
}

func (m *mockAssetRepo) Save(asset *assetdomain.Asset) error {
	m.assets = append(m.assets, asset)
	return nil
}

func (m *mockAssetRepo) Delete(name string) error {
	for i, asset := range m.assets {
		if asset.Name == name {
			m.assets = append(m.assets[:i], m.assets[i+1:]...)
			return nil
		}
	}
	return nil
}

func TestNewAssetClassifier(t *testing.T) {
	repo := &mockAssetRepo{}
	classifier := NewAssetClassifier(repo)

	if classifier == nil {
		t.Error("Expected classifier to be created, got nil")
		return
	}
	if classifier.assetRepo == nil {
		t.Error("Expected classifier to have a repository")
	}
}

func TestAssetClassifier_ClassifyTaskAsset(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "pricing", Keywords: []string{"price", "cost", "fare"}},
		{Name: "booking", Keywords: []string{"reservation", "ticket", "purchase"}},
		{Name: "search", Keywords: []string{"find", "filter", "query"}},
	}

	repo := &mockAssetRepo{assets: assets}
	classifier := NewAssetClassifier(repo)

	tests := []struct {
		name     string
		task     *taskdomain.Task
		expected string
	}{
		{
			name: "Match by asset name in summary",
			task: &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     "Fix pricing calculation bug",
				Description: "The pricing module needs fixing",
			},
			expected: "cap-asset-pricing",
		},
		{
			name: "Match by keyword in description",
			task: &taskdomain.Task{
				Key:         "TEST-2",
				Summary:     "Update system",
				Description: "Need to update the fare calculation logic",
			},
			expected: "cap-asset-pricing",
		},
		{
			name: "Match by asset name in labels",
			task: &taskdomain.Task{
				Key:         "TEST-3",
				Summary:     "Generic task",
				Description: "Generic description",
				Labels:      []string{"booking-system", "frontend"},
			},
			expected: "cap-asset-booking",
		},
		{
			name: "Match by keyword in epic",
			task: &taskdomain.Task{
				Key:         "TEST-4",
				Summary:     "Task summary",
				Description: "Task description",
				Epic:        "Improve search functionality",
			},
			expected: "cap-asset-search",
		},
		{
			name: "No match - return default",
			task: &taskdomain.Task{
				Key:         "TEST-5",
				Summary:     "Random task",
				Description: "Random description",
			},
			expected: DefaultAssetLabel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := classifier.ClassifyTaskAsset(tt.task)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAssetClassifier_ClassifyTasksAssets(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "pricing", Keywords: []string{"price", "cost"}},
		{Name: "booking", Keywords: []string{"reservation"}},
	}

	repo := &mockAssetRepo{assets: assets}
	classifier := NewAssetClassifier(repo)

	tasks := []*taskdomain.Task{
		{
			Key:         "TEST-1",
			Summary:     "Fix pricing issue",
			Description: "The pricing module needs fixing",
		},
		{
			Key:         "TEST-2",
			Summary:     "Update booking flow",
			Description: "Need to update the booking process",
		},
		{
			Key:         "TEST-3",
			Summary:     "Generic task",
			Description: "Generic description",
		},
	}

	result, err := classifier.ClassifyTasksAssets(tasks)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expected := map[string]string{
		"TEST-1": "cap-asset-pricing",
		"TEST-2": "cap-asset-booking",
		"TEST-3": DefaultAssetLabel,
	}

	for taskKey, expectedAsset := range expected {
		if result[taskKey] != expectedAsset {
			t.Errorf("For task %s, expected %s, got %s", taskKey, expectedAsset, result[taskKey])
		}
	}
}

func TestAssetClassifier_ClassifyByContent(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "pricing", Keywords: []string{"price", "cost", "fare"}},
		{Name: "booking", Keywords: []string{"reservation", "ticket"}},
	}

	repo := &mockAssetRepo{assets: assets}
	classifier := NewAssetClassifier(repo)

	tests := []struct {
		name     string
		task     *taskdomain.Task
		expected string
	}{
		{
			name: "Match by asset name",
			task: &taskdomain.Task{
				Summary:     "Fix pricing calculation",
				Description: "The pricing module needs fixing",
			},
			expected: "cap-asset-pricing",
		},
		{
			name: "Match by keyword",
			task: &taskdomain.Task{
				Summary:     "Update system",
				Description: "Need to update the fare calculation logic",
			},
			expected: "cap-asset-pricing",
		},
		{
			name: "No match",
			task: &taskdomain.Task{
				Summary:     "Random task",
				Description: "Random description",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.classifyByContent(tt.task, assets)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAssetClassifier_ClassifyByMetadata(t *testing.T) {
	assets := []*assetdomain.Asset{
		{Name: "pricing", Keywords: []string{"price", "cost"}},
		{Name: "booking", Keywords: []string{"reservation"}},
	}

	repo := &mockAssetRepo{assets: assets}
	classifier := NewAssetClassifier(repo)

	tests := []struct {
		name     string
		task     *taskdomain.Task
		expected string
	}{
		{
			name: "Match by asset name in labels",
			task: &taskdomain.Task{
				Labels: []string{"pricing-system", "backend"},
			},
			expected: "cap-asset-pricing",
		},
		{
			name: "Match by keyword in labels",
			task: &taskdomain.Task{
				Labels: []string{"price-calculation", "frontend"},
			},
			expected: "cap-asset-pricing",
		},
		{
			name: "Match by asset name in epic",
			task: &taskdomain.Task{
				Epic: "Improve booking functionality",
			},
			expected: "cap-asset-booking",
		},
		{
			name: "Match by keyword in epic",
			task: &taskdomain.Task{
				Epic: "Update reservation system",
			},
			expected: "cap-asset-booking",
		},
		{
			name: "No match",
			task: &taskdomain.Task{
				Labels: []string{"random-label"},
				Epic:   "Random epic",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.classifyByMetadata(tt.task, assets)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
