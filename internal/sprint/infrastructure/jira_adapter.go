package infrastructure

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/infrastructure"
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

	return &JiraAdapter{
		config:        jiraConfig,
		teams:         teams,
		httpClient:    httpClient,
		configService: configService,
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

// GetIssuesForSprint retrieves all issues for a given sprint
func (a *JiraAdapter) GetIssuesForSprint(project, sprintID string) ([]ports.JiraIssue, error) {
	query := fmt.Sprintf("project = %s AND sprint = '%s'", project, sprintID)
	encodedQuery := url.QueryEscape(query)
	fields := "summary,assignee,status,changelog,issuetype,customfield_10014,customfield_10015,labels"
	jiraURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s&expand=changelog&fields=%s",
		a.config.GetBaseURL(), encodedQuery, fields)

	issues, err := a.httpClient.GetJiraIssues(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprint issues: %w", err)
	}

	return a.convertToPortIssues(issues), nil
}

// GetIssuesForTeamMember retrieves all issues assigned to a team member
func (a *JiraAdapter) GetIssuesForTeamMember(member string) ([]ports.JiraIssue, error) {
	query := fmt.Sprintf("assignee = '%s'", member)
	encodedQuery := url.QueryEscape(query)
	fields := "summary,assignee,status,changelog,issuetype,customfield_10014,customfield_10015,labels"
	jiraURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s&expand=changelog&fields=%s",
		a.config.GetBaseURL(), encodedQuery, fields)

	issues, err := a.httpClient.GetJiraIssues(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch team member issues: %w", err)
	}

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
			Key:         issue.Key,
			Summary:     issue.Fields.Summary,
			Assignee:    issue.Fields.Assignee.DisplayName,
			Status:      issue.Fields.Status.Name,
			StoryPoints: issue.Fields.StoryPoints,
			IssueType:   issue.Fields.IssueType.Name,
			Labels:      issue.Fields.Labels,
			Changelog:   convertChangelog(issue.Changelog),
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
			// Log error but continue with other boards
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

// Ensure JiraAdapter implements JiraPort
var _ ports.JiraPort = (*JiraAdapter)(nil)
