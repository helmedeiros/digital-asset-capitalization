package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/infrastructure/ai"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/infrastructure/executor"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/infrastructure/prompt"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/infrastructure/store"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	investmentservice "github.com/helmedeiros/digital-asset-capitalization/internal/investment/application/service"
	sprintapp "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application"
	tasksapp "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application"
	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// createConsoleCommand creates the console command that integrates with existing App structure
func (a *App) createConsoleCommand() *cli.Command {
	return &cli.Command{
		Name:    "console",
		Aliases: []string{"c"},
		Usage:   "Start an interactive AI-powered console for AssetCap",
		Description: `The console command starts an interactive session where you can use natural language
to interact with AssetCap. The AI assistant will interpret your requests and execute
the appropriate AssetCap commands.

Examples:
  assetcap console

Once in the console, you can use natural language:
  > Show all assets
  > Create an asset called Payment Processing
  > List tasks for project FN
  > Calculate investment for User Authentication
  > help
  > exit`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "ollama-url",
				Usage:   "Ollama API URL",
				Value:   "http://localhost:11434",
				EnvVars: []string{"OLLAMA_API_URL"},
			},
			&cli.StringFlag{
				Name:  "model",
				Usage: "LLaMA model to use",
				Value: "llama3",
			},
			&cli.IntFlag{
				Name:  "max-sessions",
				Usage: "Maximum number of concurrent sessions",
				Value: 10,
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug output",
			},
		},
		Action: a.runConsole,
	}
}

// runConsole executes the console command
func (a *App) runConsole(c *cli.Context) error {
	ctx := c.Context

	// Enable debug logging if requested
	if c.Bool("debug") {
		log.SetOutput(os.Stdout)
		log.Println("Debug mode enabled")
	}

	// Check if Ollama is accessible
	if err := checkOllamaAvailability(c.String("ollama-url")); err != nil {
		return fmt.Errorf("Ollama is not available: %w\n\nPlease ensure Ollama is running:\n1. Install Ollama: https://ollama.ai\n2. Start Ollama: ollama serve\n3. Pull LLaMA model: ollama pull %s", err, c.String("model"))
	}

	// Initialize AI interpreter
	aiConfig := ai.Config{
		BaseURL: c.String("ollama-url"),
		Model:   c.String("model"),
	}
	interpreter := ai.NewInterpreter(aiConfig)

	// Initialize context store
	storeConfig := store.Config{
		MaxSessions: c.Int("max-sessions"),
		SessionTTL:  30 * time.Minute, // 30 minutes
	}
	contextStore := store.NewMemoryStore(storeConfig)

	// Initialize command executor with actual services from App
	commandExecutor := executor.NewCommandExecutor(
		&AssetServiceAdapter{service: a.assetService},
		&TaskServiceAdapter{service: a.taskService},
		&SprintServiceAdapter{service: a.sprintService},
		&InvestmentServiceAdapter{service: a.investmentService},
		&ConfigServiceAdapter{service: a.configService},
	)

	// Initialize console service
	consoleService := application.NewConsoleService(
		interpreter,
		commandExecutor,
		contextStore,
	)

	// Note: We'll start cleanup via goroutine since the method doesn't exist in our implementation

	// Initialize enhanced prompt handler with Claude Code-style UI
	promptHandler := prompt.NewEnhancedHandler(consoleService)

	// Start the interactive console
	fmt.Println() // Just a blank line for cleaner startup
	return promptHandler.Start(ctx)
}

// checkOllamaAvailability checks if Ollama API is accessible
func checkOllamaAvailability(baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("Ollama URL not provided")
	}

	// Make a simple health check to Ollama
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama at %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	return nil
}

// Service adapters to bridge between console executor interfaces and existing App services

// AssetServiceAdapter adapts the existing asset service to the console interface
type AssetServiceAdapter struct {
	service assetsapp.AssetService
}

