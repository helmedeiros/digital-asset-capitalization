package classifier

import (
	"fmt"

	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// ComprehensiveClassificationChain orchestrates multiple classifiers for comprehensive task classification
type ComprehensiveClassificationChain struct {
	assetClassifier    ports.AssetClassifier
	workTypeClassifier ports.TaskClassifier
}

// NewComprehensiveClassificationChain creates a new comprehensive classification chain
func NewComprehensiveClassificationChain(assetClassifier ports.AssetClassifier, workTypeClassifier ports.TaskClassifier) ports.ClassificationChain {
	return &ComprehensiveClassificationChain{
		assetClassifier:    assetClassifier,
		workTypeClassifier: workTypeClassifier,
	}
}

// ClassifyTask performs comprehensive classification including asset assignment and work type
func (c *ComprehensiveClassificationChain) ClassifyTask(task *taskdomain.Task) (*ports.ComprehensiveClassificationResult, error) {
	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}

	// Step 1: Classify asset
	assetResult, err := c.assetClassifier.ClassifyTaskAsset(task)
	if err != nil {
		return nil, fmt.Errorf("asset classification failed: %w", err)
	}

	// Step 2: Classify work type
	workType, err := c.workTypeClassifier.ClassifyTask(task)
	if err != nil {
		return nil, fmt.Errorf("work type classification failed: %w", err)
	}

	// Step 3: Determine work type reason
	workTypeReason := c.generateWorkTypeReason(task, assetResult, workType)

	// Create comprehensive result
	result := &ports.ComprehensiveClassificationResult{
		Task:           task,
		Asset:          assetResult,
		WorkType:       workType,
		WorkTypeReason: workTypeReason,
	}

	return result, nil
}

// ClassifyTasks performs comprehensive classification for multiple tasks
func (c *ComprehensiveClassificationChain) ClassifyTasks(tasks []*taskdomain.Task) ([]*ports.ComprehensiveClassificationResult, error) {
	if len(tasks) == 0 {
		return []*ports.ComprehensiveClassificationResult{}, nil
	}

	// Step 1: Batch classify assets
	assetResults, err := c.assetClassifier.ClassifyTasksAssets(tasks)
	if err != nil {
		return nil, fmt.Errorf("batch asset classification failed: %w", err)
	}

	// Step 2: Batch classify work types
	workTypeResults, err := c.workTypeClassifier.ClassifyTasks(tasks)
	if err != nil {
		return nil, fmt.Errorf("batch work type classification failed: %w", err)
	}

	// Step 3: Combine results
	results := make([]*ports.ComprehensiveClassificationResult, 0, len(tasks))
	for i, task := range tasks {
		var assetResult *ports.AssetClassificationResult
		if i < len(assetResults) {
			assetResult = assetResults[i]
		}

		workType, exists := workTypeResults[task.Key]
		if !exists {
			workType = taskdomain.WorkTypeDevelopment // default fallback
		}

		workTypeReason := c.generateWorkTypeReason(task, assetResult, workType)

		result := &ports.ComprehensiveClassificationResult{
			Task:           task,
			Asset:          assetResult,
			WorkType:       workType,
			WorkTypeReason: workTypeReason,
		}

		results = append(results, result)
	}

	return results, nil
}

// generateWorkTypeReason generates a human-readable reason for the work type classification
func (c *ComprehensiveClassificationChain) generateWorkTypeReason(task *taskdomain.Task, assetResult *ports.AssetClassificationResult, workType taskdomain.WorkType) string {
	switch workType {
	case taskdomain.WorkTypeDiscovery:
		if c.containsResearchKeywords(task) {
			return "spike/research task detected"
		}
		return "discovery work classification"

	case taskdomain.WorkTypeMaintenance:
		if c.containsBugKeywords(task) {
			return "bug fix or maintenance task"
		}
		if assetResult != nil && assetResult.Asset != nil {
			return fmt.Sprintf("maintenance work for %s", assetResult.Asset.Name)
		}
		return "maintenance work classification"

	case taskdomain.WorkTypeDevelopment:
		if c.containsAPIKeywords(task) {
			return "new API or feature development"
		}
		if assetResult != nil && assetResult.Asset != nil {
			return fmt.Sprintf("development work for %s", assetResult.Asset.Name)
		}
		return "development work classification"

	default:
		return "default work type classification"
	}
}

// containsResearchKeywords checks if task contains research-related keywords
func (c *ComprehensiveClassificationChain) containsResearchKeywords(task *taskdomain.Task) bool {
	content := task.Summary + " " + task.Description
	keywords := []string{"spike", "research", "discovery", "investigation", "poc", "proof-of-concept"}

	for _, keyword := range keywords {
		if contains(content, keyword) {
			return true
		}
	}

	for _, label := range task.Labels {
		for _, keyword := range keywords {
			if contains(label, keyword) {
				return true
			}
		}
	}

	return false
}

// containsBugKeywords checks if task contains bug-related keywords
func (c *ComprehensiveClassificationChain) containsBugKeywords(task *taskdomain.Task) bool {
	content := task.Summary + " " + task.Description
	keywords := []string{"bug", "fix", "hotfix", "error", "issue", "defect"}

	for _, keyword := range keywords {
		if contains(content, keyword) {
			return true
		}
	}

	return task.Type == taskdomain.TaskTypeBug
}

// containsAPIKeywords checks if task contains API-related keywords
func (c *ComprehensiveClassificationChain) containsAPIKeywords(task *taskdomain.Task) bool {
	content := task.Summary + " " + task.Description
	keywords := []string{"api", "endpoint", "service", "new", "add", "create", "implement"}

	for _, keyword := range keywords {
		if contains(content, keyword) {
			return true
		}
	}

	return false
}

// contains performs case-insensitive substring matching
func contains(content, keyword string) bool {
	contentLower := ""
	keywordLower := ""

	for _, r := range content {
		if r >= 'A' && r <= 'Z' {
			contentLower += string(r + 32)
		} else {
			contentLower += string(r)
		}
	}

	for _, r := range keyword {
		if r >= 'A' && r <= 'Z' {
			keywordLower += string(r + 32)
		} else {
			keywordLower += string(r)
		}
	}

	// Simple substring search
	if len(keywordLower) > len(contentLower) {
		return false
	}

	for i := 0; i <= len(contentLower)-len(keywordLower); i++ {
		match := true
		for j := 0; j < len(keywordLower); j++ {
			if contentLower[i+j] != keywordLower[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}
