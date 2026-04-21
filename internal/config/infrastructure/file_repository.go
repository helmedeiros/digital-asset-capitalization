package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// FileRepository implements ConfigurationRepository using the file system
type FileRepository struct {
	configDir string
}

// NewFileRepository creates a new file-based configuration repository
func NewFileRepository(configDir string) *FileRepository {
	return &FileRepository{
		configDir: configDir,
	}
}

// InitializeConfigDirectory creates the configuration directory if it doesn't exist
func (r *FileRepository) InitializeConfigDirectory() error {
	return os.MkdirAll(r.configDir, 0755)
}

// ConfigExists checks if configuration files exist
func (r *FileRepository) ConfigExists() (bool, error) {
	jiraPath := filepath.Join(r.configDir, "jira.json")
	teamsPath := filepath.Join(r.configDir, "teams.json")

	jiraExists := r.fileExists(jiraPath)
	teamsExists := r.fileExists(teamsPath)

	return jiraExists || teamsExists, nil
}

// LoadJiraConfig loads Jira configuration from file
func (r *FileRepository) LoadJiraConfig() (*domain.JiraConfig, error) {
	path := filepath.Join(r.configDir, "jira.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("jira configuration not found")
		}
		return nil, fmt.Errorf("failed to read jira config: %w", err)
	}

	var configData struct {
		BaseURL string `json:"base_url"`
		Email   string `json:"email"`
		Token   string `json:"token"`
	}

	if err := json.Unmarshal(data, &configData); err != nil {
		return nil, fmt.Errorf("failed to parse jira config: %w", err)
	}

	return domain.NewJiraConfig(configData.BaseURL, configData.Email, configData.Token)
}

// SaveJiraConfig saves Jira configuration to file
func (r *FileRepository) SaveJiraConfig(config *domain.JiraConfig) error {
	path := filepath.Join(r.configDir, "jira.json")

	configData := struct {
		BaseURL string `json:"base_url"`
		Email   string `json:"email"`
		Token   string `json:"token"`
	}{
		BaseURL: config.BaseURL(),
		Email:   config.Email(),
		Token:   config.Token(),
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jira config: %w", err)
	}

	return r.writeFile(path, data)
}

// LoadTeamConfig loads team configuration from file with format transformation
func (r *FileRepository) LoadTeamConfig() (*domain.TeamConfig, error) {
	path := filepath.Join(r.configDir, "teams.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("team configuration not found")
		}
		return nil, fmt.Errorf("failed to read team config: %w", err)
	}

	// Parse existing file format
	var fileFormat map[string]struct {
		Team         []string `json:"team"`
		TeamTimeline []struct {
			Member string `json:"member"`
			Joined string `json:"joined"`
			Left   string `json:"left,omitempty"`
		} `json:"team_timeline,omitempty"`
		Nicknames            []string          `json:"nicknames,omitempty"`
		Tribe                string            `json:"tribe,omitempty"`
		Company              string            `json:"company,omitempty"`
		ConfluenceSpace      string            `json:"confluence_space,omitempty"`
		ConfluenceParentPage string            `json:"confluence_parent_page,omitempty"`
		ExcludedIssueTypes   []string          `json:"excluded_issue_types,omitempty"`
		BoardWorkStreams     map[string]string `json:"board_work_streams,omitempty"`
	}

	if err := json.Unmarshal(data, &fileFormat); err != nil {
		return nil, fmt.Errorf("failed to parse team config: %w", err)
	}

	// Transform to domain format
	teams := make(map[string][]string)
	teamTimelines := make(map[string][]domain.TeamMemberPeriod)
	nicknames := make(map[string][]string)
	tribes := make(map[string]string)
	companies := make(map[string]string)
	confluenceSpaces := make(map[string]string)
	confluenceParentPages := make(map[string]string)
	excludedIssueTypes := make(map[string][]string)
	boardWorkStreams := make(map[string]map[int]string)

	for project, teamInfo := range fileFormat {
		teams[project] = teamInfo.Team

		if len(teamInfo.TeamTimeline) > 0 {
			var periods []domain.TeamMemberPeriod
			for _, entry := range teamInfo.TeamTimeline {
				joined, err := time.Parse("2006-01-02", entry.Joined)
				if err != nil {
					return nil, fmt.Errorf("failed to parse joined date for %s in %s: %w", entry.Member, project, err)
				}
				period := domain.TeamMemberPeriod{
					Member: entry.Member,
					Joined: joined,
				}
				if entry.Left != "" {
					left, err := time.Parse("2006-01-02", entry.Left)
					if err != nil {
						return nil, fmt.Errorf("failed to parse left date for %s in %s: %w", entry.Member, project, err)
					}
					period.Left = &left
				}
				periods = append(periods, period)
			}
			teamTimelines[project] = periods
		}
		if len(teamInfo.Nicknames) > 0 {
			nicknames[project] = teamInfo.Nicknames
		}
		if teamInfo.Tribe != "" {
			tribes[project] = teamInfo.Tribe
		}
		if teamInfo.Company != "" {
			companies[project] = teamInfo.Company
		}
		if teamInfo.ConfluenceSpace != "" {
			confluenceSpaces[project] = teamInfo.ConfluenceSpace
		}
		if teamInfo.ConfluenceParentPage != "" {
			confluenceParentPages[project] = teamInfo.ConfluenceParentPage
		}
		if len(teamInfo.ExcludedIssueTypes) > 0 {
			excludedIssueTypes[project] = teamInfo.ExcludedIssueTypes
		}
		if len(teamInfo.BoardWorkStreams) > 0 {
			mapping := make(map[int]string, len(teamInfo.BoardWorkStreams))
			for boardIDStr, workStream := range teamInfo.BoardWorkStreams {
				boardID, err := strconv.Atoi(boardIDStr)
				if err != nil {
					continue // skip invalid board IDs
				}
				mapping[boardID] = workStream
			}
			if len(mapping) > 0 {
				boardWorkStreams[project] = mapping
			}
		}
	}

	return domain.NewTeamConfigWithTimelines(teams, nicknames, tribes, companies, confluenceSpaces, confluenceParentPages, excludedIssueTypes, boardWorkStreams, teamTimelines)
}

