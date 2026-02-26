package classifier

import (
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// ComprehensiveClassifierAdapter adapts the ComprehensiveClassificationChain to the TaskClassifier interface
// This allows us to integrate the new comprehensive classification system with existing code
// that expects the simpler TaskClassifier interface
type ComprehensiveClassifierAdapter struct {
	chain ports.ClassificationChain
}

// NewComprehensiveClassifierAdapter creates a new adapter for the comprehensive classification chain
func NewComprehensiveClassifierAdapter(chain ports.ClassificationChain) ports.ComprehensiveTaskClassifier {
	return &ComprehensiveClassifierAdapter{
		chain: chain,
	}
}

// ClassifyTask implements TaskClassifier.ClassifyTask by delegating to the comprehensive chain
// and extracting just the work type from the comprehensive result
func (a *ComprehensiveClassifierAdapter) ClassifyTask(task *taskdomain.Task) (taskdomain.WorkType, error) {
	result, err := a.chain.ClassifyTask(task)
	if err != nil {
		return "", err
	}

	return result.WorkType, nil
}

// ClassifyTasks implements TaskClassifier.ClassifyTasks by delegating to the comprehensive chain
// and extracting just the work types from the comprehensive results
func (a *ComprehensiveClassifierAdapter) ClassifyTasks(tasks []*taskdomain.Task) (map[string]taskdomain.WorkType, error) {
	results, err := a.chain.ClassifyTasks(tasks)
	if err != nil {
		return nil, err
	}

	workTypes := make(map[string]taskdomain.WorkType)
	for _, result := range results {
		workTypes[result.Task.Key] = result.WorkType
	}

	return workTypes, nil
}

// ClassifyTasksComprehensive implements ComprehensiveTaskClassifier.ClassifyTasksComprehensive
// by directly delegating to the comprehensive chain to return full classification results
func (a *ComprehensiveClassifierAdapter) ClassifyTasksComprehensive(tasks []*taskdomain.Task) ([]*ports.ComprehensiveClassificationResult, error) {
	return a.chain.ClassifyTasks(tasks)
}

// SetLLMEnabled forwards the LLM toggle to the underlying classification chain
func (a *ComprehensiveClassifierAdapter) SetLLMEnabled(enabled bool) {
	if chain, ok := a.chain.(*ComprehensiveClassificationChainWithInheritance); ok {
		chain.SetLLMEnabled(enabled)
	}
}
