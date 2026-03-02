package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// ClassifyTasksUseCase handles the classification of tasks for a project/sprint
type ClassifyTasksUseCase struct {
	localRepo    ports.TaskRepository
	remoteRepo   ports.TaskRepository
	classifier   ports.TaskClassifier
	userInput    ports.UserInput
	assetService assetsapp.AssetService
	lockRepo     ports.SprintLockRepository
}

// NewClassifyTasksUseCase creates a new instance of ClassifyTasksUseCase
func NewClassifyTasksUseCase(
	localRepo ports.TaskRepository,
	remoteRepo ports.TaskRepository,
	classifier ports.TaskClassifier,
	userInput ports.UserInput,
	assetService assetsapp.AssetService,
	lockRepo ports.SprintLockRepository,
) *ClassifyTasksUseCase {
	return &ClassifyTasksUseCase{
		localRepo:    localRepo,
		remoteRepo:   remoteRepo,
		classifier:   classifier,
		userInput:    userInput,
		assetService: assetService,
		lockRepo:     lockRepo,
	}
}

// Execute runs the task classification process
func (uc *ClassifyTasksUseCase) Execute(ctx context.Context, input domain.ClassifyTasksInput) error {
	// First, try to find existing tasks for the project/sprint
	tasks, err := uc.localRepo.FindByProjectAndSprint(ctx, input.Project, input.Sprint)
	if err != nil {
		return fmt.Errorf("failed to find existing tasks: %w", err)
	}

	// If no tasks found, ask user if they want to fetch them
	if len(tasks) == 0 {
		shouldFetch, confirmErr := uc.userInput.Confirm("No tasks found for project %s and sprint %s. Would you like to fetch them?", input.Project, input.Sprint)
		if confirmErr != nil {
			return fmt.Errorf("failed to get user confirmation: %w", confirmErr)
		}

		if shouldFetch {
			// Fetch tasks from the platform
			var fetchedTasks []*domain.Task
			var fetchErr error
			fetchedTasks, fetchErr = uc.remoteRepo.FindByProjectAndSprint(ctx, input.Project, input.Sprint)
			if fetchErr != nil {
				return fmt.Errorf("failed to fetch tasks: %w", fetchErr)
			}

			// Save fetched tasks to repository
			for _, task := range fetchedTasks {
				if saveErr := uc.localRepo.Save(ctx, task); saveErr != nil {
					return fmt.Errorf("failed to save fetched task %s: %w", task.Key, saveErr)
				}
			}
			tasks = fetchedTasks
		} else {
			return fmt.Errorf("no tasks available for classification")
		}
	}

	// Check sprint lock before applying to remote
	if input.Apply && uc.lockRepo != nil {
		lock, lockErr := uc.lockRepo.FindLock(ctx, input.Project, input.Sprint)
		if lockErr != nil {
			return fmt.Errorf("failed to check sprint lock: %w", lockErr)
		}

		if lock != nil {
			if !input.Force {
				return fmt.Errorf(
					"sprint %q in project %q was already classified on %s (%d tasks). Use --force to override",
					input.Sprint, input.Project, lock.LockedAt.Format("2006-01-02 15:04"), lock.TaskCount,
				)
			}

			confirmed, confirmErr := uc.userInput.Confirm(
				"Sprint %q in project %q was already classified on %s (%d tasks). Re-apply classifications?",
				input.Sprint, input.Project, lock.LockedAt.Format("2006-01-02 15:04"), lock.TaskCount,
			)
			if confirmErr != nil {
				return fmt.Errorf("failed to get user confirmation: %w", confirmErr)
			}
			if !confirmed {
				return fmt.Errorf("classification aborted by user")
			}
		}
	}

	// Preview classifications if in dry run mode
	if input.DryRun {
		return uc.previewClassifications(tasks, input.WithLLM)
	}

	// Update tasks with their classifications
	fmt.Printf("\n📝 APPLYING CLASSIFICATIONS\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")

	// Sort tasks alphabetically for consistent processing order
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Key < tasks[j].Key
	})

	// Use comprehensive classification to get both work type and asset information
	var classificationResults []*ports.ComprehensiveClassificationResult
	if comprehensiveClassifier, ok := uc.classifier.(ports.ComprehensiveTaskClassifier); ok {
		results, err := comprehensiveClassifier.ClassifyTasksComprehensive(tasks)
		if err != nil {
			return fmt.Errorf("failed to classify tasks comprehensively: %w", err)
		}
		classificationResults = results
	} else {
		// Fallback to simple classification if comprehensive is not available
		workTypes, err := uc.classifier.ClassifyTasks(tasks)
		if err != nil {
			return fmt.Errorf("failed to classify tasks: %w", err)
		}

		// Convert simple results to comprehensive format
		classificationResults = make([]*ports.ComprehensiveClassificationResult, 0, len(tasks))
		for _, task := range tasks {
			result := &ports.ComprehensiveClassificationResult{
				Task:     task,
				WorkType: workTypes[task.Key],
				Asset:    nil, // No asset information in simple classification
			}
			classificationResults = append(classificationResults, result)
		}
	}

	successCount := 0
	for _, result := range classificationResults {
		task := result.Task
		workType := result.WorkType

		if err := task.UpdateWorkType(workType); err != nil {
			return fmt.Errorf("failed to update work type for task %s: %w", task.Key, err)
		}

		// Save updated task locally
		if err := uc.localRepo.Save(ctx, task); err != nil {
			return fmt.Errorf("failed to save classified task %s: %w", task.Key, err)
		}

		// Apply labels to Jira if requested
		if input.Apply {
			// Build new labels preserving existing ones but updating work type and asset
			newLabels := uc.buildUpdatedLabels(task.Labels, workType, result.Asset)

			fmt.Printf("  🏷️  %s → %s", task.Key, workType)
			if result.Asset != nil && result.Asset.Asset != nil {
				fmt.Printf(" + %s", uc.getAssetLabel(result.Asset.Asset))
			}

			if err := uc.remoteRepo.UpdateLabels(ctx, task.Key, newLabels); err != nil {
				fmt.Printf(" ❌ Failed to update JIRA\n")
				return fmt.Errorf("failed to apply labels to task %s: %w", task.Key, err)
			}
			fmt.Printf(" ✅ Applied to JIRA\n")
		} else {
			fmt.Printf("  💾 %s → %s (saved locally)\n", task.Key, workType)
		}
		successCount++
	}

	fmt.Printf("\n✅ Successfully processed %d tasks\n", successCount)
	if input.Apply {
		fmt.Printf("🎯 All work type and asset labels have been written to JIRA\n")

		// Save sprint lock after successful apply
		if uc.lockRepo != nil {
			lock := domain.NewSprintLock(input.Project, input.Sprint, successCount)
			if lockErr := uc.lockRepo.SaveLock(ctx, lock); lockErr != nil {
				fmt.Printf("⚠️  Warning: failed to save sprint lock: %v\n", lockErr)
			}
		}
	} else {
		fmt.Printf("💾 Classifications saved locally (use --apply to write to JIRA)\n")
	}

	return nil
}

