package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// JiraTaskInfo represents a JIRA task with relevant information for team extraction
type JiraTaskInfo struct {
	Key      string
	Assignee string
	Reporter string
	Labels   []string
}

// JiraQueryPort defines the interface for querying JIRA tasks
type JiraQueryPort interface {
	// SearchTasksByLabelPrefix searches for tasks with labels starting with the given prefix
	SearchTasksByLabelPrefix(ctx context.Context, labelPrefix string, maxResults int) ([]JiraTaskInfo, error)
	// SearchTasksWithFilters searches for tasks with additional filtering options
	SearchTasksWithFilters(ctx context.Context, filters JiraSearchFilters) ([]JiraTaskInfo, error)
}

// JiraSearchFilters contains filtering options for JIRA task searches
type JiraSearchFilters struct {
	LabelPrefix string
	ProjectKey  string
	SprintName  string
	TeamName    string
	MaxResults  int
}

// TeamConfigPort defines the interface for team configuration operations
type TeamConfigPort interface {
	// GetTeamForUser returns the team name for a given user (by email or display name)
	GetTeamForUser(userIdentifier string) (string, error)
	// GetAllTeams returns all configured teams
	GetAllTeams() (map[string][]string, error)
}

// SyncAssetContributorsFromJiraUseCase handles syncing asset contributors from JIRA
type SyncAssetContributorsFromJiraUseCase struct {
	assetRepo  ports.AssetRepository
	jiraQuery  JiraQueryPort
	teamConfig TeamConfigPort
}

// NewSyncAssetContributorsFromJiraUseCase creates a new use case instance
func NewSyncAssetContributorsFromJiraUseCase(
	assetRepo ports.AssetRepository,
	jiraQuery JiraQueryPort,
	teamConfig TeamConfigPort,
) *SyncAssetContributorsFromJiraUseCase {
	return &SyncAssetContributorsFromJiraUseCase{
		assetRepo:  assetRepo,
		jiraQuery:  jiraQuery,
		teamConfig: teamConfig,
	}
}

// SyncContributorsInput contains input parameters for syncing contributors
type SyncContributorsInput struct {
	DryRun     bool   `json:"dry_run"`
	MaxResults int    `json:"max_results"`
	ProjectKey string `json:"project_key,omitempty"` // Filter by JIRA project
	SprintName string `json:"sprint_name,omitempty"` // Filter by sprint name
	TeamName   string `json:"team_name,omitempty"`   // Only sync for assets this team works on
	AssetName  string `json:"asset_name,omitempty"`  // Only sync this specific asset
}

// AssetContributorSyncResult represents the result of syncing contributors for an asset
type AssetContributorSyncResult struct {
	AssetName           string   `json:"asset_name"`
	TasksAnalyzed       int      `json:"tasks_analyzed"`
	TeamsFound          []string `json:"teams_found"`
	CurrentContributors []string `json:"current_contributors"`
	NewContributors     []string `json:"new_contributors"`
	RemovedContributors []string `json:"removed_contributors"`
	Updated             bool     `json:"updated"`
	Error               string   `json:"error,omitempty"`
}

// SyncContributorsResult contains the overall result of the sync operation
type SyncContributorsResult struct {
	AssetsProcessed []AssetContributorSyncResult `json:"assets_processed"`
	TotalTasks      int                          `json:"total_tasks"`
	AssetsUpdated   int                          `json:"assets_updated"`
	Errors          []string                     `json:"errors"`
}

