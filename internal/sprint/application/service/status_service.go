package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	sharedDomain "github.com/helmedeiros/digital-asset-capitalization/internal/shared/domain"
)

// StatusService handles status normalization for sprint calculations
type StatusService struct {
	statusMapper *sharedDomain.StatusMapper
	teamConfigs  map[string]sharedDomain.TeamConfig
}

// NewStatusService creates a new status service
func NewStatusService() (*StatusService, error) {
	return NewStatusServiceWithPath("")
}

// NewStatusServiceWithPath creates a new status service with a specific config path
func NewStatusServiceWithPath(configPath string) (*StatusService, error) {
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		configPath = filepath.Join(homeDir, "Library", "CloudStorage", "Dropbox",
			"PROJECTS", "workspaceGoEuro", "assetcap", ".assetcap", "teams.json")

		// Try alternate path if primary doesn't exist
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = filepath.Join(homeDir, ".assetcap", "teams.json")
		}
	}

	// Load team configurations
	teamConfigs, err := loadTeamConfigs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load team configs: %w", err)
	}

	return NewStatusServiceWithConfigs(teamConfigs), nil
}

// NewStatusServiceWithConfigs creates a new status service with provided team configs (for testing)
func NewStatusServiceWithConfigs(teamConfigs map[string]sharedDomain.TeamConfig) *StatusService {
	mapper := sharedDomain.NewStatusMapper(teamConfigs)

	return &StatusService{
		statusMapper: mapper,
		teamConfigs:  teamConfigs,
	}
}

// loadTeamConfigs loads team configurations from JSON file
func loadTeamConfigs(path string) (map[string]sharedDomain.TeamConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return make(map[string]sharedDomain.TeamConfig), nil
		}
		return nil, err
	}

	// Parse the JSON into a map with mixed structure
	var rawConfigs map[string]interface{}
	if err := json.Unmarshal(data, &rawConfigs); err != nil {
		return nil, err
	}

	// Convert to proper TeamConfig structure
	configs := make(map[string]sharedDomain.TeamConfig)
	for key, value := range rawConfigs {
		configMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		var config sharedDomain.TeamConfig

		// Parse team members
		if team, ok := configMap["team"].([]interface{}); ok {
			config.Team = make([]string, len(team))
			for i, member := range team {
				config.Team[i] = member.(string)
			}
		}

		// Parse nicknames
		if nicknames, ok := configMap["nicknames"].([]interface{}); ok {
			config.Nicknames = make([]string, len(nicknames))
			for i, nickname := range nicknames {
				config.Nicknames[i] = nickname.(string)
			}
		}

		// Parse boards
		if boards, ok := configMap["boards"].(map[string]interface{}); ok {
			config.Boards = make(map[string]sharedDomain.BoardConfig)
			for boardID, boardData := range boards {
				boardMap, ok := boardData.(map[string]interface{})
				if !ok {
					continue
				}

				boardConfig := sharedDomain.BoardConfig{
					ID:   boardID,
					Name: boardMap["name"].(string),
				}

				// Parse status mappings
				if mappings, ok := boardMap["statusMappings"].(map[string]interface{}); ok {
					boardConfig.StatusMappings = make(map[string][]string)
					for statusType, statusList := range mappings {
						if list, ok := statusList.([]interface{}); ok {
							statuses := make([]string, len(list))
							for i, status := range list {
								statuses[i] = status.(string)
							}
							boardConfig.StatusMappings[statusType] = statuses
						}
					}
				}

				config.Boards[boardID] = boardConfig
			}
		}

		configs[key] = config
	}

	return configs, nil
}

// NormalizeStatus normalizes a JIRA status for a given team and board
func (s *StatusService) NormalizeStatus(status string, projectKey string, boardID string) sharedDomain.StatusType {
	return s.statusMapper.NormalizeStatus(status, projectKey, boardID)
}

// IsDone checks if a status represents completed work
func (s *StatusService) IsDone(status string, projectKey string, boardID string) bool {
	return s.statusMapper.IsDone(status, projectKey, boardID)
}

// IsWontDo checks if a status represents work that won't be done
func (s *StatusService) IsWontDo(status string, projectKey string, boardID string) bool {
	return s.statusMapper.IsWontDo(status, projectKey, boardID)
}

// IsInProgress checks if a status represents work in progress
func (s *StatusService) IsInProgress(status string, projectKey string, boardID string) bool {
	return s.statusMapper.IsInProgress(status, projectKey, boardID)
}

// GetBoardIDForTeam gets the board ID for a team (returns first board if multiple)
func (s *StatusService) GetBoardIDForTeam(projectKey string) string {
	if team, exists := s.teamConfigs[projectKey]; exists {
		for boardID := range team.Boards {
			return boardID
		}
	}
	return ""
}