// previewClassifications shows classification preview with enhanced output when comprehensive results are available
// Includes intelligent asset syncing when unassigned tasks are detected
func (uc *ClassifyTasksUseCase) previewClassifications(tasks []*domain.Task, withLLM bool) error {
	fmt.Printf("\n🔍 CLASSIFICATION PREVIEW\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("Found %d task(s) to classify\n\n", len(tasks))
	if withLLM {
		fmt.Printf("LLM comparison mode enabled\n\n")
	}
	return uc.previewClassificationsWithRetry(tasks, false, withLLM)
}

// LLMToggler allows toggling LLM comparison mode on a classifier
type LLMToggler interface {
	SetLLMEnabled(enabled bool)
}

// previewClassificationsWithRetry handles the classification preview with optional asset sync retry
func (uc *ClassifyTasksUseCase) previewClassificationsWithRetry(tasks []*domain.Task, hasTriedSync bool, withLLM bool) error {
	fmt.Println("\nPreview of task classifications:")

	// Enable LLM if requested
	if withLLM {
		if toggler, ok := uc.classifier.(LLMToggler); ok {
			toggler.SetLLMEnabled(true)
			defer toggler.SetLLMEnabled(false)
		}
	}

	// Check if classifier supports comprehensive results
	if comprehensiveClassifier, ok := uc.classifier.(ports.ComprehensiveTaskClassifier); ok {
		// Use comprehensive classification for detailed preview
		results, err := comprehensiveClassifier.ClassifyTasksComprehensive(tasks)
		if err != nil {
			return fmt.Errorf("failed to classify tasks comprehensively: %w", err)
		}

		// Group results by work type for better organization
		workTypeGroups := make(map[domain.WorkType][]*ports.ComprehensiveClassificationResult)
		var unassignedTasks []string

		for _, result := range results {
			workTypeGroups[result.WorkType] = append(workTypeGroups[result.WorkType], result)
			if result.Asset == nil || result.Asset.Asset == nil {
				unassignedTasks = append(unassignedTasks, result.Task.Key)
			}
		}

		// Display results grouped by work type
		for workType, groupResults := range workTypeGroups {
			// Sort tasks within each group alphabetically by task key
			sort.Slice(groupResults, func(i, j int) bool {
				return groupResults[i].Task.Key < groupResults[j].Task.Key
			})

			fmt.Printf("📋 %s (%d tasks)\n", formatWorkType(workType), len(groupResults))
			fmt.Printf("─────────────────────────────────────────────────────────────\n")

			for _, result := range groupResults {
				fmt.Printf("  🎯 %s: %s\n", result.Task.Key, result.Task.Summary)

				// Show asset association
				if result.Asset != nil && result.Asset.Asset != nil {
					if result.LLMAsset != nil {
						fmt.Printf("     [Heuristic] Asset: %s (%.0f%% confidence)\n", result.Asset.Asset.Name, result.Asset.Confidence*100)
					} else {
						fmt.Printf("     💼 Asset: %s (%.0f%% confidence)\n", result.Asset.Asset.Name, result.Asset.Confidence*100)
					}
					if result.Asset.Reason != "" {
						fmt.Printf("     📝 Match: %s\n", result.Asset.Reason)
					}
				} else {
					fmt.Printf("     ❌ Asset: No assignment found\n")
				}

				// Show LLM classification if available
				if result.LLMAsset != nil {
					if result.LLMAsset.Asset != nil {
						fmt.Printf("     [LLM] Asset: %s (%.0f%% confidence)\n", result.LLMAsset.Asset.Name, result.LLMAsset.Confidence*100)
						if result.LLMAsset.Reason != "" {
							fmt.Printf("     📝 LLM Reason: %s\n", result.LLMAsset.Reason)
						}
					} else {
						fmt.Printf("     [LLM] Asset: No assignment found\n")
					}

					// Highlight disagreement
					heuristicName := ""
					llmName := ""
					if result.Asset != nil && result.Asset.Asset != nil {
						heuristicName = result.Asset.Asset.Name
					}
					if result.LLMAsset.Asset != nil {
						llmName = result.LLMAsset.Asset.Name
					}
					if heuristicName != llmName {
						fmt.Printf("     ** DISAGREEMENT: Heuristic=%q vs LLM=%q **\n", heuristicName, llmName)
					}
				}

				// Show work type reasoning
				if result.WorkTypeReason != "" {
					fmt.Printf("     🔍 Reason: %s\n", result.WorkTypeReason)
				}

				// Show task metadata
				fmt.Printf("     📊 Type: %s | Status: %s", result.Task.Type, result.Task.Status)
				if result.Task.Epic != "" {
					fmt.Printf(" | Epic: %s", result.Task.Epic)
				}
				if len(result.Task.Labels) > 0 {
					fmt.Printf(" | Labels: %v", result.Task.Labels)
				}
				fmt.Printf("\n\n")
			}
		}

		// Show LLM comparison summary if LLM was used
		if withLLM {
			agreements := 0
			disagreements := 0
			llmUsed := 0
			for _, result := range results {
				if result.LLMAsset == nil {
					continue
				}
				llmUsed++
				heuristicName := ""
				llmName := ""
				if result.Asset != nil && result.Asset.Asset != nil {
					heuristicName = result.Asset.Asset.Name
				}
				if result.LLMAsset.Asset != nil {
					llmName = result.LLMAsset.Asset.Name
				}
				if heuristicName == llmName {
					agreements++
				} else {
					disagreements++
				}
			}
			if llmUsed > 0 {
				fmt.Printf("\n📊 LLM COMPARISON SUMMARY\n")
				fmt.Printf("─────────────────────────────────────────────────────────────\n")
				fmt.Printf("  Total tasks: %d | LLM classified: %d\n", len(results), llmUsed)
				fmt.Printf("  Agreements: %d | Disagreements: %d\n", agreements, disagreements)
				if disagreements > 0 {
					fmt.Printf("  Review disagreements above to evaluate LLM accuracy\n")
				}
				fmt.Println()
			}
		}

		// If there are unassigned tasks and we haven't tried syncing yet, offer to sync assets
		if len(unassignedTasks) > 0 && !hasTriedSync {
			// Sort unassigned tasks alphabetically for consistent display
			sort.Strings(unassignedTasks)

			fmt.Printf("\nFound %d task(s) without asset assignments: %v\n", len(unassignedTasks), unassignedTasks)

			shouldSync, confirmErr := uc.userInput.Confirm("Would you like to sync assets from Confluence to potentially improve classification?")
			if confirmErr != nil {
				return fmt.Errorf("failed to get user confirmation for asset sync: %w", confirmErr)
			}

			if shouldSync {
				fmt.Println("Syncing assets from Confluence...")

				// Sync assets with default parameters (CAP space, cap-asset label)
				syncResult, syncErr := uc.assetService.SyncFromConfluence("CAP", "cap-asset", false)
				if syncErr != nil {
					fmt.Printf("Warning: Asset sync failed: %v\n", syncErr)
					fmt.Println("Continuing with current classification results...")
				} else {
					fmt.Printf("Asset sync completed: %d assets synced, %d not synced\n",
						len(syncResult.SyncedAssets), len(syncResult.NotSyncedAssets))

					// Re-run classification with updated assets (only once to avoid loops)
					fmt.Println("\nRe-running classification with updated assets...")
					return uc.previewClassificationsWithRetry(tasks, true, withLLM)
				}
			}
		}
	} else {
		// Fallback to simple classification for backward compatibility
		workTypes, err := uc.classifier.ClassifyTasks(tasks)
		if err != nil {
			return fmt.Errorf("failed to classify tasks: %w", err)
		}

		// Sort tasks alphabetically for consistent display
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].Key < tasks[j].Key
		})

		for _, task := range tasks {
			workType := workTypes[task.Key]
			fmt.Printf("- %s: %s (%s)\n", task.Key, workType, task.Summary)
		}
	}

	return nil
}

