package infrastructure

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
	sharedjira "github.com/helmedeiros/digital-asset-capitalization/internal/shared/jira"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/config"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// JiraAdapter implements the JiraPort interface
type JiraAdapter struct {
	config        *config.JiraConfig
	teams         domain.TeamMap
	httpClient    *HTTPClient
	configService *service.ConfigService
	fieldIDs      *sharedjira.CustomFieldIDs
}

// NewJiraAdapter creates a new Jira adapter using shared configuration
func NewJiraAdapter() (*JiraAdapter, error) {
	// Initialize configuration service with file repository
	configRepo := infrastructure.NewFileRepository(".assetcap")
	configService := service.NewConfigService(configRepo)

	// Load Jira configuration - fallback to legacy approach if new config doesn't exist
	jiraConfig, err := loadJiraConfig(configService)
	if err != nil {
		return nil, fmt.Errorf("failed to load Jira configuration: %w", err)
	}

	// Load teams data using shared configuration service
	teams, err := configService.GetTeamMapForSprint()
	if err != nil {
		return nil, fmt.Errorf("failed to load team configuration: %w", err)
	}

	// Create HTTP client
	httpClient := NewHTTPClient(jiraConfig.GetBaseURL(), jiraConfig.GetAuthHeader())

	// Resolve custom field IDs for TPD Business Unit, Work Stream, Engineering Hours
	fieldResolver := sharedjira.NewFieldResolver(jiraConfig.GetBaseURL(), jiraConfig.GetAuthHeader())
	fieldIDs, err := fieldResolver.ResolveCustomFieldIDs()
	if err != nil {
		// Non-fatal: log warning and continue without custom fields
		fmt.Printf("Warning: could not resolve custom field IDs: %v\n", err)
		fieldIDs = &sharedjira.CustomFieldIDs{}
	}

	return &JiraAdapter{
		config:        jiraConfig,
		teams:         teams,
		httpClient:    httpClient,
		configService: configService,
		fieldIDs:      fieldIDs,
	}, nil
}

// NewJiraAdapterLegacy creates a new Jira adapter using legacy file path (for backward compatibility)
// Deprecated: Use NewJiraAdapter() instead
func NewJiraAdapterLegacy(_ string) (*JiraAdapter, error) {
	return NewJiraAdapter()
}

// loadJiraConfig loads Jira configuration, falling back to legacy approach if needed
func loadJiraConfig(configService *service.ConfigService) (*config.JiraConfig, error) {
	// Try to load from new configuration system first
	jiraConfig, err := configService.GetJiraConfig()
	if err == nil {
		// Set environment variables temporarily for legacy config creation
		origBaseURL := os.Getenv("JIRA_BASE_URL")
		origEmail := os.Getenv("JIRA_EMAIL")
		origToken := os.Getenv("JIRA_TOKEN")

		// Temporarily set environment variables
		os.Setenv("JIRA_BASE_URL", jiraConfig.BaseURL())
		os.Setenv("JIRA_EMAIL", jiraConfig.Email())
		os.Setenv("JIRA_TOKEN", jiraConfig.Token())

		// Create legacy config
		legacyConfig, createErr := config.NewJiraConfig()

		// Restore original environment variables
		os.Setenv("JIRA_BASE_URL", origBaseURL)
		os.Setenv("JIRA_EMAIL", origEmail)
		os.Setenv("JIRA_TOKEN", origToken)

		if createErr != nil {
			return nil, fmt.Errorf("failed to create legacy config from new configuration: %w", createErr)
		}

		return legacyConfig, nil
	}

	// Fall back to legacy environment-based configuration
	legacyConfig, legacyErr := config.NewJiraConfig()
	if legacyErr != nil {
		return nil, fmt.Errorf("failed to load configuration from both new system and legacy environment: new=%v, legacy=%v", err, legacyErr)
	}

	return legacyConfig, nil
}