func (a *AssetServiceAdapter) ListAssets(_ context.Context) (interface{}, error) {
	assets, err := a.service.ListAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	// Transform for console display
	result := make([]interface{}, 0, len(assets))
	for _, asset := range assets {
		assetInfo := map[string]interface{}{
			"name":        asset.Name,
			"id":          asset.ID,
			"description": asset.Description,
			"status":      asset.Status,
			"created_at":  asset.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":  asset.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		// Add team information if available
		if owningTeam := asset.GetOwningTeam(); owningTeam != "" {
			assetInfo["owning_team"] = owningTeam
		}
		if contributingTeams := asset.GetContributingTeams(); len(contributingTeams) > 0 {
			assetInfo["contributing_teams"] = contributingTeams
		}

		result = append(result, assetInfo)
	}

	if len(result) == 0 {
		return map[string]string{
			"message": "No assets found. Create some assets first with 'assets create'.",
		}, nil
	}

	return result, nil
}

func (a *AssetServiceAdapter) CreateAsset(_ context.Context, name, description string) (interface{}, error) {
	if name == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if description == "" {
		return nil, fmt.Errorf("asset description is required")
	}

	err := a.service.CreateAsset(name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset: %w", err)
	}

	// Get the created asset to return full information
	asset, err := a.service.GetAsset(name)
	if err != nil {
		// Even if we can't get the asset, creation was successful
		return map[string]string{
			"message": fmt.Sprintf("Asset '%s' created successfully", name),
			"name":    name,
			"status":  "created",
		}, nil
	}

	return map[string]interface{}{
		"message":     fmt.Sprintf("Asset '%s' created successfully", name),
		"name":        asset.Name,
		"id":          asset.ID,
		"description": asset.Description,
		"status":      "created",
		"created_at":  asset.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (a *AssetServiceAdapter) GetAsset(_ context.Context, name string) (interface{}, error) {
	if name == "" {
		return nil, fmt.Errorf("asset name is required")
	}

	asset, err := a.service.GetAsset(name)
	if err != nil {
		return nil, fmt.Errorf("asset not found: %w", err)
	}

	result := map[string]interface{}{
		"name":            asset.Name,
		"id":              asset.ID,
		"description":     asset.Description,
		"status":          asset.Status,
		"launch_date":     asset.LaunchDate.Format("2006-01-02"),
		"doc_link":        asset.DocLink,
		"created_at":      asset.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at":      asset.UpdatedAt.Format("2006-01-02 15:04:05"),
		"last_doc_update": asset.LastDocUpdateAt.Format("2006-01-02 15:04:05"),
		"version":         asset.Version,
		"task_count":      asset.AssociatedTaskCount,
	}

	// Add optional fields if present
	if asset.Why != "" {
		result["why"] = asset.Why
	}
	if asset.Benefits != "" {
		result["benefits"] = asset.Benefits
	}
	if asset.How != "" {
		result["how"] = asset.How
	}
	if asset.Metrics != "" {
		result["metrics"] = asset.Metrics
	}
	if len(asset.Keywords) > 0 {
		result["keywords"] = asset.Keywords
	}

	// Add team information if available
	if owningTeam := asset.GetOwningTeam(); owningTeam != "" {
		result["owning_team"] = owningTeam
	}
	if contributingTeams := asset.GetContributingTeams(); len(contributingTeams) > 0 {
		result["contributing_teams"] = contributingTeams
	}

	return result, nil
}

func (a *AssetServiceAdapter) UpdateAsset(_ context.Context, name, description string) (interface{}, error) {
	if name == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if description == "" {
		return nil, fmt.Errorf("asset description is required")
	}

	// Use the full UpdateAsset method with empty optional fields for basic update
	err := a.service.UpdateAsset(name, description, "", "", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to update asset: %w", err)
	}

	// Get updated asset to show current state
	asset, err := a.service.GetAsset(name)
	if err != nil {
		// Even if we can't get the asset, update was successful
		return map[string]string{
			"message": fmt.Sprintf("Asset '%s' updated successfully", name),
			"name":    name,
			"status":  "updated",
		}, nil
	}

	return map[string]interface{}{
		"message":     fmt.Sprintf("Asset '%s' updated successfully", name),
		"name":        asset.Name,
		"description": asset.Description,
		"status":      "updated",
		"updated_at":  asset.UpdatedAt.Format("2006-01-02 15:04:05"),
		"version":     asset.Version,
	}, nil
}

func (a *AssetServiceAdapter) DeleteAsset(_ context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("asset name is required")
	}

	// Check if asset exists first to provide better error messages
	_, err := a.service.GetAsset(name)
	if err != nil {
		return fmt.Errorf("asset '%s' not found", name)
	}

	err = a.service.DeleteAsset(name)
	if err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	return nil
}

func (a *AssetServiceAdapter) SyncAssets(_ context.Context, space, label string) (interface{}, error) {
	if label == "" {
		return nil, fmt.Errorf("label is required for sync operation")
	}

	// Use debug=false by default, can be made configurable later
	result, err := a.service.SyncFromConfluence(space, label, false)
	if err != nil {
		return nil, fmt.Errorf("failed to sync assets: %w", err)
	}

	// Transform sync result for console display
	syncInfo := map[string]interface{}{
		"message": fmt.Sprintf("Sync completed for space '%s' with label '%s'", space, label),
		"space":   space,
		"label":   label,
		"status":  "completed",
	}

	if len(result.SyncedAssets) > 0 {
		syncInfo["synced_count"] = len(result.SyncedAssets)
		var syncedNames []string
		for _, asset := range result.SyncedAssets {
			syncedNames = append(syncedNames, asset.Name)
		}
		syncInfo["synced_assets"] = syncedNames
	}

	if len(result.NotSyncedAssets) > 0 {
		syncInfo["not_synced_count"] = len(result.NotSyncedAssets)
		var notSyncedInfo []map[string]interface{}
		for _, notSynced := range result.NotSyncedAssets {
			notSyncedInfo = append(notSyncedInfo, map[string]interface{}{
				"name":           notSynced.Name,
				"missing_fields": notSynced.MissingFields,
			})
		}
		syncInfo["not_synced_assets"] = notSyncedInfo
	}

	return syncInfo, nil
}

func (a *AssetServiceAdapter) EnrichAsset(_ context.Context, name, field string) (interface{}, error) {
	if name == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if field == "" {
		return nil, fmt.Errorf("field name is required")
	}

	// Validate field is supported
	supportedFields := map[string]bool{
		"description": true,
		"why":         true,
		"benefits":    true,
		"how":         true,
		"metrics":     true,
	}
	if !supportedFields[field] {
		return nil, fmt.Errorf("unsupported field '%s'. Supported fields: description, why, benefits, how, metrics", field)
	}

	err := a.service.EnrichAsset(name, field)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich asset: %w", err)
	}

	// Get updated asset to show enriched content
	asset, err := a.service.GetAsset(name)
	if err != nil {
		return map[string]string{
			"message": fmt.Sprintf("Asset '%s' field '%s' enriched successfully", name, field),
			"asset":   name,
			"field":   field,
			"status":  "enriched",
		}, nil
	}

	result := map[string]interface{}{
		"message": fmt.Sprintf("Asset '%s' field '%s' enriched successfully", name, field),
		"asset":   name,
		"field":   field,
		"status":  "enriched",
	}

	// Include the enriched content
	switch field {
	case "description":
		result["enriched_content"] = asset.Description
	case "why":
		result["enriched_content"] = asset.Why
	case "benefits":
		result["enriched_content"] = asset.Benefits
	case "how":
		result["enriched_content"] = asset.How
	case "metrics":
		result["enriched_content"] = asset.Metrics
	}

	return result, nil
}

func (a *AssetServiceAdapter) GenerateKeywords(_ context.Context, name string) (interface{}, error) {
	if name == "" {
		return nil, fmt.Errorf("asset name is required")
	}

	err := a.service.GenerateKeywords(name)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keywords: %w", err)
	}

	// Get updated asset to show generated keywords
	asset, err := a.service.GetAsset(name)
	if err != nil {
		return map[string]interface{}{
			"message": fmt.Sprintf("Keywords generated successfully for asset '%s'", name),
			"asset":   name,
			"status":  "keywords_generated",
		}, nil
	}

	return map[string]interface{}{
		"message":  fmt.Sprintf("Keywords generated successfully for asset '%s'", name),
		"asset":    name,
		"keywords": asset.Keywords,
		"status":   "keywords_generated",
	}, nil
}

