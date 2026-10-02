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

// CapitalizationLabelProvider looks up the real accounting labels a company
// configures per project (e.g. "IAS38-CapEx" / "IAS38-OpEx") for the
// capitalizable/non-capitalizable split. That label vocabulary is company
// configuration data -- it lives in team config (teams.json), never as a
// hardcoded string in this framework -- so implementations read it from
// wherever that configuration is stored. An empty return value means the
// project hasn't configured one, and no label is written for it.
type CapitalizationLabelProvider interface {
	// GetCapExLabel returns the label for capitalizable (development) work.
	GetCapExLabel(project string) string
	// GetOpExLabel returns the label for non-capitalizable (discovery and
	// maintenance) work.
	GetOpExLabel(project string) string
}

// ClassifyTasksUseCase handles the classification of tasks for a project/sprint
type ClassifyTasksUseCase struct {
	localRepo        ports.TaskRepository
	remoteRepo       ports.TaskRepository
	classifier       ports.TaskClassifier
	userInput        ports.UserInput
	assetService     assetsapp.AssetService
	lockRepo         ports.SprintLockRepository
	capLabelProvider CapitalizationLabelProvider
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

// SetCapitalizationLabelProvider configures the per-project real accounting
// label lookup. Optional: when never called, no project has a configured
// label and classification behaves exactly as it did before this existed
// (only the internal cap-development/cap-maintenance/cap-discovery label is
// written).
func (uc *ClassifyTasksUseCase) SetCapitalizationLabelProvider(provider CapitalizationLabelProvider) {
	uc.capLabelProvider = provider
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

			// Save fetched tasks to repository in one batch -- per-task Save
			// would re-read and re-write the entire JSON file once per task.
			if saveErr := uc.localRepo.SaveAll(ctx, fetchedTasks); saveErr != nil {
				return fmt.Errorf("failed to save fetched tasks: %w", saveErr)
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

	// Phase 1: stamp every task with its new work type in memory.
	updatedTasks := make([]*domain.Task, 0, len(classificationResults))
	for _, result := range classificationResults {
		if err := result.Task.UpdateWorkType(result.WorkType); err != nil {
			return fmt.Errorf("failed to update work type for task %s: %w", result.Task.Key, err)
		}
		updatedTasks = append(updatedTasks, result.Task)
	}

	// Phase 2: persist the whole batch locally in a single load/store
	// cycle. Doing this BEFORE pushing to JIRA preserves the original
	// invariant that any task whose classification we attempted to push
	// remotely is already saved locally -- so a mid-loop JIRA failure
	// doesn't leave the on-disk state out of sync with what's already
	// landed on the remote.
	if err := uc.localRepo.SaveAll(ctx, updatedTasks); err != nil {
		return fmt.Errorf("failed to persist classified tasks: %w", err)
	}

	// Phase 3: apply label updates to JIRA (or just narrate when in
	// local-only mode). A single task's JIRA update can fail for reasons
	// specific to that one issue (e.g. its screen configuration rejects
	// the labels field) that have nothing to do with the rest of the
	// batch, so a failure here is recorded and the loop continues rather
	// than abandoning every task still to come. Previously persisted
	// local state stays correct because of phase 2.
	successCount := 0
	var failures []string
	for _, result := range classificationResults {
		task := result.Task
		workType := result.WorkType

		if input.Apply {
			var capExLabel, opExLabel string
			if uc.capLabelProvider != nil {
				capExLabel = uc.capLabelProvider.GetCapExLabel(input.Project)
				opExLabel = uc.capLabelProvider.GetOpExLabel(input.Project)
			}
			addLabels, removeLabels := uc.buildLabelChanges(task.Labels, workType, result.Asset, capExLabel, opExLabel)

			fmt.Printf("  🏷️  %s → %s", task.Key, workType)
			if result.Asset != nil && result.Asset.Asset != nil {
				fmt.Printf(" + %s", uc.getAssetLabel(result.Asset.Asset))
			}

			if err := uc.remoteRepo.UpdateLabels(ctx, task.Key, addLabels, removeLabels); err != nil {
				fmt.Printf(" ❌ Failed to update JIRA: %v\n", err)
				failures = append(failures, fmt.Sprintf("%s: %v", task.Key, err))
				continue
			}
			fmt.Printf(" ✅ Applied to JIRA\n")
		} else {
			fmt.Printf("  💾 %s → %s (saved locally)\n", task.Key, workType)
		}
		successCount++
	}

	fmt.Printf("\n✅ Successfully processed %d tasks\n", successCount)
	if len(failures) > 0 {
		fmt.Printf("⚠️  %d task(s) failed to update in JIRA:\n", len(failures))
		for _, f := range failures {
			fmt.Printf("    - %s\n", f)
		}
	}
	if input.Apply {
		if len(failures) > 0 {
			fmt.Printf("🎯 Work type and asset labels have been written to JIRA for %d of %d tasks\n", successCount, len(classificationResults))
		} else {
			fmt.Printf("🎯 All work type and asset labels have been written to JIRA\n")
		}

		// Save sprint lock after apply, even with partial failures: the
		// tasks that did succeed are already live on JIRA, and a locked
		// sprint can still be revisited with --force once the few
		// failures are investigated.
		if uc.lockRepo != nil {
			lock := domain.NewSprintLock(input.Project, input.Sprint, successCount)
			if lockErr := uc.lockRepo.SaveLock(ctx, lock); lockErr != nil {
				fmt.Printf("⚠️  Warning: failed to save sprint lock: %v\n", lockErr)
			}
		}
	} else {
		fmt.Printf("💾 Classifications saved locally (use --apply to write to JIRA)\n")
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to apply labels to %d task(s): %s", len(failures), strings.Join(failures, "; "))
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

		// Save remote tasks to local repository in one batch.
		if saveErr := uc.localRepo.SaveAll(ctx, remoteTasks); saveErr != nil {
			return nil, fmt.Errorf("failed to save fetched tasks: %w", saveErr)
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

// buildLabelChanges computes which cap-prefixed labels (and, when the
// project has configured one, a real accounting label) to add and remove so
// that JIRA's update operations only touch labels this tool owns.
//
// capExLabel/opExLabel are the project's configured real accounting labels
// (e.g. "IAS38-CapEx" / "IAS38-OpEx") for capitalizable/non-capitalizable
// work, looked up from team configuration -- never hardcoded here, since
// that vocabulary is company-specific data, not framework behavior. Pass ""
// for either when the project hasn't configured one; no label is added for
// an empty value.
func (uc *ClassifyTasksUseCase) buildLabelChanges(existingLabels []string, workType domain.WorkType, assetResult *ports.AssetClassificationResult, capExLabel, opExLabel string) (addLabels, removeLabels []string) {
	preserveExistingAsset := assetResult != nil && assetResult.Reason == "existing asset label preserved" && assetResult.Confidence >= 0.95

	// A human/finance-reviewed accounting label is treated as final: once
	// one of the project's configured labels is present, this tool never
	// adds, removes, or replaces it, even on re-classification.
	hasExistingAccountingLabel := false
	if capExLabel != "" || opExLabel != "" {
		for _, label := range existingLabels {
			if label == capExLabel || label == opExLabel {
				hasExistingAccountingLabel = true
				break
			}
		}
	}

	// Collect old cap work-type and asset labels to remove
	for _, label := range existingLabels {
		if label == "cap-development" || label == "cap-maintenance" || label == "cap-discovery" {
			removeLabels = append(removeLabels, label)
		}
		if strings.HasPrefix(label, "cap-asset-") && !preserveExistingAsset {
			removeLabels = append(removeLabels, label)
		}
	}

	// Add new work type label
	addLabels = append(addLabels, string(workType))

	// Add the project's configured real accounting label too, unless one
	// is already set or the project hasn't configured this mapping.
	if !hasExistingAccountingLabel {
		accountingLabel := opExLabel
		if workType == domain.WorkTypeDevelopment {
			accountingLabel = capExLabel
		}
		if accountingLabel != "" {
			addLabels = append(addLabels, accountingLabel)
		}
	}

	// Add new asset label if available and not preserving existing ones
	if assetResult != nil && assetResult.Asset != nil && !preserveExistingAsset {
		assetLabel := uc.getAssetLabel(assetResult.Asset)
		addLabels = append(addLabels, assetLabel)
	}

	// Remove the new label from removeLabels if it was already there (no-op)
	addSet := make(map[string]bool, len(addLabels))
	for _, l := range addLabels {
		addSet[l] = true
	}
	filtered := removeLabels[:0]
	for _, l := range removeLabels {
		if !addSet[l] {
			filtered = append(filtered, l)
		}
	}
	removeLabels = filtered

	return addLabels, removeLabels
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