// buildFieldsParam returns the fields parameter including any discovered custom field IDs
func (a *JiraAdapter) buildFieldsParam() string {
	fields := "summary,assignee,status,changelog,issuetype,customfield_10014,customfield_10015,labels"
	if a.fieldIDs != nil {
		if a.fieldIDs.TPDBusinessUnit != "" {
			fields += "," + a.fieldIDs.TPDBusinessUnit
		}
		if a.fieldIDs.EngineeringHours != "" {
			fields += "," + a.fieldIDs.EngineeringHours
		}
		if a.fieldIDs.WorkStream != "" {
			fields += "," + a.fieldIDs.WorkStream
		}
	}
	return fields
}

// enrichIssues enriches domain issues with custom field data
func (a *JiraAdapter) enrichIssues(issues []domain.JiraIssue) {
	if a.fieldIDs == nil {
		return
	}
	for i := range issues {
		issues[i].EnrichCustomFields(*a.fieldIDs)
	}
}

// GetIssuesForSprint retrieves all issues for a given sprint
func (a *JiraAdapter) GetIssuesForSprint(project, sprintID string) ([]ports.JiraIssue, error) {
	query := fmt.Sprintf("project = %s AND sprint = '%s'", project, sprintID)
	encodedQuery := url.QueryEscape(query)
	fields := a.buildFieldsParam()
	jiraURL := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&expand=changelog&fields=%s",
		a.config.GetBaseURL(), encodedQuery, fields)

	issues, err := a.httpClient.GetJiraIssues(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprint issues: %w", err)
	}

	a.enrichIssues(issues)
	return a.convertToPortIssues(issues), nil
}

// GetIssuesForTeamMember retrieves all issues assigned to a team member
func (a *JiraAdapter) GetIssuesForTeamMember(member string) ([]ports.JiraIssue, error) {
	query := fmt.Sprintf("assignee = '%s'", member)
	encodedQuery := url.QueryEscape(query)
	fields := a.buildFieldsParam()
	jiraURL := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&expand=changelog&fields=%s",
		a.config.GetBaseURL(), encodedQuery, fields)

	issues, err := a.httpClient.GetJiraIssues(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch team member issues: %w", err)
	}

	a.enrichIssues(issues)
	return a.convertToPortIssues(issues), nil
}

// GetSprintIssues retrieves all issues in a sprint
func (a *JiraAdapter) GetSprintIssues(sprint *domain.Sprint) ([]ports.JiraIssue, error) {
	issues, err := a.GetIssuesForSprint(sprint.Project, sprint.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get issues for sprint %s: %w", sprint.ID, err)
	}

	portIssues := make([]ports.JiraIssue, 0, len(issues))
	portIssues = append(portIssues, issues...)
	return portIssues, nil
}