// GetTasks retrieves tasks for a project and sprint
func (uc *ClassifyTasksUseCase) GetTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	// Try to get tasks from local repository first
	tasks, err := uc.localRepo.FindByProjectAndSprint(ctx, project, sprint)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing tasks: %w", err)
	}

	// If no tasks found locally, try to fetch from remote
	if len(tasks) == 0 {
		remoteTasks, fetchErr := uc.remoteRepo.FindByProjectAndSprint(ctx, project, sprint)
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch tasks from remote: %w", fetchErr)
		}

		// Save remote tasks to local repository
		for _, task := range remoteTasks {
			if saveErr := uc.localRepo.Save(ctx, task); saveErr != nil {
				return nil, fmt.Errorf("failed to save fetched task: %w", saveErr)
			}
		}

		return remoteTasks, nil
	}

	return tasks, nil
}

// GetAllTasks retrieves all tasks from the local repository
func (uc *ClassifyTasksUseCase) GetAllTasks(ctx context.Context) ([]*domain.Task, error) {
	return uc.localRepo.FindAll(ctx)
}

func (uc *ClassifyTasksUseCase) GetLocalRepository() ports.TaskRepository {
	return uc.localRepo
}

// formatWorkType formats work type for display
func formatWorkType(workType domain.WorkType) string {
	switch workType {
	case domain.WorkTypeDiscovery:
		return "🔍 DISCOVERY"
	case domain.WorkTypeDevelopment:
		return "🚀 DEVELOPMENT"
	case domain.WorkTypeMaintenance:
		return "🔧 MAINTENANCE"
	default:
		return "❓ UNKNOWN"
	}
}