// Execute performs the contributor sync operation
func (uc *SyncAssetContributorsFromJiraUseCase) Execute(ctx context.Context, input SyncContributorsInput) (*SyncContributorsResult, error) {
	// Set default max results if not specified
	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = 1000 // Default limit
	}

	// Use new filtered search if we have additional filters
	var tasks []JiraTaskInfo
	var err error

	if input.ProjectKey != "" || input.SprintName != "" || input.TeamName != "" {
		filters := JiraSearchFilters{
			LabelPrefix: "cap-asset-",
			ProjectKey:  input.ProjectKey,
			SprintName:  input.SprintName,
			TeamName:    input.TeamName,
			MaxResults:  maxResults,
		}
		tasks, err = uc.jiraQuery.SearchTasksWithFilters(ctx, filters)
	} else {
		// Fallback to original search for backward compatibility
		tasks, err = uc.jiraQuery.SearchTasksByLabelPrefix(ctx, "cap-asset-", maxResults)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query JIRA tasks: %w", err)
	}

	// Group tasks by asset
	assetTasks := uc.groupTasksByAsset(tasks)

	result := &SyncContributorsResult{
		TotalTasks: len(tasks),
	}

	// If specific asset is requested, filter to only that asset
	if input.AssetName != "" {
		filteredAssetTasks := make(map[string][]JiraTaskInfo)
		for assetLabel, taskList := range assetTasks {
			assetName := uc.extractAssetNameFromLabel(assetLabel)
			convertedName := uc.convertLabelToAssetName(assetLabel)

			if strings.EqualFold(assetName, input.AssetName) || strings.EqualFold(convertedName, input.AssetName) {
				filteredAssetTasks[assetLabel] = taskList
				break
			}
		}
		assetTasks = filteredAssetTasks
	}

	// Process each asset
	for assetLabel, assetTaskList := range assetTasks {
		assetResult := uc.processAssetContributors(ctx, assetLabel, assetTaskList, input.DryRun)
		result.AssetsProcessed = append(result.AssetsProcessed, assetResult)

		if assetResult.Error != "" {
			result.Errors = append(result.Errors, assetResult.Error)
		}
		if assetResult.Updated {
			result.AssetsUpdated++
		}
	}

	return result, nil
}

// groupTasksByAsset groups tasks by their asset labels
func (uc *SyncAssetContributorsFromJiraUseCase) groupTasksByAsset(tasks []JiraTaskInfo) map[string][]JiraTaskInfo {
	assetTasks := make(map[string][]JiraTaskInfo)

	for _, task := range tasks {
		for _, label := range task.Labels {
			if strings.HasPrefix(strings.ToLower(label), "cap-asset-") {
				assetTasks[label] = append(assetTasks[label], task)
				break // Only count each task once per asset
			}
		}
	}

	return assetTasks
}

// processAssetContributors processes contributors for a single asset
func (uc *SyncAssetContributorsFromJiraUseCase) processAssetContributors(
	_ context.Context,
	assetLabel string,
	tasks []JiraTaskInfo,
	dryRun bool,
) AssetContributorSyncResult {
	// Extract asset name from label
	assetName := uc.extractAssetNameFromLabel(assetLabel)

	result := AssetContributorSyncResult{
		AssetName:     assetName,
		TasksAnalyzed: len(tasks),
	}

	// Find the asset
	asset, err := uc.assetRepo.FindByName(assetName)
	if err != nil {
		// Try to find by label format (converted)
		convertedName := uc.convertLabelToAssetName(assetLabel)
		asset, err = uc.assetRepo.FindByName(convertedName)
		if err != nil {
			result.Error = fmt.Sprintf("asset not found: %s (also tried: %s)", assetName, convertedName)
			return result
		}
		result.AssetName = convertedName
	}

	result.CurrentContributors = asset.GetContributingTeams()

	// Extract teams from task assignees/reporters
	teamsFound := uc.extractTeamsFromTasks(tasks)

	result.TeamsFound = teamsFound

	// Calculate changes
	result.NewContributors = uc.findNewTeams(result.CurrentContributors, teamsFound)
	result.RemovedContributors = uc.findRemovedTeams(result.CurrentContributors, teamsFound)

	// Update asset if there are changes and not dry run
	if (len(result.NewContributors) > 0 || len(result.RemovedContributors) > 0) && !dryRun {
		// Don't remove the owning team from contributors
		owningTeam := asset.GetOwningTeam()
		finalTeams := uc.mergeTeams(teamsFound, owningTeam)

		if err := asset.SetContributingTeams(finalTeams); err != nil {
			result.Error = fmt.Sprintf("failed to set contributing teams: %v", err)
			return result
		}

		if err := uc.assetRepo.Save(asset); err != nil {
			result.Error = fmt.Sprintf("failed to save asset: %v", err)
			return result
		}

		result.Updated = true
	}

	return result
}

