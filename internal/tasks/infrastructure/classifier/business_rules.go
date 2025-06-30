package classifier

import (
	"strings"
	"time"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	assetports "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// BusinessRulesClassifier implements TaskClassifier using business rules from the spec
type BusinessRulesClassifier struct {
	assetRepo assetports.AssetRepository
}

// NewBusinessRulesClassifier creates a new business rules classifier
func NewBusinessRulesClassifier(assetRepo assetports.AssetRepository) *BusinessRulesClassifier {
	return &BusinessRulesClassifier{
		assetRepo: assetRepo,
	}
}

// ClassifyTask determines the work type of a task based on business rules
func (c *BusinessRulesClassifier) ClassifyTask(task *taskdomain.Task) (taskdomain.WorkType, error) {
	// Rule 1: If labeled as a spike or research → cap-discovery
	if c.isSpikeOrResearch(task) {
		return taskdomain.WorkTypeDiscovery, nil
	}

	// Rule 3: If adds new API/inventory → cap-development (check before asset lookup)
	if c.addsNewAPIOrInventory(task) {
		return taskdomain.WorkTypeDevelopment, nil
	}

	// Get associated asset for time-based rules
	asset, err := c.findRelatedAsset(task)
	if err != nil {
		// If we can't find asset, fall back to content-based classification
		return c.classifyByContent(task), nil
	}

	// Rule 2: If within 6 months or pre-100% rollout → cap-development
	if c.isWithinDevelopmentPeriod(asset) {
		return taskdomain.WorkTypeDevelopment, nil
	}

	// Rule 4: Otherwise, if bug/fix past rollout → cap-maintenance
	if c.isBugOrFixPastRollout(task, asset) {
		return taskdomain.WorkTypeMaintenance, nil
	}

	// Default fallback
	return c.classifyByContent(task), nil
}

// ClassifyTasks determines the work type for multiple tasks
func (c *BusinessRulesClassifier) ClassifyTasks(tasks []*taskdomain.Task) (map[string]taskdomain.WorkType, error) {
	result := make(map[string]taskdomain.WorkType)

	for _, task := range tasks {
		workType, err := c.ClassifyTask(task)
		if err != nil {
			return nil, err
		}
		result[task.Key] = workType
	}

	return result, nil
}

// isSpikeOrResearch checks if task is labeled as spike or research
func (c *BusinessRulesClassifier) isSpikeOrResearch(task *taskdomain.Task) bool {
	content := strings.ToLower(task.Summary + " " + task.Description)

	// Check labels
	for _, label := range task.Labels {
		labelLower := strings.ToLower(label)
		if strings.Contains(labelLower, "spike") ||
			strings.Contains(labelLower, "research") ||
			strings.Contains(labelLower, "discovery") ||
			strings.Contains(labelLower, "investigation") ||
			strings.Contains(labelLower, "poc") ||
			strings.Contains(labelLower, "proof-of-concept") {
			return true
		}
	}

	// Check content for spike/research keywords
	if strings.Contains(content, "spike") ||
		strings.Contains(content, "research") ||
		strings.Contains(content, "investigate") ||
		strings.Contains(content, "discovery") ||
		strings.Contains(content, "proof of concept") ||
		strings.Contains(content, "poc") {
		return true
	}

	return false
}

// findRelatedAsset attempts to find the asset related to this task
func (c *BusinessRulesClassifier) findRelatedAsset(task *taskdomain.Task) (*assetdomain.Asset, error) {
	// Get all assets to check against
	assets, err := c.assetRepo.FindAll()
	if err != nil {
		return nil, err
	}

	// Try to match by keywords first
	for _, asset := range assets {
		if c.taskMatchesAsset(task, asset) {
			return asset, nil
		}
	}

	// If no specific asset found, return nil (no error, just no asset)
	return nil, nil
}

// taskMatchesAsset checks if a task is related to an asset
func (c *BusinessRulesClassifier) taskMatchesAsset(task *taskdomain.Task, asset *assetdomain.Asset) bool {
	if asset == nil {
		return false
	}

	content := strings.ToLower(task.Summary + " " + task.Description)

	// Check asset name
	if strings.Contains(content, strings.ToLower(asset.Name)) {
		return true
	}

	// Check asset keywords
	for _, keyword := range asset.Keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			return true
		}
	}

	// Check task labels for asset references
	for _, label := range task.Labels {
		if strings.Contains(strings.ToLower(label), strings.ToLower(asset.Name)) {
			return true
		}
		for _, keyword := range asset.Keywords {
			if strings.Contains(strings.ToLower(label), strings.ToLower(keyword)) {
				return true
			}
		}
	}

	return false
}