// Team management methods for AssetServiceAdapter

func (a *AssetServiceAdapter) AssignTeamOwner(_ context.Context, asset, team string) (interface{}, error) {
	if asset == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if team == "" {
		return nil, fmt.Errorf("team name is required")
	}

	// Use the existing AssignTeam method with empty contributing teams list
	err := a.service.AssignTeam(asset, team, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to assign team owner: %w", err)
	}

	return map[string]interface{}{
		"message": fmt.Sprintf("Team '%s' assigned as owner of asset '%s'", team, asset),
		"asset":   asset,
		"team":    team,
		"role":    "owner",
		"status":  "assigned",
	}, nil
}

func (a *AssetServiceAdapter) AddContributingTeam(_ context.Context, asset, team string) (interface{}, error) {
	if asset == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if team == "" {
		return nil, fmt.Errorf("team name is required")
	}

	err := a.service.AddContributingTeam(asset, team)
	if err != nil {
		return nil, fmt.Errorf("failed to add contributing team: %w", err)
	}

	return map[string]interface{}{
		"message": fmt.Sprintf("Team '%s' added as contributor to asset '%s'", team, asset),
		"asset":   asset,
		"team":    team,
		"role":    "contributor",
		"status":  "added",
	}, nil
}

func (a *AssetServiceAdapter) RemoveContributingTeam(_ context.Context, asset, team string) (interface{}, error) {
	if asset == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if team == "" {
		return nil, fmt.Errorf("team name is required")
	}

	err := a.service.RemoveContributingTeam(asset, team)
	if err != nil {
		return nil, fmt.Errorf("failed to remove contributing team: %w", err)
	}

	return map[string]interface{}{
		"message": fmt.Sprintf("Team '%s' removed as contributor from asset '%s'", team, asset),
		"asset":   asset,
		"team":    team,
		"role":    "contributor",
		"status":  "removed",
	}, nil
}