// GetTeamIssues retrieves all issues for a team
func (a *JiraAdapter) GetTeamIssues(team *domain.Team) ([]ports.JiraIssue, error) {
	var allIssues []ports.JiraIssue

	// Get issues for each team member
	for _, member := range team.Team {
		issues, err := a.GetIssuesForTeamMember(member)
		if err != nil {
			return nil, fmt.Errorf("failed to get issues for team member %s: %w", member, err)
		}
		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}

// convertChangelog converts domain changelog to ports changelog
func convertChangelog(changelog domain.JiraChangelog) ports.JiraChangelog {
	portChangelog := ports.JiraChangelog{
		Histories: make([]ports.JiraChangeHistory, len(changelog.Histories)),
	}

	// Convert changelog histories
	for i, history := range changelog.Histories {
		portHistory := ports.JiraChangeHistory{
			Created: history.Created,
			Items:   make([]ports.JiraChangeItem, len(history.Items)),
		}

		// Convert changelog items
		for j, item := range history.Items {
			portHistory.Items[j] = ports.JiraChangeItem{
				Field:      item.Field,
				FromString: item.FromString,
				ToString:   item.ToString,
			}
		}

		portChangelog.Histories[i] = portHistory
	}

	return portChangelog
}

// convertToPortIssues converts domain JiraIssue to port JiraIssue
func (a *JiraAdapter) convertToPortIssues(issues []domain.JiraIssue) []ports.JiraIssue {
	var portIssues = make([]ports.JiraIssue, 0, len(issues))

	for _, issue := range issues {
		portIssue := ports.JiraIssue{
			Key:              issue.Key,
			Summary:          issue.Fields.Summary,
			Assignee:         issue.Fields.Assignee.DisplayName,
			Status:           issue.Fields.Status.Name,
			StoryPoints:      issue.Fields.StoryPoints,
			IssueType:        issue.Fields.IssueType.Name,
			Labels:           issue.Fields.Labels,
			Changelog:        convertChangelog(issue.Changelog),
			TPDBusinessUnits: issue.Fields.TPDBusinessUnits,
			EngineeringHours: issue.Fields.EngineeringHours,
			WorkStream:       issue.Fields.WorkStream,
		}

		portIssues = append(portIssues, portIssue)
	}

	return portIssues
}

// GetSprintsForProject retrieves all sprints for a given project
func (a *JiraAdapter) GetSprintsForProject(project string, states []string) ([]ports.Sprint, error) {
	// First, get all boards for the project
	boards, err := a.getBoardsForProject(project)
	if err != nil {
		return nil, fmt.Errorf("failed to get boards for project %s: %w", project, err)
	}

	var allSprints []ports.Sprint

	// Get sprints from each board
	for _, board := range boards {
		sprints, err := a.getSprintsForBoard(board.ID, states)
		if err != nil {
			// Skip Kanban boards silently (they don't support sprints)
			errMsg := err.Error()
			if strings.Contains(errMsg, "board does not support sprints") || strings.Contains(errMsg, "Board does not support sprints") {
				continue
			}
			// Log other errors but continue with other boards
			fmt.Printf("Warning: failed to get sprints for board %d: %v\n", board.ID, err)
			continue
		}
		allSprints = append(allSprints, sprints...)
	}

	return allSprints, nil
}

// GetSprintsForProjectWithBoardInfo returns sprints along with board information
func (a *JiraAdapter) GetSprintsForProjectWithBoardInfo(project string, states []string) ([]ports.Sprint, []ports.BoardInfo, error) {
	// First, get all boards for the project
	boards, err := a.getBoardsForProject(project)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get boards for project %s: %w", project, err)
	}

	var allSprints []ports.Sprint
	boardInfo := make([]ports.BoardInfo, 0, len(boards))

	// Get sprints from each board and collect board information
	for _, board := range boards {
		sprints, err := a.getSprintsForBoard(board.ID, states)
		hasSprints := err == nil && len(sprints) > 0

		// Add board info
		boardInfo = append(boardInfo, ports.BoardInfo{
			ID:         board.ID,
			Name:       board.Name,
			Type:       board.Type,
			HasSprints: hasSprints,
		})

		if err != nil {
			// Skip Kanban boards silently (they don't support sprints)
			errMsg := err.Error()
			if strings.Contains(errMsg, "board does not support sprints") || strings.Contains(errMsg, "Board does not support sprints") {
				continue
			}
			// Don't log warning here - we'll handle it in the formatter
			continue
		}
		allSprints = append(allSprints, sprints...)
	}

	return allSprints, boardInfo, nil
}

// getBoardsForProject retrieves all boards for a given project
func (a *JiraAdapter) getBoardsForProject(project string) ([]Board, error) {
	url := fmt.Sprintf("%s/rest/agile/1.0/board?projectKeyOrId=%s", a.config.GetBaseURL(), project)

	boards, err := a.httpClient.GetBoards(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch boards: %w", err)
	}

	return boards, nil
}

