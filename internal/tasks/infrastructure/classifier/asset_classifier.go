package classifier

import (
	"fmt"
	"strings"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	assetports "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// ContentBasedAssetClassifier implements asset classification based on content analysis
type ContentBasedAssetClassifier struct {
	assetRepo assetports.AssetRepository
}

// NewContentBasedAssetClassifier creates a new content-based asset classifier
func NewContentBasedAssetClassifier(assetRepo assetports.AssetRepository) ports.AssetClassifier {
	return &ContentBasedAssetClassifier{
		assetRepo: assetRepo,
	}
}

// ClassifyTaskAsset determines which asset a task belongs to based on content analysis
func (c *ContentBasedAssetClassifier) ClassifyTaskAsset(task *taskdomain.Task) (*ports.AssetClassificationResult, error) {
	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}

	// Get all assets to analyze against
	assets, err := c.assetRepo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assets: %w", err)
	}

	// FIRST: Get the natural classification (what the content suggests)
	naturalClassification := c.findBestAssetMatch(task, assets)

	// SECOND: Check if task has existing cap-asset labels
	for _, label := range task.Labels {
		if strings.HasPrefix(strings.ToLower(label), "cap-asset-") {
			// Extract asset identifier from existing label
			assetIdentifier := strings.TrimPrefix(strings.ToLower(label), "cap-asset-")
			assetName := c.formatAssetNameFromLabel(assetIdentifier)

			// Create existing label result
			existingLabelResult := &ports.AssetClassificationResult{
				Task: task,
				Asset: &assetdomain.Asset{
					Name:     assetName,
					Keywords: []string{assetIdentifier},
				},
				Confidence: 0.85, // Good confidence for existing labels
				Reason:     "existing asset label preserved",
			}

			// Always preserve existing human-applied labels.
			// Existing cap-asset-* labels are the most reliable classification source.
			return existingLabelResult, nil
		}
	}

	// No existing labels, return natural classification
	return naturalClassification, nil
}