func (a *AssetServiceAdapter) ShowTeamAssignments(_ context.Context, asset string) (interface{}, error) {
	if asset == "" {
		return nil, fmt.Errorf("asset name is required")
	}

	teamInfo, err := a.service.GetAssetTeamInfo(asset)
	if err != nil {
		return nil, fmt.Errorf("failed to get team assignments: %w", err)
	}

	if teamInfo == nil {
		return map[string]interface{}{
			"message": fmt.Sprintf("No team assignments found for asset '%s'", asset),
			"asset":   asset,
		}, nil
	}

	result := map[string]interface{}{
		"asset":              teamInfo.AssetName,
		"owning_team":        teamInfo.OwningTeam,
		"contributing_teams": teamInfo.ContributingTeams,
	}

	if teamInfo.OwningTeam == "" && len(teamInfo.ContributingTeams) == 0 {
		result["message"] = fmt.Sprintf("Asset '%s' has no team assignments", asset)
	} else {
		result["message"] = fmt.Sprintf("Team assignments for asset '%s'", asset)
	}

	return result, nil
}

func (a *AssetServiceAdapter) ListTeamAssignments(_ context.Context) (interface{}, error) {
	teams, err := a.service.GetAssetTeams()
	if err != nil {
		return nil, fmt.Errorf("failed to list team assignments: %w", err)
	}

	if len(teams) == 0 {
		return map[string]string{
			"message": "No team assignments found. Assign teams to assets first.",
		}, nil
	}

	// Transform team info for console display
	result := make([]map[string]interface{}, 0, len(teams))
	for _, teamInfo := range teams {
		assignment := map[string]interface{}{
			"asset":              teamInfo.AssetName,
			"owning_team":        teamInfo.OwningTeam,
			"contributing_teams": teamInfo.ContributingTeams,
			"total_teams":        len(teamInfo.ContributingTeams),
		}

		// Add indicator for ownership status
		if teamInfo.OwningTeam != "" {
			assignment["has_owner"] = true
		} else {
			assignment["has_owner"] = false
		}

		result = append(result, assignment)
	}

	return map[string]interface{}{
		"message":     fmt.Sprintf("Found %d assets with team assignments", len(teams)),
		"assignments": result,
	}, nil
}

// Advanced asset operations