// SaveTeamConfig saves team configuration to file with format transformation
func (r *FileRepository) SaveTeamConfig(config *domain.TeamConfig) error {
	path := filepath.Join(r.configDir, "teams.json")

	// Transform from domain format to file format
	teams, nicknames, tribes, companies, confluenceSpaces, confluenceParentPages, excludedIssueTypes, boardWorkStreams := config.ToCompleteMapWithBoardWorkStreams()
	allTimelines := config.GetAllTeamTimelines()

	type timelineEntry struct {
		Member string `json:"member"`
		Joined string `json:"joined"`
		Left   string `json:"left,omitempty"`
	}

	fileFormat := make(map[string]struct {
		Team                 []string          `json:"team"`
		TeamTimeline         []timelineEntry   `json:"team_timeline,omitempty"`
		Nicknames            []string          `json:"nicknames,omitempty"`
		Tribe                string            `json:"tribe,omitempty"`
		Company              string            `json:"company,omitempty"`
		ConfluenceSpace      string            `json:"confluence_space,omitempty"`
		ConfluenceParentPage string            `json:"confluence_parent_page,omitempty"`
		ExcludedIssueTypes   []string          `json:"excluded_issue_types,omitempty"`
		BoardWorkStreams     map[string]string `json:"board_work_streams,omitempty"`
	})

	for project, members := range teams {
		// If timeline exists, derive active team from it
		activeMembers := members
		if timeline, hasTimeline := allTimelines[project]; hasTimeline && len(timeline) > 0 {
			activeMembers = config.DeriveActiveTeamFromTimeline(project)
		}

		entry := struct {
			Team                 []string          `json:"team"`
			TeamTimeline         []timelineEntry   `json:"team_timeline,omitempty"`
			Nicknames            []string          `json:"nicknames,omitempty"`
			Tribe                string            `json:"tribe,omitempty"`
			Company              string            `json:"company,omitempty"`
			ConfluenceSpace      string            `json:"confluence_space,omitempty"`
			ConfluenceParentPage string            `json:"confluence_parent_page,omitempty"`
			ExcludedIssueTypes   []string          `json:"excluded_issue_types,omitempty"`
			BoardWorkStreams     map[string]string `json:"board_work_streams,omitempty"`
		}{
			Team: activeMembers,
		}

		// Add timeline if it exists for this project
		if timeline, hasTimeline := allTimelines[project]; hasTimeline && len(timeline) > 0 {
			entries := make([]timelineEntry, 0, len(timeline))
			for _, p := range timeline {
				te := timelineEntry{
					Member: p.Member,
					Joined: p.Joined.Format("2006-01-02"),
				}
				if p.Left != nil {
					te.Left = p.Left.Format("2006-01-02")
				}
				entries = append(entries, te)
			}
			entry.TeamTimeline = entries
		}

		// Add nicknames if they exist for this project
		if nicks, exists := nicknames[project]; exists && len(nicks) > 0 {
			entry.Nicknames = nicks
		}

		// Add tribe if it exists for this project
		if tribe, exists := tribes[project]; exists && tribe != "" {
			entry.Tribe = tribe
		}

		// Add company if it exists for this project
		if company, exists := companies[project]; exists && company != "" {
			entry.Company = company
		}

		// Add confluence space if it exists for this project
		if space, exists := confluenceSpaces[project]; exists && space != "" {
			entry.ConfluenceSpace = space
		}

		// Add confluence parent page if it exists for this project
		if pageID, exists := confluenceParentPages[project]; exists && pageID != "" {
			entry.ConfluenceParentPage = pageID
		}

		// Add excluded issue types if they exist for this project
		if types, exists := excludedIssueTypes[project]; exists && len(types) > 0 {
			entry.ExcludedIssueTypes = types
		}

		// Add board work streams if they exist for this project
		if mapping, exists := boardWorkStreams[project]; exists && len(mapping) > 0 {
			strMapping := make(map[string]string, len(mapping))
			for boardID, ws := range mapping {
				strMapping[strconv.Itoa(boardID)] = ws
			}
			entry.BoardWorkStreams = strMapping
		}

		fileFormat[project] = entry
	}

	data, err := json.MarshalIndent(fileFormat, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal team config: %w", err)
	}

	return r.writeFile(path, data)
}

// fileExists checks if a file exists
func (r *FileRepository) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// writeFile writes data to a file atomically
func (r *FileRepository) writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first for atomic operation
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
