package ports

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// AssetClassificationResult represents the result of asset classification
type AssetClassificationResult struct {
	Task       *taskdomain.Task `json:"task"`
	Asset      *domain.Asset    `json:"asset,omitempty"`
	Confidence float64          `json:"confidence"`
	Reason     string           `json:"reason"`
}

// AssetClassifier defines the interface for classifying tasks by related asset
type AssetClassifier interface {
	// ClassifyTaskAsset determines which asset a task belongs to based on content analysis
	ClassifyTaskAsset(task *taskdomain.Task) (*AssetClassificationResult, error)

	// ClassifyTasksAssets determines the related asset for multiple tasks
	ClassifyTasksAssets(tasks []*taskdomain.Task) ([]*AssetClassificationResult, error)
}
