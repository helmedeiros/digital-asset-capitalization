package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

	// Parse existing file format: {"FN": {"team": ["member1", "member2"], "nicknames": ["pricing", "fintech"]}}
	var fileFormat map[string]struct {
		Team      []string `json:"team"`
		Nicknames []string `json:"nicknames,omitempty"`
	}

	if err := json.Unmarshal(data, &fileFormat); err != nil {
		return nil, fmt.Errorf("failed to parse team config: %w", err)
	}

	// Transform to domain format
	teams := make(map[string][]string)
	nicknames := make(map[string][]string)

	for project, teamInfo := range fileFormat {
		teams[project] = teamInfo.Team
		if len(teamInfo.Nicknames) > 0 {
			nicknames[project] = teamInfo.Nicknames
		}
	}

	return domain.NewTeamConfigWithNicknames(teams, nicknames)
}

// SaveTeamConfig saves team configuration to file with format transformation
func (r *FileRepository) SaveTeamConfig(config *domain.TeamConfig) error {
	path := filepath.Join(r.configDir, "teams.json")

	// Transform from domain format to file format
	teams, nicknames := config.ToMapWithNicknames()
	fileFormat := make(map[string]struct {
		Team      []string `json:"team"`
		Nicknames []string `json:"nicknames,omitempty"`
	})

	for project, members := range teams {
		entry := struct {
			Team      []string `json:"team"`
			Nicknames []string `json:"nicknames,omitempty"`
		}{
			Team: members,
		}

		// Add nicknames if they exist for this project
		if nicks, exists := nicknames[project]; exists && len(nicks) > 0 {
			entry.Nicknames = nicks
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