// isWithinDevelopmentPeriod checks if asset is within 6 months of launch or not 100% rolled out
func (c *BusinessRulesClassifier) isWithinDevelopmentPeriod(asset *assetdomain.Asset) bool {
	if asset == nil {
		return false
	}

	// If not 100% rolled out, it's still in development
	if !asset.IsRolledOut100 {
		return true
	}

	// If launch date is zero, assume it's still in development
	if asset.LaunchDate.IsZero() {
		return true
	}

	// Check if within 6 months of launch date
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	return asset.LaunchDate.After(sixMonthsAgo)
}

// addsNewAPIOrInventory checks if task adds new API or inventory
func (c *BusinessRulesClassifier) addsNewAPIOrInventory(task *taskdomain.Task) bool {
	content := strings.ToLower(task.Summary + " " + task.Description)

	// API keywords
	apiKeywords := []string{
		"api", "endpoint", "rest", "graphql", "microservice",
		"service", "integration", "webhook", "sdk",
	}

	// Inventory keywords
	inventoryKeywords := []string{
		"inventory", "product", "catalog", "stock",
		"item", "sku", "variant", "listing", "rule",
		"model", "insurance", "policy", "coverage",
		"eligibility", "benefit", "plan",
	}

	// Addition keywords
	additionKeywords := []string{
		"add", "new", "create", "implement", "build",
		"develop", "introduce", "launch",
	}

	// Check if task mentions adding new API
	hasAPI := false
	for _, keyword := range apiKeywords {
		if strings.Contains(content, keyword) {
			hasAPI = true
			break
		}
	}

	// Check if task mentions adding new inventory
	hasInventory := false
	for _, keyword := range inventoryKeywords {
		if strings.Contains(content, keyword) {
			hasInventory = true
			break
		}
	}

	// Check if task mentions addition/creation
	hasAddition := false
	for _, keyword := range additionKeywords {
		if strings.Contains(content, keyword) {
			hasAddition = true
			break
		}
	}

	// Return true if it's adding new API or inventory
	return hasAddition && (hasAPI || hasInventory)
}

// isBugOrFixPastRollout checks if task is a bug fix after asset rollout
func (c *BusinessRulesClassifier) isBugOrFixPastRollout(task *taskdomain.Task, asset *assetdomain.Asset) bool {
	// Check if it's a bug or fix
	if !c.isBugOrFix(task) {
		return false
	}

	// If no asset or asset is not rolled out, it's not past rollout
	if asset == nil || !asset.IsRolledOut100 {
		return false
	}

	// If launch date is zero, we can't determine if it's past rollout
	if asset.LaunchDate.IsZero() {
		return false
	}

	// If asset was launched more than 6 months ago, consider it past rollout
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	return asset.LaunchDate.Before(sixMonthsAgo)
}

// isBugOrFix checks if task is a bug fix or maintenance task
func (c *BusinessRulesClassifier) isBugOrFix(task *taskdomain.Task) bool {
	// Check task type
	if task.Type == taskdomain.TaskTypeBug {
		return true
	}

	content := strings.ToLower(task.Summary + " " + task.Description)

	// Bug/fix keywords
	bugKeywords := []string{
		"bug", "fix", "error", "issue", "problem",
		"defect", "hotfix", "patch", "repair",
		"broken", "failing", "regression",
	}

	for _, keyword := range bugKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	// Check labels
	for _, label := range task.Labels {
		labelLower := strings.ToLower(label)
		for _, keyword := range bugKeywords {
			if strings.Contains(labelLower, keyword) {
				return true
			}
		}
	}

	return false
}

// classifyByContent provides fallback classification based on content analysis
func (c *BusinessRulesClassifier) classifyByContent(task *taskdomain.Task) taskdomain.WorkType {
	content := strings.ToLower(task.Summary + " " + task.Description)

	// Discovery keywords
	discoveryKeywords := []string{
		"research", "investigate", "analyze", "study",
		"explore", "discover", "understand", "learn",
	}

	// Development keywords
	developmentKeywords := []string{
		"implement", "build", "create", "develop",
		"add", "new", "feature", "enhancement",
	}

	// Maintenance keywords
	maintenanceKeywords := []string{
		"fix", "bug", "issue", "problem", "error",
		"update", "improve", "optimize", "refactor",
	}

	// Count keyword matches
	discoveryCount := 0
	developmentCount := 0
	maintenanceCount := 0

	for _, keyword := range discoveryKeywords {
		if strings.Contains(content, keyword) {
			discoveryCount++
		}
	}

	for _, keyword := range developmentKeywords {
		if strings.Contains(content, keyword) {
			developmentCount++
		}
	}

	for _, keyword := range maintenanceKeywords {
		if strings.Contains(content, keyword) {
			maintenanceCount++
		}
	}

	// Return the category with the most matches
	if discoveryCount > developmentCount && discoveryCount > maintenanceCount {
		return taskdomain.WorkTypeDiscovery
	}

	if developmentCount > maintenanceCount {
		return taskdomain.WorkTypeDevelopment
	}

	if maintenanceCount > 0 {
		return taskdomain.WorkTypeMaintenance
	}

	// Default to development if no clear match
	return taskdomain.WorkTypeDevelopment
}
