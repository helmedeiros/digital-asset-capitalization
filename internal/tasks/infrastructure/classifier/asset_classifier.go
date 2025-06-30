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

	// Find the best matching asset
	bestMatch := c.findBestAssetMatch(task, assets)

	return bestMatch, nil
}

// ClassifyTasksAssets determines the related asset for multiple tasks
func (c *ContentBasedAssetClassifier) ClassifyTasksAssets(tasks []*taskdomain.Task) ([]*ports.AssetClassificationResult, error) {
	var results []*ports.AssetClassificationResult

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
	var bestAsset *assetdomain.Asset
	var bestScore float64
	var bestReason string

	// Check each asset for matches
	for _, asset := range assets {
		score, reason := c.calculateAssetMatchScore(task, asset)
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
	taskContent := strings.ToLower(task.Summary + " " + task.Description)
	epicContent := strings.ToLower(task.Epic)
	assetNameLower := strings.ToLower(asset.Name)

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

	// 2. Check for exact asset name match in task content (high priority)
	nameInContentMatch := strings.Contains(taskContent, assetNameLower)
	if nameInContentMatch {
		currentScore := 0.9
		if currentScore > bestScore {
			bestScore = currentScore
			primaryReason = "asset name match in task summary"
		}
		matchTypes++
	}

	// 3. Check for asset name match in epic (medium-high priority)
	nameInEpicMatch := epicContent != "" && strings.Contains(epicContent, assetNameLower)
	if nameInEpicMatch {
		currentScore := 0.8
		if currentScore > bestScore {
			bestScore = currentScore
			primaryReason = "asset name match in epic name"
		}
		matchTypes++
	}

	// 4. Check for keyword matches (medium priority)
	keywordMatches := 0
	for _, keyword := range asset.Keywords {
		keywordLower := strings.ToLower(keyword)
		if strings.Contains(taskContent, keywordLower) || strings.Contains(epicContent, keywordLower) {
			keywordMatches++
		}
	}

	if keywordMatches > 0 {
		// Score based on number of keyword matches
		currentScore := 0.4 + float64(keywordMatches)*0.1
		if keywordMatches >= 3 {
			currentScore = 0.7 // Cap for multiple matches
		}
		if currentScore > bestScore {
			bestScore = currentScore
			primaryReason = "keyword match in task content"
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