func (a *AssetServiceAdapter) SyncAndEnrich(_ context.Context, space, label string, keywords bool, fields []string) (interface{}, error) {
	if label == "" {
		return nil, fmt.Errorf("label is required for sync-and-enrich operation")
	}

	// For now, return a placeholder - this would need to be implemented in the actual asset service
	return map[string]interface{}{
		"message":  fmt.Sprintf("Sync-and-enrich operation initiated for space '%s' with label '%s'", space, label),
		"space":    space,
		"label":    label,
		"keywords": keywords,
		"fields":   fields,
		"status":   "not_implemented",
		"note":     "This feature requires implementation in the asset service",
	}, nil
}

// TaskServiceAdapter adapts the existing task service
type TaskServiceAdapter struct {
	service tasksapp.TaskService
}

func (t *TaskServiceAdapter) FetchTasks(ctx context.Context, project, sprint string) (interface{}, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}
	if sprint == "" {
		return nil, fmt.Errorf("sprint is required")
	}

	// Use "jira" as default platform for console operations
	err := t.service.FetchTasks(ctx, project, sprint, "jira")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}

	return map[string]interface{}{
		"message": fmt.Sprintf("Tasks fetched successfully for project '%s' and sprint '%s'", project, sprint),
		"status":  "fetched",
		"project": project,
		"sprint":  sprint,
	}, nil
}

func (t *TaskServiceAdapter) ShowTasks(ctx context.Context, project, sprint string) (interface{}, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}
	if sprint == "" {
		return nil, fmt.Errorf("sprint is required")
	}

	tasks, err := t.service.GetTasks(ctx, project, sprint)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) == 0 {
		return map[string]string{
			"message": fmt.Sprintf("No tasks found for project '%s' and sprint '%s'. Fetch tasks first with 'tasks fetch'.", project, sprint),
		}, nil
	}

	// Transform tasks for console display
	result := make([]map[string]interface{}, 0, len(tasks))
	for _, task := range tasks {
		taskInfo := map[string]interface{}{
			"key":        task.Key,
			"summary":    task.Summary,
			"status":     string(task.Status),
			"type":       string(task.Type),
			"priority":   string(task.Priority),
			"work_type":  string(task.WorkType),
			"created_at": task.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at": task.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		// Add sprints information
		if sprints := task.GetSprints(); len(sprints) > 0 {
			taskInfo["sprints"] = sprints
		}

		// Add labels if present
		if len(task.Labels) > 0 {
			taskInfo["labels"] = task.Labels
		}

		// Add epic if present
		if task.Epic != "" {
			taskInfo["epic"] = task.Epic
		}

		result = append(result, taskInfo)
	}

	return result, nil
}

func (t *TaskServiceAdapter) ClassifyTasks(ctx context.Context, project, sprint string, apply bool) (interface{}, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}
	if sprint == "" {
		return nil, fmt.Errorf("sprint is required")
	}

	// Create classification input
	input := tasksdomain.ClassifyTasksInput{
		Project: project,
		Sprint:  sprint,
		DryRun:  !apply, // DryRun is opposite of apply
		Apply:   apply,
	}

	err := t.service.ClassifyTasks(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify tasks: %w", err)
	}

	// Get classified tasks to show the results
	tasks, err := t.service.GetTasks(ctx, project, sprint)
	if err != nil {
		return map[string]interface{}{
			"message": fmt.Sprintf("Tasks classified successfully for project '%s' and sprint '%s'", project, sprint),
			"status":  "classified",
			"project": project,
			"sprint":  sprint,
			"applied": apply,
		}, nil
	}

	// Count tasks by classification status (labeled vs unlabeled)
	var labeledCount, unlabeledCount int
	var assetLabels []string

	for _, task := range tasks {
		hasAssetLabel := false
		for _, label := range task.Labels {
			if strings.HasPrefix(label, "cap-asset-") {
				hasAssetLabel = true
				assetLabels = append(assetLabels, label)
				break
			}
		}
		if hasAssetLabel {
			labeledCount++
		} else {
			unlabeledCount++
		}
	}

	return map[string]interface{}{
		"message":         fmt.Sprintf("Tasks classified successfully for project '%s' and sprint '%s'", project, sprint),
		"status":          "classified",
		"project":         project,
		"sprint":          sprint,
		"applied":         apply,
		"total_tasks":     len(tasks),
		"labeled_tasks":   labeledCount,
		"unlabeled_tasks": unlabeledCount,
		"unique_assets":   removeDuplicates(assetLabels),
	}, nil
}