// getSprintsForBoard retrieves sprints for a given board
func (a *JiraAdapter) getSprintsForBoard(boardID int, states []string) ([]ports.Sprint, error) {
	var url string
	if len(states) == 0 {
		// Fetch all sprints without state filter
		url = fmt.Sprintf("%s/rest/agile/1.0/board/%d/sprint", a.config.GetBaseURL(), boardID)
	} else {
		statesParam := strings.Join(states, ",")
		url = fmt.Sprintf("%s/rest/agile/1.0/board/%d/sprint?state=%s", a.config.GetBaseURL(), boardID, statesParam)
	}

	sprints, err := a.httpClient.GetSprints(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprints for board %d: %w", boardID, err)
	}

	// Convert to ports.Sprint
	portSprints := make([]ports.Sprint, 0, len(sprints))
	for _, sprint := range sprints {
		portSprint := ports.Sprint{
			ID:        sprint.ID,
			Name:      sprint.Name,
			State:     sprint.State,
			StartDate: sprint.StartDate,
			EndDate:   sprint.EndDate,
			Goal:      sprint.Goal,
		}
		portSprints = append(portSprints, portSprint)
	}

	return portSprints, nil
}

// GetSprintByName retrieves sprint details by project and sprint name
func (a *JiraAdapter) GetSprintByName(project, sprintName string) (*ports.Sprint, error) {
	// Get all sprints for the project
	sprints, err := a.GetSprintsForProject(project, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get sprints for project %s: %w", project, err)
	}

	// Find sprint by name
	for _, sprint := range sprints {
		if sprint.Name == sprintName {
			return &sprint, nil
		}
	}

	return nil, fmt.Errorf("sprint '%s' not found in project %s", sprintName, project)
}

// GetIssuesForSprintOnBoard retrieves issues for a sprint filtered to a specific board.
// For scrum boards, it finds the sprint by name on that board and fetches its issues.
// For kanban boards (no sprints), it fetches board issues updated within the sprint date range.
func (a *JiraAdapter) GetIssuesForSprintOnBoard(project, sprintName string, boardID int) ([]ports.JiraIssue, error) {
	// Try to find the sprint on this board
	sprints, err := a.getSprintsForBoard(boardID, []string{})
	if err != nil {
		// Kanban board — no sprints. Fall back to date-range query using sprint dates from other boards.
		sprintDetails, lookupErr := a.GetSprintByName(project, sprintName)
		if lookupErr != nil {
			return nil, fmt.Errorf("failed to get sprint details for date range: %w", lookupErr)
		}
		return a.getIssuesForBoardByDateRange(project, boardID, sprintDetails.StartDate, sprintDetails.EndDate)
	}

	// Find matching sprint on this board
	var sprintID string
	for _, s := range sprints {
		if s.Name == sprintName {
			sprintID = s.ID
			break
		}
	}

	if sprintID == "" {
		// Sprint not on this board — try date-range approach
		sprintDetails, lookupErr := a.GetSprintByName(project, sprintName)
		if lookupErr != nil {
			return nil, fmt.Errorf("sprint '%s' not found on board %d and failed date range fallback: %w", sprintName, boardID, lookupErr)
		}
		return a.getIssuesForBoardByDateRange(project, boardID, sprintDetails.StartDate, sprintDetails.EndDate)
	}

	// Scrum board with matching sprint — use sprint-filtered JQL
	query := fmt.Sprintf("project = %s AND sprint = '%s'", project, sprintName)
	encodedQuery := url.QueryEscape(query)
	fields := a.buildFieldsParam()
	jiraURL := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&expand=changelog&fields=%s",
		a.config.GetBaseURL(), encodedQuery, fields)

	issues, err := a.httpClient.GetJiraIssues(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprint issues for board %d: %w", boardID, err)
	}

	a.enrichIssues(issues)
	return a.convertToPortIssues(issues), nil
}