// extractAssetNameFromLabel converts cap-asset-xyz to a readable asset name
func (uc *SyncAssetContributorsFromJiraUseCase) extractAssetNameFromLabel(label string) string {
	// Remove cap-asset- prefix
	name := strings.TrimPrefix(strings.ToLower(label), "cap-asset-")
	// Convert hyphens to spaces and capitalize first letters
	name = strings.ReplaceAll(name, "-", " ")
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// convertLabelToAssetName tries different name conversion strategies
func (uc *SyncAssetContributorsFromJiraUseCase) convertLabelToAssetName(label string) string {
	// For labels like cap-asset-omio-flex, try to find "Omio Flex"
	name := strings.TrimPrefix(strings.ToLower(label), "cap-asset-")

	// Strategy 1: Capitalize first letters of words
	words := strings.Split(name, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	converted := strings.Join(words, " ")

	// Strategy 2: Keep original casing for known patterns
	if strings.Contains(name, "omio") {
		converted = strings.ReplaceAll(converted, "Omio", "Omio")
	}

	return converted
}

// extractTeamsFromTasks extracts team names from task assignees and reporters
func (uc *SyncAssetContributorsFromJiraUseCase) extractTeamsFromTasks(tasks []JiraTaskInfo) []string {
	userSet := make(map[string]bool)
	teamSet := make(map[string]bool)

	// Collect unique users
	for _, task := range tasks {
		if task.Assignee != "" {
			userSet[task.Assignee] = true
		}
		if task.Reporter != "" {
			userSet[task.Reporter] = true
		}
	}

	// Map users to teams
	for user := range userSet {
		team, err := uc.teamConfig.GetTeamForUser(user)
		if err != nil {
			// If we can't map the user to a team, skip them
			continue
		}
		if team != "" {
			teamSet[team] = true
		}
	}

	// Convert to slice
	teams := make([]string, 0, len(teamSet))
	for team := range teamSet {
		teams = append(teams, team)
	}

	return teams
}

// findNewTeams returns teams that are in found but not in current
func (uc *SyncAssetContributorsFromJiraUseCase) findNewTeams(current, found []string) []string {
	currentSet := make(map[string]bool)
	for _, team := range current {
		currentSet[team] = true
	}

	var newTeams []string
	for _, team := range found {
		if !currentSet[team] {
			newTeams = append(newTeams, team)
		}
	}

	return newTeams
}

// findRemovedTeams returns teams that are in current but not in found
func (uc *SyncAssetContributorsFromJiraUseCase) findRemovedTeams(current, found []string) []string {
	foundSet := make(map[string]bool)
	for _, team := range found {
		foundSet[team] = true
	}

	var removedTeams []string
	for _, team := range current {
		if !foundSet[team] {
			removedTeams = append(removedTeams, team)
		}
	}

	return removedTeams
}

// mergeTeams combines found teams while ensuring owning team is not included in contributors
func (uc *SyncAssetContributorsFromJiraUseCase) mergeTeams(foundTeams []string, owningTeam string) []string {
	teamSet := make(map[string]bool)

	for _, team := range foundTeams {
		if team != owningTeam && team != "" {
			teamSet[team] = true
		}
	}

	teams := make([]string, 0, len(teamSet))
	for team := range teamSet {
		teams = append(teams, team)
	}

	return teams
}
