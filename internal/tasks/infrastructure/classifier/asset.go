package classifier

import (
	"strings"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	assetports "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

const (
	// DefaultAssetLabel is used when no asset is found for a task
	DefaultAssetLabel = "cap-asset-not-applicable"
)

// AssetClassifier implements the AssetClassifier interface using a chain of classifiers
type AssetClassifier struct {
	assetRepo assetports.AssetRepository
}

// NewAssetClassifier creates a new asset classifier
func NewAssetClassifier(assetRepo assetports.AssetRepository) *AssetClassifier {
	return &AssetClassifier{
		assetRepo: assetRepo,
	}
}

// ClassifyTaskAsset determines the associated asset for a task
func (c *AssetClassifier) ClassifyTaskAsset(task *taskdomain.Task) (string, error) {
	// Get all assets to check against
	assets, err := c.assetRepo.FindAll()
	if err != nil {
		return DefaultAssetLabel, err
	}

	// Try each classifier in the chain
	bestMatch := c.classifyByContent(task, assets)
	if bestMatch != "" {
		return bestMatch, nil
	}

	bestMatch = c.classifyByMetadata(task, assets)
	if bestMatch != "" {
		return bestMatch, nil
	}

	return DefaultAssetLabel, nil
}

// ClassifyTasksAssets determines the associated assets for multiple tasks
func (c *AssetClassifier) ClassifyTasksAssets(tasks []*taskdomain.Task) (map[string]string, error) {
	result := make(map[string]string)

	for _, task := range tasks {
		assetLabel, err := c.ClassifyTaskAsset(task)
		if err != nil {
			return nil, err
		}
		result[task.Key] = assetLabel
	}

	return result, nil
}

// classifyByContent checks task title and description for keywords matching asset names or keywords
func (c *AssetClassifier) classifyByContent(task *taskdomain.Task, assets []*assetdomain.Asset) string {
	content := strings.ToLower(task.Summary + " " + task.Description)
	var bestMatch string
	var bestScore float64

	for _, asset := range assets {
		// Check asset name
		if strings.Contains(content, strings.ToLower(asset.Name)) {
			score := float64(len(asset.Name)) / float64(len(content))
			if score > bestScore {
				bestScore = score
				bestMatch = "cap-asset-" + asset.Name
			}
		}

		// Check asset keywords
		for _, keyword := range asset.Keywords {
			if strings.Contains(content, strings.ToLower(keyword)) {
				score := float64(len(keyword)) / float64(len(content))
				if score > bestScore {
					bestScore = score
					bestMatch = "cap-asset-" + asset.Name
				}
			}
		}
	}

	return bestMatch
}

// classifyByMetadata checks task labels, epic title, and components for keywords matching asset names or keywords
func (c *AssetClassifier) classifyByMetadata(task *taskdomain.Task, assets []*assetdomain.Asset) string {
	var bestMatch string
	var bestScore float64

	// Check task labels
	for _, label := range task.Labels {
		label = strings.ToLower(label)
		for _, asset := range assets {
			// Check asset name in label
			if strings.Contains(label, strings.ToLower(asset.Name)) {
				score := float64(len(asset.Name)) / float64(len(label))
				if score > bestScore {
					bestScore = score
					bestMatch = "cap-asset-" + asset.Name
				}
			}

			// Check asset keywords in label
			for _, keyword := range asset.Keywords {
				if strings.Contains(label, strings.ToLower(keyword)) {
					score := float64(len(keyword)) / float64(len(label))
					if score > bestScore {
						bestScore = score
						bestMatch = "cap-asset-" + asset.Name
					}
				}
			}
		}
	}

	// Check epic title if available
	if task.Epic != "" {
		epicTitle := strings.ToLower(task.Epic)
		for _, asset := range assets {
			// Check asset name in epic title
			if strings.Contains(epicTitle, strings.ToLower(asset.Name)) {
				score := float64(len(asset.Name)) / float64(len(epicTitle))
				if score > bestScore {
					bestScore = score
					bestMatch = "cap-asset-" + asset.Name
				}
			}

			// Check asset keywords in epic title
			for _, keyword := range asset.Keywords {
				if strings.Contains(epicTitle, strings.ToLower(keyword)) {
					score := float64(len(keyword)) / float64(len(epicTitle))
					if score > bestScore {
						bestScore = score
						bestMatch = "cap-asset-" + asset.Name
					}
				}
			}
		}
	}

	return bestMatch
}