// buildUpdatedLabels builds new labels preserving existing ones but updating work type and asset
func (uc *ClassifyTasksUseCase) buildUpdatedLabels(existingLabels []string, workType domain.WorkType, assetResult *ports.AssetClassificationResult) []string {
	// Pre-allocate with estimated capacity: existing labels + work type + potential asset label
	newLabels := make([]string, 0, len(existingLabels)+2)

	// Check if we should preserve existing asset labels
	preserveExistingAsset := assetResult != nil && assetResult.Reason == "existing asset label preserved" && assetResult.Confidence >= 0.95

	// Keep all labels except old work type and conditionally old asset labels
	for _, label := range existingLabels {
		// Skip old work type labels
		if label == "cap-development" || label == "cap-maintenance" || label == "cap-discovery" {
			continue
		}
		// Skip old asset labels ONLY if we're not preserving them
		if strings.HasPrefix(label, "cap-asset-") {
			if preserveExistingAsset {
				// Keep existing asset labels when they should be preserved
				newLabels = append(newLabels, label)
			}
			continue
		}
		// Keep all other labels
		newLabels = append(newLabels, label)
	}

	// Add new work type label
	newLabels = append(newLabels, string(workType))

	// Add new asset label if available and not preserving existing ones
	if assetResult != nil && assetResult.Asset != nil && !preserveExistingAsset {
		assetLabel := uc.getAssetLabel(assetResult.Asset)
		newLabels = append(newLabels, assetLabel)
	}

	return newLabels
}

// getAssetLabel returns the proper asset label, preferring the asset ID if available
func (uc *ClassifyTasksUseCase) getAssetLabel(asset interface{}) string {
	// If we receive a full Asset object, use its ID or Name
	if assetObj, ok := asset.(*assetdomain.Asset); ok {
		id := assetObj.GetID()
		if strings.HasPrefix(id, "cap-asset-") {
			return id
		}
		// Fallback to generating from Name
		if assetObj.Name != "" {
			return formatAssetLabel(assetObj.Name)
		}
	}

	// Fallback: generate from asset name string
	var assetName string
	switch v := asset.(type) {
	case string:
		assetName = v
	default:
		assetName = "unknown"
	}

	return formatAssetLabel(assetName)
}

// formatAssetLabel converts an asset name to a cap-asset-* label format
func formatAssetLabel(name string) string {
	labelName := strings.ToLower(name)
	labelName = strings.ReplaceAll(labelName, " ", "-")
	labelName = strings.ReplaceAll(labelName, "(", "")
	labelName = strings.ReplaceAll(labelName, ")", "")
	labelName = strings.ReplaceAll(labelName, "&", "and")
	return fmt.Sprintf("cap-asset-%s", labelName)
}