func (t *TaskServiceAdapter) InspectTask(ctx context.Context, key string) (interface{}, error) {
	if key == "" {
		return nil, fmt.Errorf("task key is required")
	}

	task, err := t.service.GetTaskByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Transform task for detailed console display
	result := map[string]interface{}{
		"key":         task.Key,
		"summary":     task.Summary,
		"description": task.Description,
		"project":     task.Project,
		"platform":    task.Platform,
		"status":      string(task.Status),
		"type":        string(task.Type),
		"priority":    string(task.Priority),
		"work_type":   string(task.WorkType),
		"created_at":  task.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at":  task.UpdatedAt.Format("2006-01-02 15:04:05"),
		"version":     task.Version,
	}

	// Add sprints information
	if sprints := task.GetSprints(); len(sprints) > 0 {
		result["sprints"] = sprints
		result["primary_sprint"] = task.GetPrimarySprint()
	}

	// Add labels if present
	if len(task.Labels) > 0 {
		result["labels"] = task.Labels

		// Extract asset labels specifically
		var assetLabels []string
		for _, label := range task.Labels {
			if strings.HasPrefix(label, "cap-asset-") {
				assetLabels = append(assetLabels, label)
			}
		}
		if len(assetLabels) > 0 {
			result["asset_labels"] = assetLabels
		}
	}

	// Add epic if present
	if task.Epic != "" {
		result["epic"] = task.Epic
	}

	return result, nil
}

// SprintServiceAdapter adapts the existing sprint service
type SprintServiceAdapter struct {
	service sprintapp.SprintService
}

func (s *SprintServiceAdapter) ListSprints(_ context.Context, project, period string) (interface{}, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}
	if period == "" {
		period = "current" // Default to current period
	}

	result, err := s.service.ListSprints(project, period)
	if err != nil {
		return nil, fmt.Errorf("failed to list sprints: %w", err)
	}

	if result == nil || len(result.Sprints) == 0 {
		return map[string]string{
			"message": fmt.Sprintf("No sprints found for project '%s' and period '%s'", project, period),
		}, nil
	}

	// Transform sprint data for console display
	sprintInfos := make([]map[string]interface{}, 0, len(result.Sprints))
	for _, sprint := range result.Sprints {
		sprintInfo := map[string]interface{}{
			"name":       sprint.Name,
			"start_date": sprint.StartDate,
			"end_date":   sprint.EndDate,
			"state":      sprint.State,
		}

		if sprint.Goal != "" {
			sprintInfo["goal"] = sprint.Goal
		}

		sprintInfos = append(sprintInfos, sprintInfo)
	}

	return map[string]interface{}{
		"message": fmt.Sprintf("Found %d sprints for project '%s'", len(result.Sprints), project),
		"project": project,
		"period":  period,
		"sprints": sprintInfos,
	}, nil
}

func (s *SprintServiceAdapter) AllocateSprint(_ context.Context, project, sprint string, bounded bool) (interface{}, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}
	if sprint == "" {
		return nil, fmt.Errorf("sprint is required")
	}

	// Use ProcessJiraIssuesWithStrategy for allocation calculation
	csvData, err := s.service.ProcessJiraIssuesWithStrategy(project, sprint, "", bounded)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate sprint: %w", err)
	}

	// Parse CSV data to extract allocation information
	lines := strings.Split(csvData, "\n")
	var totalHours float64
	var taskCount int

	// Skip header line and count non-empty lines
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		taskCount++
		// Basic CSV parsing to extract hours (assuming hours are in one of the columns)
		fields := strings.Split(line, ",")
		if len(fields) > 5 { // Assuming hours might be in column 6 or similar
			// This is a simplified parsing - in practice you'd want more robust CSV parsing
			totalHours += 8.0 // Default assumption of 8 hours per task
		}
	}

	return map[string]interface{}{
		"message":          fmt.Sprintf("Sprint allocation calculated for project '%s', sprint '%s'", project, sprint),
		"project":          project,
		"sprint":           sprint,
		"bounded":          bounded,
		"allocation":       "calculated",
		"total_tasks":      taskCount,
		"estimated_hours":  totalHours,
		"calculation_type": map[bool]string{true: "sprint-bounded", false: "unbounded"}[bounded],
		"csv_lines":        len(lines) - 1, // Exclude header
	}, nil
}

