package infrastructure

import (
	"context"
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain/ports"
	tasksDomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	tasksPorts "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// AssetResolverAdapter adapts the task classifier to resolve assets for deployments
type AssetResolverAdapter struct {
	taskRepo       tasksPorts.TaskRepository
	taskClassifier tasksPorts.TaskClassifier
}

// NewAssetResolverAdapter creates a new asset resolver adapter
func NewAssetResolverAdapter(taskRepo tasksPorts.TaskRepository, classifier tasksPorts.TaskClassifier) ports.AssetResolver {
	return &AssetResolverAdapter{
		taskRepo:       taskRepo,
		taskClassifier: classifier,
	}
}

// ResolveAssetsForTasks resolves asset associations for given task keys
func (a *AssetResolverAdapter) ResolveAssetsForTasks(ctx context.Context, taskKeys []string) ([]ports.AssetInfo, error) {
	if len(taskKeys) == 0 {
		return []ports.AssetInfo{}, nil
	}

	// Fetch tasks by keys
	tasks := make([]*tasksDomain.Task, 0, len(taskKeys))
	for _, key := range taskKeys {
		task, err := a.taskRepo.FindByKey(ctx, key)
		if err != nil {
			// Skip tasks that can't be found
			continue
		}
		tasks = append(tasks, task)
	}

	if len(tasks) == 0 {
		return []ports.AssetInfo{}, nil
	}

	// Use comprehensive classification if available
	assetMap := make(map[string]*ports.AssetInfo)

	if comprehensiveClassifier, ok := a.taskClassifier.(tasksPorts.ComprehensiveTaskClassifier); ok {
		results, err := comprehensiveClassifier.ClassifyTasksComprehensive(tasks)
		if err != nil {
			return nil, fmt.Errorf("failed to classify tasks: %w", err)
		}

		// Aggregate results by asset
		for i, result := range results {
			if i >= len(tasks) {
				break
			}
			task := tasks[i]

			// Check if the result has an asset classification
			if result.Asset != nil && result.Asset.Asset != nil {
				assetName := result.Asset.Asset.Name
				if assetName == "" {
					continue
				}

				if _, exists := assetMap[assetName]; !exists {
					assetMap[assetName] = &ports.AssetInfo{
						Name:      assetName,
						TaskCount: 0,
						TaskKeys:  []string{},
					}
				}

				assetMap[assetName].TaskCount++
				assetMap[assetName].TaskKeys = append(assetMap[assetName].TaskKeys, task.Key)
			}
		}
	} else {
		// Fallback: Use existing labels from tasks
		for _, task := range tasks {
			for _, label := range task.Labels {
				if label == "" {
					continue
				}

				if _, exists := assetMap[label]; !exists {
					assetMap[label] = &ports.AssetInfo{
						Name:      label,
						TaskCount: 0,
						TaskKeys:  []string{},
					}
				}

				assetMap[label].TaskCount++
				assetMap[label].TaskKeys = append(assetMap[label].TaskKeys, task.Key)
			}
		}
	}

	// Convert map to slice
	result := make([]ports.AssetInfo, 0, len(assetMap))
	for _, asset := range assetMap {
		result = append(result, *asset)
	}

	return result, nil
}

// ResolveAssetsForTask resolves asset associations for a single task
func (a *AssetResolverAdapter) ResolveAssetsForTask(ctx context.Context, taskKey string) ([]string, error) {
	if taskKey == "" {
		return []string{}, nil
	}

	task, err := a.taskRepo.FindByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find task %s: %w", taskKey, err)
	}

	// Use comprehensive classification if available
	if comprehensiveClassifier, ok := a.taskClassifier.(tasksPorts.ComprehensiveTaskClassifier); ok {
		results, err := comprehensiveClassifier.ClassifyTasksComprehensive([]*tasksDomain.Task{task})
		if err != nil {
			return nil, fmt.Errorf("failed to classify task: %w", err)
		}

		if len(results) > 0 && results[0].Asset != nil && results[0].Asset.Asset != nil {
			return []string{results[0].Asset.Asset.Name}, nil
		}
	}

	// Fallback to existing labels
	return task.Labels, nil
}