// formatAssetNameFromLabel converts an asset label identifier to a proper asset name
func (c *ContentBasedAssetClassifier) formatAssetNameFromLabel(identifier string) string {
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

// ClassifyTasksAssets determines the related asset for multiple tasks
func (c *ContentBasedAssetClassifier) ClassifyTasksAssets(tasks []*taskdomain.Task) ([]*ports.AssetClassificationResult, error) {
	results := make([]*ports.AssetClassificationResult, 0, len(tasks))

	for _, task := range tasks {
		result, err := c.ClassifyTaskAsset(task)
		if err != nil {
			return nil, fmt.Errorf("failed to classify task %s: %w", task.Key, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// findBestAssetMatch finds the asset that best matches the task
func (c *ContentBasedAssetClassifier) findBestAssetMatch(task *taskdomain.Task, assets []*assetdomain.Asset) *ports.AssetClassificationResult {
	// NEW: First check if there's a clear primary subject in the title
	primarySubject := c.detectPrimarySubject(task, assets)
	if primarySubject != nil {
		return &ports.AssetClassificationResult{
			Task:       task,
			Asset:      primarySubject,
			Confidence: 0.95,
			Reason:     "detected as primary subject based on title emphasis",
		}
	}

	var bestAsset *assetdomain.Asset
	var bestScore float64
	var bestReason string

	// Check each asset for matches
	for _, asset := range assets {
		score, reason := c.calculateAssetMatchScore(task, asset)

		// Boost score for team-owned assets: if asset's owning team matches task's project
		if asset.OwningTeam != "" && asset.OwningTeam == task.Project {
			score *= 1.2 // 20% boost for team-owned assets
			if score > 1.0 {
				score = 1.0
			}
			if score > bestScore {
				reason = reason + " (team-owned asset priority)"
			}
		}

		if score > bestScore {
			bestScore = score
			bestAsset = asset
			bestReason = reason
		}
	}

	// Create result based on best match
	result := &ports.AssetClassificationResult{
		Task:       task,
		Asset:      bestAsset,
		Confidence: bestScore,
		Reason:     bestReason,
	}

	if bestAsset == nil {
		result.Reason = "no matching asset found"
	}

	return result
}

// calculateAssetMatchScore calculates how well an asset matches a task
func (c *ContentBasedAssetClassifier) calculateAssetMatchScore(task *taskdomain.Task, asset *assetdomain.Asset) (float64, string) {
	var bestScore float64
	var primaryReason string
	matchTypes := 0

	// Prepare content for analysis (case-insensitive)
	// Separate title and description for weighted analysis
	taskSummary := strings.ToLower(task.Summary)
	taskDescription := strings.ToLower(task.Description)
	taskContent := taskSummary + " " + taskDescription
	epicContent := strings.ToLower(task.Epic)
	assetNameLower := strings.ToLower(asset.Name)

	// NEW: Check title first with highest priority (0.95 confidence)
	titleMatch := MatchesAssetNameEnhanced(taskSummary, assetNameLower)
	if titleMatch && bestScore < 0.95 {
		// Title match is very strong signal - the task is ABOUT this asset
		bestScore = 0.95
		primaryReason = "asset name appears in task title (primary indicator)"
		matchTypes++
	}

	// SPECIAL LOGIC: Detect primary focus for multi-asset tasks
	// If task title follows pattern "X-Based Y Experiment" or "X using Y",
	// then Y is likely the primary focus
	taskSummaryLower := strings.ToLower(task.Summary)
	if c.shouldPrioritizeSecondaryAsset(taskSummaryLower, assetNameLower) {
		// This asset appears to be the primary focus despite being mentioned second
		bestScore = 0.95
		primaryReason = "detected as primary focus despite secondary mention"
		return bestScore, primaryReason
	}

	// 1. Check for explicit asset label (highest priority)
	for _, label := range task.Labels {
		labelLower := strings.ToLower(label)
		if strings.HasPrefix(labelLower, "cap-asset-") {
			// Extract asset identifier from label
			assetIdentifier := strings.TrimPrefix(labelLower, "cap-asset-")
			// Check if this matches our asset (use first word of asset name)
			if len(strings.Fields(asset.Name)) > 0 {
				assetFirstWord := strings.ToLower(strings.Fields(asset.Name)[0])
				if assetIdentifier == assetFirstWord || strings.Contains(assetNameLower, assetIdentifier) {
					currentScore := 0.9
					if currentScore > bestScore {
						bestScore = currentScore
						primaryReason = "explicit asset label match"
					}
					matchTypes++
				}
			}
		}
	}

	// 2. Check for exact asset name match in task content (high priority) - ENHANCED
	nameInContentMatch := MatchesAssetNameEnhanced(taskContent, assetNameLower)
	if nameInContentMatch {
		// Check if this might be a secondary mention in a multi-asset task
		if c.isSecondaryMentionInMultiAssetTask(taskSummaryLower, assetNameLower) {
			// Lower the confidence for secondary mentions
			currentScore := 0.7
			if currentScore > bestScore {
				bestScore = currentScore
				primaryReason = "asset name match but appears secondary to main focus"
			}
		} else {
			currentScore := 0.9
			if currentScore > bestScore {
				bestScore = currentScore
				primaryReason = "asset name match in task summary"
			}
		}
		matchTypes++
	}

	// 3. Check for asset name match in epic (medium-high priority) - ENHANCED
	nameInEpicMatch := epicContent != "" && MatchesAssetNameEnhanced(epicContent, assetNameLower)
	if nameInEpicMatch {
		currentScore := 0.8
		if currentScore > bestScore {
			bestScore = currentScore
			primaryReason = "asset name match in epic name"
		}
		matchTypes++
	}

	// 4. Check for keyword matches (medium priority) - ENHANCED with title priority
	keywordMatches := 0
	strongKeywordMatches := 0
	titleKeywordMatches := 0
	for _, keyword := range asset.Keywords {
		inTitle := MatchesKeywordEnhanced(taskSummary, keyword)
		inDescription := MatchesKeywordEnhanced(taskDescription, keyword)
		inEpic := MatchesKeywordEnhanced(epicContent, keyword)

		if inTitle {
			titleKeywordMatches++ // Count title matches separately
			keywordMatches++
		} else if inDescription || inEpic {
			keywordMatches++
		}

		// Check if this is a strong contextual keyword match
		if c.isStrongContextualMatch(taskContent, keyword, asset.Name) {
			strongKeywordMatches++
		}
	}

	if keywordMatches > 0 && bestScore < 0.95 {
		// Don't override strong title matches
		// Score based on number of keyword matches
		currentScore := 0.4 + float64(keywordMatches)*0.1

		// Boost for title keyword matches (higher priority)
		if titleKeywordMatches >= 2 {
			currentScore += 0.3 // Strong boost for multiple title keywords
		} else if strongKeywordMatches >= 2 {
			currentScore += 0.3 // Boost for strong contextual matches
		}

		if keywordMatches >= 3 {
			currentScore = 0.7 // Cap for multiple matches
		}
		if currentScore > bestScore {
			bestScore = currentScore
			if titleKeywordMatches > 0 {
				primaryReason = fmt.Sprintf("keyword match in task title (%d matches)", titleKeywordMatches)
			} else {
				primaryReason = "keyword match in task content"
			}
		}
		matchTypes++
	}

	// 5. Check for partial asset name matches (lower priority)
	assetWords := strings.Fields(assetNameLower)
	partialMatches := 0
	for _, word := range assetWords {
		if len(word) > 3 && (strings.Contains(taskContent, word) || strings.Contains(epicContent, word)) {
			partialMatches++
		}
	}

	if partialMatches > 0 && bestScore < 0.5 {
		currentScore := 0.3 + float64(partialMatches)*0.1
		if currentScore > bestScore {
			bestScore = currentScore
			primaryReason = "partial asset name match"
		}
		matchTypes++
	}

	// Boost score for multiple match types and set appropriate reason
	if matchTypes >= 4 {
		bestScore = bestScore * 1.1
		if bestScore > 1.0 {
			bestScore = 1.0
		}
		primaryReason = "multiple strong matches"
	} else if matchTypes >= 2 && nameInContentMatch && keywordMatches >= 3 {
		// Only use "multiple strong matches" for very strong combinations
		bestScore = bestScore * 1.05
		if bestScore > 1.0 {
			bestScore = 1.0
		}
		primaryReason = "multiple strong matches"
	} else if matchTypes >= 2 {
		bestScore = bestScore * 1.05
		if bestScore > 1.0 {
			bestScore = 1.0
		}
		// Keep the highest priority reason for moderate multiple matches
	}

	// Set default reason if none found
	if primaryReason == "" {
		primaryReason = "no significant match"
	}

	return bestScore, primaryReason
}

// shouldPrioritizeSecondaryAsset detects if an asset should be prioritized despite being mentioned second
func (c *ContentBasedAssetClassifier) shouldPrioritizeSecondaryAsset(taskSummary, assetName string) bool {
	// Get individual words from asset name for partial matching
	assetWords := strings.Fields(strings.ToLower(assetName))

	// Pattern: "X-Based Y Experiment" -> Y is the primary focus
	// Example: "Dynamic Markup-Based Rounding Experiment" -> "Dynamic Rounding" is primary
	if strings.Contains(taskSummary, "-based") && strings.Contains(taskSummary, "experiment") {
		// Extract the word that appears immediately after "-based"
		parts := strings.Split(taskSummary, "-based")
		if len(parts) >= 2 {
			secondPart := strings.TrimSpace(parts[1])
			words := strings.Fields(secondPart)
			if len(words) > 0 {
				// Get the first word after "-based" (this is the focus word)
				focusWord := words[0]

				// Check if this focus word appears in our asset name
				for _, assetWord := range assetWords {
					if len(assetWord) > 3 && strings.Contains(focusWord, assetWord) {
						return true
					}
				}
			}
		}
	}

	// Pattern: "X using Y" -> Y is the primary focus
	if strings.Contains(taskSummary, " using ") {
		parts := strings.Split(taskSummary, " using ")
		if len(parts) >= 2 {
			secondPart := parts[1]
			// Check if any word from the asset name appears after "using"
			for _, assetWord := range assetWords {
				if len(assetWord) > 3 && strings.Contains(secondPart, assetWord) {
					return true
				}
			}
		}
	}

	// Pattern: task is about experiments/testing of asset keywords
	if strings.Contains(taskSummary, "experiment") || strings.Contains(taskSummary, "test") {
		// Check if any asset word appears in the context of being tested/experimented
		for _, assetWord := range assetWords {
			if len(assetWord) > 3 {
				// Look for patterns like "rounding experiment", "pricing test", etc.
				if strings.Contains(taskSummary, assetWord+" experiment") ||
					strings.Contains(taskSummary, assetWord+" test") {
					return true
				}
			}
		}
	}

	// Special case: if asset contains "rounding" and task is a "rounding experiment"
	if strings.Contains(strings.ToLower(assetName), "rounding") &&
		strings.Contains(taskSummary, "rounding") && strings.Contains(taskSummary, "experiment") {
		return true
	}

	return false
}

// isSecondaryMentionInMultiAssetTask checks if an asset mention appears to be secondary
func (c *ContentBasedAssetClassifier) isSecondaryMentionInMultiAssetTask(taskSummary, assetName string) bool {
	// If task mentions this asset in a "using X" or "X-based" context, it might be secondary
	return strings.Contains(taskSummary, assetName+"-based") ||
		strings.Contains(taskSummary, "using "+assetName)
}

// isStrongContextualMatch checks if a keyword match is particularly strong given the context
func (c *ContentBasedAssetClassifier) isStrongContextualMatch(content, keyword, assetName string) bool {
	// If keyword appears near other words from the asset name, it's a strong match
	assetWords := strings.Fields(strings.ToLower(assetName))
	for _, assetWord := range assetWords {
		if strings.Contains(content, keyword+" "+assetWord) ||
			strings.Contains(content, assetWord+" "+keyword) {
			return true
		}
	}

	// If keyword appears in a strong context (experiment, test, implementation)
	contextWords := []string{"experiment", "test", "implementation", "platform", "system"}
	for _, contextWord := range contextWords {
		if strings.Contains(content, keyword+" "+contextWord) ||
			strings.Contains(content, contextWord+" "+keyword) {
			return true
		}
	}

	return false
}

// detectPrimarySubject identifies the main asset being discussed in the task
// Returns the asset with highest confidence if it's clearly the primary focus
func (c *ContentBasedAssetClassifier) detectPrimarySubject(task *taskdomain.Task, assets []*assetdomain.Asset) *assetdomain.Asset {
	taskSummary := strings.ToLower(task.Summary)

	// Count mentions of each asset in the title
	titleMentions := make(map[string]int)
	for _, asset := range assets {
		assetNameLower := strings.ToLower(asset.Name)

		// Check direct asset name mention
		if strings.Contains(taskSummary, assetNameLower) {
			titleMentions[asset.Name]++
		}

		// Check keyword mentions in title
		for _, keyword := range asset.Keywords {
			if len(keyword) > 3 && strings.Contains(taskSummary, strings.ToLower(keyword)) {
				titleMentions[asset.Name]++
			}
		}
	}

	// If one asset clearly dominates the title, it's the primary subject
	var primaryAsset *assetdomain.Asset
	maxMentions := 0
	for _, asset := range assets {
		if mentions := titleMentions[asset.Name]; mentions > maxMentions {
			maxMentions = mentions
			primaryAsset = asset
		}
	}

	// Only return if we have clear evidence (at least 2 mentions)
	if maxMentions >= 2 {
		return primaryAsset
	}

	return nil
}