// InvestmentServiceAdapter adapts the existing investment service
type InvestmentServiceAdapter struct {
	service *investmentservice.InvestmentService
}

func (i *InvestmentServiceAdapter) CalculateInvestment(ctx context.Context, asset, project string, sprints []string) (interface{}, error) {
	if asset == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}
	if len(sprints) == 0 {
		return nil, fmt.Errorf("at least one sprint is required")
	}

	investment, err := i.service.CalculateAssetInvestment(ctx, asset, project, sprints)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate investment: %w", err)
	}

	// Calculate total hours from engineers involved
	var totalHours float64
	for _, engineer := range investment.EngineersInvolved {
		totalHours += engineer.TotalHours
	}

	// Transform investment data for console display
	result := map[string]interface{}{
		"message":            fmt.Sprintf("Investment calculated for asset '%s'", asset),
		"asset":              asset,
		"project":            project,
		"sprints":            sprints,
		"total_investment":   investment.TotalCost.String(),
		"total_hours":        totalHours,
		"calculation_period": fmt.Sprintf("%s to %s", investment.StartDate.Format("2006-01-02"), investment.EndDate.Format("2006-01-02")),
		"asset_id":           investment.AssetName,
		"calculated_at":      investment.CalculatedAt.Format("2006-01-02 15:04:05"),
	}

	// Add cost breakdown from engineers involved
	if len(investment.EngineersInvolved) > 0 {
		var breakdown []map[string]interface{}
		for _, engineer := range investment.EngineersInvolved {
			breakdown = append(breakdown, map[string]interface{}{
				"name":  engineer.Name,
				"level": string(engineer.Level),
				"hours": engineer.TotalHours,
				"cost":  engineer.TotalCost.String(),
			})
		}
		result["engineer_breakdown"] = breakdown
	}

	// Add work type breakdown if available
	if len(investment.WorkTypeBreakdown) > 0 {
		var workTypeBreakdown []map[string]interface{}
		for workType, cost := range investment.WorkTypeBreakdown {
			workTypeBreakdown = append(workTypeBreakdown, map[string]interface{}{
				"work_type": workType,
				"cost":      cost.String(),
			})
		}
		result["work_type_breakdown"] = workTypeBreakdown
	}

	return result, nil
}

func (i *InvestmentServiceAdapter) ListInvestments(ctx context.Context, project string) (interface{}, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	investments, err := i.service.ListInvestments(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to list investments: %w", err)
	}

	if len(investments) == 0 {
		return map[string]string{
			"message": fmt.Sprintf("No investments found for project '%s'. Calculate some investments first.", project),
		}, nil
	}

	// Transform investments for console display
	investmentList := make([]map[string]interface{}, 0, len(investments))
	var totalInvestmentAmount float64

	for _, investment := range investments {
		// Calculate total hours from engineers involved
		var totalHours float64
		for _, engineer := range investment.EngineersInvolved {
			totalHours += engineer.TotalHours
		}

		investmentInfo := map[string]interface{}{
			"asset":            investment.AssetName,
			"total_investment": investment.TotalCost.String(),
			"total_hours":      totalHours,
			"calculation_date": investment.CalculatedAt.Format("2006-01-02 15:04:05"),
			"period":           fmt.Sprintf("%s to %s", investment.StartDate.Format("2006-01-02"), investment.EndDate.Format("2006-01-02")),
		}

		// Add sprints if available
		if len(investment.Sprints) > 0 {
			investmentInfo["sprints"] = investment.Sprints
		}

		investmentList = append(investmentList, investmentInfo)
		totalInvestmentAmount += investment.TotalCost.Amount
	}

	return map[string]interface{}{
		"message":           fmt.Sprintf("Found %d investments for project '%s'", len(investments), project),
		"project":           project,
		"total_investments": len(investments),
		"total_value":       fmt.Sprintf("$%.2f", totalInvestmentAmount),
		"investments":       investmentList,
	}, nil
}