// getIssuesForBoardByDateRange fetches issues from a board that were updated within the given date range.
func (a *JiraAdapter) getIssuesForBoardByDateRange(project string, boardID int, startDate, endDate string) ([]ports.JiraIssue, error) {
	// Parse dates to extract just the date portion for JQL
	start, err := time.Parse(time.RFC3339, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start date: %w", err)
	}
	end, err := time.Parse(time.RFC3339, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse end date: %w", err)
	}

	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	// Query issues updated during the sprint period, filtered by board membership
	query := fmt.Sprintf("project = %s AND updatedDate >= '%s' AND updatedDate <= '%s'",
		project, startStr, endStr)
	encodedQuery := url.QueryEscape(query)
	fields := a.buildFieldsParam()

	// Use the Agile board issue endpoint to only get issues visible on this board
	jiraURL := fmt.Sprintf("%s/rest/agile/1.0/board/%d/issue?jql=%s&expand=changelog&fields=%s",
		a.config.GetBaseURL(), boardID, encodedQuery, fields)

	issues, err := a.httpClient.GetJiraIssues(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch board %d issues by date range: %w", boardID, err)
	}

	a.enrichIssues(issues)
	return a.convertToPortIssues(issues), nil
}

// UpdateCustomFields writes custom field values to a JIRA issue via PUT /rest/api/3/issue/{key}
func (a *JiraAdapter) UpdateCustomFields(issueKey string, update ports.CustomFieldUpdate) error {
	if a.fieldIDs == nil {
		return fmt.Errorf("custom field IDs not resolved")
	}

	fields := make(map[string]interface{})

	if update.EngineeringHours != nil && a.fieldIDs.EngineeringHours != "" {
		fields[a.fieldIDs.EngineeringHours] = *update.EngineeringHours
	}

	if update.WorkStream != nil && a.fieldIDs.WorkStream != "" {
		fields[a.fieldIDs.WorkStream] = map[string]string{"value": *update.WorkStream}
	}

	if len(update.TPDBusinessUnits) > 0 && a.fieldIDs.TPDBusinessUnit != "" {
		values := make([]map[string]string, len(update.TPDBusinessUnits))
		for i, bu := range update.TPDBusinessUnits {
			values[i] = map[string]string{"value": bu}
		}
		fields[a.fieldIDs.TPDBusinessUnit] = values
	}

	if len(fields) == 0 {
		return nil
	}

	payload := map[string]interface{}{"fields": fields}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal update payload: %w", err)
	}

	putURL := fmt.Sprintf("%s/rest/api/3/issue/%s", a.config.GetBaseURL(), issueKey)
	return a.httpClient.Put(putURL, body)
}

// FetchCustomFields reads current custom field values from a single JIRA issue
func (a *JiraAdapter) FetchCustomFields(issueKey string) (*ports.CustomFieldValues, error) {
	if a.fieldIDs == nil {
		return nil, fmt.Errorf("custom field IDs not resolved")
	}

	var fieldParams []string
	if a.fieldIDs.EngineeringHours != "" {
		fieldParams = append(fieldParams, a.fieldIDs.EngineeringHours)
	}
	if a.fieldIDs.WorkStream != "" {
		fieldParams = append(fieldParams, a.fieldIDs.WorkStream)
	}
	if a.fieldIDs.TPDBusinessUnit != "" {
		fieldParams = append(fieldParams, a.fieldIDs.TPDBusinessUnit)
	}

	getURL := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=%s",
		a.config.GetBaseURL(), issueKey, strings.Join(fieldParams, ","))

	body, err := a.httpClient.Get(getURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue %s: %w", issueKey, err)
	}

	// Parse the issue to extract custom field values using the existing EnrichCustomFields logic
	var issue domain.JiraIssue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal issue %s: %w", issueKey, err)
	}

	issue.EnrichCustomFields(*a.fieldIDs)

	return &ports.CustomFieldValues{
		EngineeringHours: issue.Fields.EngineeringHours,
		WorkStream:       issue.Fields.WorkStream,
		TPDBusinessUnits: issue.Fields.TPDBusinessUnits,
	}, nil
}

// Ensure JiraAdapter implements JiraPort
var _ ports.JiraPort = (*JiraAdapter)(nil)
