package classifier

import (
	"fmt"
	"log"
	"strings"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
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

// ComprehensiveClassificationChainWithInheritance orchestrates multiple classifiers with subtask inheritance support
type ComprehensiveClassificationChainWithInheritance struct {
	assetClassifier    ports.AssetClassifier
	workTypeClassifier ports.TaskClassifier
	llmClassifier      ports.AssetClassifier
	llmEnabled         bool
	taskLookup         map[string]*taskdomain.Task // For parent task lookup
}

// NewComprehensiveClassificationChainWithInheritance creates a new comprehensive classification chain with inheritance support
func NewComprehensiveClassificationChainWithInheritance(assetClassifier ports.AssetClassifier, workTypeClassifier ports.TaskClassifier) ports.ClassificationChain {
	return &ComprehensiveClassificationChainWithInheritance{
		assetClassifier:    assetClassifier,
		workTypeClassifier: workTypeClassifier,
		taskLookup:         make(map[string]*taskdomain.Task),
	}
}

// NewComprehensiveClassificationChainWithLLM creates a classification chain with an optional LLM classifier
func NewComprehensiveClassificationChainWithLLM(assetClassifier ports.AssetClassifier, workTypeClassifier ports.TaskClassifier, llmClassifier ports.AssetClassifier) ports.ClassificationChain {
	return &ComprehensiveClassificationChainWithInheritance{
		assetClassifier:    assetClassifier,
		workTypeClassifier: workTypeClassifier,
		llmClassifier:      llmClassifier,
		taskLookup:         make(map[string]*taskdomain.Task),
	}
}

// SetLLMEnabled enables or disables the LLM classifier for comparison mode
func (c *ComprehensiveClassificationChainWithInheritance) SetLLMEnabled(enabled bool) {
	c.llmEnabled = enabled
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

// ClassifyTask for the inheritance version performs comprehensive classification with subtask inheritance
func (c *ComprehensiveClassificationChainWithInheritance) ClassifyTask(task *taskdomain.Task) (*ports.ComprehensiveClassificationResult, error) {
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

	// Step 3: Check if we need inheritance for subtasks or discovery tasks
	isDiscoveryTask := workType == taskdomain.WorkTypeDiscovery || c.containsResearchKeywords(task)
	if (task.Type == taskdomain.TaskTypeSubtask || isDiscoveryTask) && c.needsInheritance(task, assetResult, workType) {
		inheritedAsset, inheritedWorkType, inheritedReason := c.inheritFromParent(task)
		if inheritedAsset != nil {
			assetResult = inheritedAsset
		}

		// Discovery tasks should inherit asset but keep their own work type (cap-discovery)
		// Subtasks should inherit both asset AND work type from parent
		shouldInheritWorkType := true
		if isDiscoveryTask && c.hasExplicitWorkTypeLabel(task) {
			// Discovery tasks with explicit cap-discovery label keep their work type
			shouldInheritWorkType = false
		}

		if inheritedWorkType != "" && shouldInheritWorkType {
			workType = inheritedWorkType
		}

		workTypeReason := inheritedReason
		if workTypeReason == "" {
			workTypeReason = c.generateWorkTypeReason(task, assetResult, workType)
		}

		result := &ports.ComprehensiveClassificationResult{
			Task:           task,
			Asset:          assetResult,
			WorkType:       workType,
			WorkTypeReason: workTypeReason,
		}
		c.runLLMClassification(task, result)
		return result, nil
	}

	// Step 4: Determine work type reason
	workTypeReason := c.generateWorkTypeReason(task, assetResult, workType)

	// Create comprehensive result
	result := &ports.ComprehensiveClassificationResult{
		Task:           task,
		Asset:          assetResult,
		WorkType:       workType,
		WorkTypeReason: workTypeReason,
	}
	c.runLLMClassification(task, result)

	return result, nil
}

// runLLMClassification runs the LLM classifier if enabled and attaches the result
func (c *ComprehensiveClassificationChainWithInheritance) runLLMClassification(task *taskdomain.Task, result *ports.ComprehensiveClassificationResult) {
	if !c.llmEnabled || c.llmClassifier == nil {
		return
	}
	llmResult, err := c.llmClassifier.ClassifyTaskAsset(task)
	if err != nil {
		log.Printf("Warning: LLM classification failed for %s: %v", task.Key, err)
		return
	}
	result.LLMAsset = llmResult
}

// hasExplicitWorkTypeLabel checks if the task has an explicit work type label
func (c *ComprehensiveClassificationChainWithInheritance) hasExplicitWorkTypeLabel(task *taskdomain.Task) bool {
	for _, label := range task.Labels {
		switch label {
		case "cap-development", "cap-discovery", "cap-maintenance":
			return true
		}
	}
	return false
}

// needsInheritance determines if a task needs to inherit classification from its parent
// This applies to subtasks and discovery tasks (spikes, research) that may lack explicit asset mentions
func (c *ComprehensiveClassificationChainWithInheritance) needsInheritance(task *taskdomain.Task, assetResult *ports.AssetClassificationResult, workType taskdomain.WorkType) bool {
	// FIRST: Never inherit if we have a strong natural classification that overrides existing labels
	if assetResult != nil && assetResult.Confidence >= 0.9 && strings.Contains(assetResult.Reason, "overrides existing label") {
		return false
	}

	// SECOND: Never inherit if we have preserved existing asset labels with good confidence
	if assetResult != nil && assetResult.Reason == "existing asset label preserved" && assetResult.Confidence >= 0.85 {
		return false
	}

	// THIRD: Never inherit if we detected a strong primary subject
	if assetResult != nil && assetResult.Reason == "detected as primary subject based on title emphasis" {
		return false
	}

	// If no asset was found or confidence is not strong enough, we need inheritance
	// Subtasks with weak keyword/partial matches (< 0.8) should inherit from parent
	// rather than keeping a potentially incorrect low-confidence match
	hasWeakAssetClassification := assetResult == nil || assetResult.Asset == nil || assetResult.Confidence < 0.8

	// Discovery/spike tasks often lack explicit asset mentions but should inherit from epic
	isDiscoveryTask := workType == taskdomain.WorkTypeDiscovery || c.containsResearchKeywords(task)

	// Subtasks OR discovery tasks can inherit
	isInheritableType := task.Type == taskdomain.TaskTypeSubtask || isDiscoveryTask

	return isInheritableType && hasWeakAssetClassification
}

// inheritFromParent attempts to inherit classification from parent task or epic
func (c *ComprehensiveClassificationChainWithInheritance) inheritFromParent(task *taskdomain.Task) (*ports.AssetClassificationResult, taskdomain.WorkType, string) {
	if task.Epic == "" {
		return nil, "", ""
	}

	// Look up parent task
	parentTask, exists := c.taskLookup[task.Epic]
	if !exists {
		return nil, "", ""
	}

	// First, try to inherit from parent's existing labels (already classified assets)
	var inheritedAsset *ports.AssetClassificationResult
	var inheritedWorkType taskdomain.WorkType

	// Check parent's labels for asset assignments
	for _, label := range parentTask.Labels {
		if strings.HasPrefix(label, "cap-asset-") {
			// Extract asset name from label (remove "cap-asset-" prefix)
			assetIdentifier := strings.TrimPrefix(label, "cap-asset-")

			// Create asset from the label identifier
			assetName := c.formatAssetName(assetIdentifier)

			inheritedAsset = &ports.AssetClassificationResult{
				Task: task,
				Asset: &assetdomain.Asset{
					Name:     assetName,
					Keywords: []string{assetIdentifier},
				},
				Confidence: 0.85, // High confidence for inherited labels
				Reason:     fmt.Sprintf("inherited asset label '%s' from parent task %s", assetIdentifier, parentTask.Key),
			}
			break
		}
	}

	// Check parent's labels for work type
	for _, label := range parentTask.Labels {
		switch label {
		case "cap-maintenance":
			inheritedWorkType = taskdomain.WorkTypeMaintenance
		case "cap-discovery":
			inheritedWorkType = taskdomain.WorkTypeDiscovery
		case "cap-development":
			inheritedWorkType = taskdomain.WorkTypeDevelopment
		}
		if inheritedWorkType != "" {
			break
		}
	}

	// If we didn't find explicit labels, classify the parent directly (but avoid recursion)
	if inheritedAsset == nil || inheritedWorkType == "" {
		// Directly classify parent without inheritance to avoid recursion
		parentAssetResult, err := c.assetClassifier.ClassifyTaskAsset(parentTask)
		if err == nil && parentAssetResult != nil && parentAssetResult.Asset != nil && inheritedAsset == nil {
			inheritedAsset = &ports.AssetClassificationResult{
				Task:       task,
				Asset:      parentAssetResult.Asset,
				Confidence: parentAssetResult.Confidence * 0.8, // Slightly lower confidence for inherited
				Reason:     fmt.Sprintf("inherited from parent task %s", parentTask.Key),
			}
		}

		if inheritedWorkType == "" {
			parentWorkType, err := c.workTypeClassifier.ClassifyTask(parentTask)
			if err == nil {
				inheritedWorkType = parentWorkType
			}
		}
	}

	inheritedReason := fmt.Sprintf("inherited from parent task %s", parentTask.Key)

	return inheritedAsset, inheritedWorkType, inheritedReason
}

// generateWorkTypeReason generates a human-readable reason for the work type classification (inheritance version)
func (c *ComprehensiveClassificationChainWithInheritance) generateWorkTypeReason(task *taskdomain.Task, assetResult *ports.AssetClassificationResult, workType taskdomain.WorkType) string {
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

// containsResearchKeywords checks if task contains research-related keywords (inheritance version)
func (c *ComprehensiveClassificationChainWithInheritance) containsResearchKeywords(task *taskdomain.Task) bool {
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

// containsBugKeywords checks if task contains bug-related keywords (inheritance version)
func (c *ComprehensiveClassificationChainWithInheritance) containsBugKeywords(task *taskdomain.Task) bool {
	content := task.Summary + " " + task.Description
	keywords := []string{"bug", "fix", "hotfix", "error", "issue", "defect"}

	for _, keyword := range keywords {
		if contains(content, keyword) {
			return true
		}
	}

	return task.Type == taskdomain.TaskTypeBug
}

// containsAPIKeywords checks if task contains API-related keywords (inheritance version)
func (c *ComprehensiveClassificationChainWithInheritance) containsAPIKeywords(task *taskdomain.Task) bool {
	content := task.Summary + " " + task.Description
	keywords := []string{"api", "endpoint", "service", "new", "add", "create", "implement"}

	for _, keyword := range keywords {
		if contains(content, keyword) {
			return true
		}
	}

	return false
}

// formatAssetName converts an asset identifier like "cabin-markup" to a proper asset name like "Cabin Markup"
func (c *ComprehensiveClassificationChainWithInheritance) formatAssetName(identifier string) string {
	// Split on hyphens and capitalize each part
	parts := strings.Split(identifier, "-")
	for i, part := range parts {
		if len(part) > 0 {
			// Simple capitalization: first letter uppercase, rest lowercase
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
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

// ClassifyTasks for the inheritance version performs comprehensive classification with subtask inheritance
func (c *ComprehensiveClassificationChainWithInheritance) ClassifyTasks(tasks []*taskdomain.Task) ([]*ports.ComprehensiveClassificationResult, error) {
	if len(tasks) == 0 {
		return []*ports.ComprehensiveClassificationResult{}, nil
	}

	// Step 1: Build task lookup map for parent-child relationships
	c.taskLookup = make(map[string]*taskdomain.Task)
	for _, task := range tasks {
		c.taskLookup[task.Key] = task
	}

	// Step 2: Classify all tasks (this will handle inheritance internally)
	results := make([]*ports.ComprehensiveClassificationResult, 0, len(tasks))

	// First pass: classify non-subtasks to establish parent classifications
	nonSubtasks := make([]*taskdomain.Task, 0)
	subtasks := make([]*taskdomain.Task, 0)

	for _, task := range tasks {
		if task.Type == taskdomain.TaskTypeSubtask {
			subtasks = append(subtasks, task)
		} else {
			nonSubtasks = append(nonSubtasks, task)
		}
	}

	// Classify non-subtasks first
	for _, task := range nonSubtasks {
		result, err := c.ClassifyTask(task)
		if err != nil {
			return nil, fmt.Errorf("failed to classify task %s: %w", task.Key, err)
		}
		results = append(results, result)
	}

	// Then classify subtasks (which can inherit from the already classified parents)
	for _, task := range subtasks {
		result, err := c.ClassifyTask(task)
		if err != nil {
			return nil, fmt.Errorf("failed to classify task %s: %w", task.Key, err)
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