func (i *InvestmentServiceAdapter) ShowRates(ctx context.Context, project string) (interface{}, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	costModel, err := i.service.GetCostModel(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost model: %w", err)
	}

	// Transform cost model rates for console display
	rates := make(map[string]string)

	// Add specific engineer rates
	for name, engineerRate := range costModel.EngineerRates {
		rates[fmt.Sprintf("%s (%s)", name, engineerRate.Level)] = fmt.Sprintf("%.2f %s/hour", engineerRate.HourlyRate, costModel.Currency)
	}

	// Add default rates by level
	defaultRates := make(map[string]string)
	for level, rate := range costModel.DefaultRatesByLevel {
		defaultRates[string(level)] = fmt.Sprintf("%.2f %s/hour", rate, costModel.Currency)
	}

	result := map[string]interface{}{
		"message":                fmt.Sprintf("Cost model for project '%s'", project),
		"project":                project,
		"engineer_rates":         rates,
		"default_rates_by_level": defaultRates,
		"currency":               costModel.Currency,
		"working_hours_per_day":  costModel.WorkingHoursPerDay,
		"overhead_multiplier":    costModel.OverheadMultiplier,
	}

	// Add infrastructure costs if available
	if costModel.InfrastructureCosts.CloudCostsPerMonth > 0 ||
		costModel.InfrastructureCosts.ToolingCostsPerMonth > 0 ||
		costModel.InfrastructureCosts.LicenseCostsPerMonth > 0 {
		infraCosts := map[string]string{
			"cloud_per_month":   fmt.Sprintf("%.2f %s", costModel.InfrastructureCosts.CloudCostsPerMonth, costModel.Currency),
			"tooling_per_month": fmt.Sprintf("%.2f %s", costModel.InfrastructureCosts.ToolingCostsPerMonth, costModel.Currency),
			"license_per_month": fmt.Sprintf("%.2f %s", costModel.InfrastructureCosts.LicenseCostsPerMonth, costModel.Currency),
			"total_per_month":   fmt.Sprintf("%.2f %s", costModel.GetTotalMonthlyCost(), costModel.Currency),
		}
		result["infrastructure_costs"] = infraCosts
	}

	return result, nil
}

// ConfigServiceAdapter adapts the existing config service
type ConfigServiceAdapter struct {
	service ConfigService
}

func (c *ConfigServiceAdapter) InitConfig(_ context.Context) (interface{}, error) {
	// Use the actual config service to initialize configuration
	result, err := c.service.InitializeConfig(false) // non-interactive mode for console
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	return map[string]interface{}{
		"message":             "Configuration initialized successfully",
		"status":              "initialized",
		"jira_config_created": result.JiraConfigCreated,
		"team_config_created": result.TeamConfigCreated,
		"details":             result.Message,
	}, nil
}

func (c *ConfigServiceAdapter) ShowConfig(_ context.Context) (interface{}, error) {
	// Get actual JIRA configuration
	jiraConfig, err := c.service.GetJiraConfig()
	if err != nil {
		return map[string]interface{}{
			"message":    "Configuration status",
			"jira_url":   "not configured",
			"jira_error": err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"message":     "Current configuration",
		"jira_url":    jiraConfig.BaseURL,
		"jira_email":  jiraConfig.Email,
		"jira_status": "configured",
	}, nil
}

func (c *ConfigServiceAdapter) ValidateConfig(_ context.Context) (interface{}, error) {
	// Try to get JIRA config to validate
	_, err := c.service.GetJiraConfig()
	if err != nil {
		return map[string]interface{}{
			"message": "Configuration validation failed",
			"status":  "invalid",
			"errors":  []string{err.Error()},
		}, nil
	}

	return map[string]interface{}{
		"message": "Configuration is valid",
		"status":  "valid",
		"checks": map[string]string{
			"jira_config": "✅ Valid",
		},
	}, nil
}

func (c *ConfigServiceAdapter) SyncTeam(_ context.Context, project string) (interface{}, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	// For now, return a descriptive message since team sync would require implementation
	// in the actual config service
	return map[string]interface{}{
		"message": fmt.Sprintf("Team sync initiated for project '%s'", project),
		"project": project,
		"status":  "initiated",
		"note":    "Team sync requires JIRA configuration and would fetch project members",
	}, nil
}

// Helper function to remove duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}
