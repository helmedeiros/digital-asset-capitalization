package ports

import (
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// ComprehensiveClassificationResult represents the complete classification result including both asset and work type
type ComprehensiveClassificationResult struct {
	Task           *taskdomain.Task           `json:"task"`
	Asset          *AssetClassificationResult `json:"asset_classification"`
	LLMAsset       *AssetClassificationResult `json:"llm_asset_classification,omitempty"`
	WorkType       taskdomain.WorkType        `json:"work_type"`
	WorkTypeReason string                     `json:"work_type_reason"`
}

// ClassificationChain defines the interface for orchestrating multiple classifiers
type ClassificationChain interface {
	// ClassifyTask performs comprehensive classification including asset assignment and work type
	ClassifyTask(task *taskdomain.Task) (*ComprehensiveClassificationResult, error)

	// ClassifyTasks performs comprehensive classification for multiple tasks
	ClassifyTasks(tasks []*taskdomain.Task) ([]*ComprehensiveClassificationResult, error)
}
