package domain

import "strings"

// StatusType represents the normalized status type
type StatusType string

const (
	StatusTypeDone       StatusType = "done"
	StatusTypeInProgress StatusType = "in_progress"
	StatusTypeWontDo     StatusType = "wont_do"
	StatusTypeTodo       StatusType = "todo"
	StatusTypeUnknown    StatusType = "unknown"
)

// BoardConfig represents board-specific configuration
type BoardConfig struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	StatusMappings map[string][]string `json:"statusMappings"`
}

// TeamConfig represents the complete team configuration
type TeamConfig struct {
	Team      []string               `json:"team"`
	Nicknames []string               `json:"nicknames,omitempty"`
	Boards    map[string]BoardConfig `json:"boards,omitempty"`
}

// StatusMapper handles status normalization
type StatusMapper struct {
	teamConfigs map[string]TeamConfig
}

// NewStatusMapper creates a new status mapper
func NewStatusMapper(configs map[string]TeamConfig) *StatusMapper {
	return &StatusMapper{
		teamConfigs: configs,
	}
}

// NormalizeStatus converts a JIRA status to a normalized status type
func (sm *StatusMapper) NormalizeStatus(status string, teamKey string, boardID string) StatusType {
	// First, try team-specific mappings
	if team, exists := sm.teamConfigs[teamKey]; exists {
		if board, boardExists := team.Boards[boardID]; boardExists {
			return sm.findStatusType(status, board.StatusMappings)
		}
	}

	// Fall back to default mappings
	return sm.getDefaultStatusType(status)
}

// findStatusType searches for a status in the mapping
func (sm *StatusMapper) findStatusType(status string, mappings map[string][]string) StatusType {
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))

	for statusType, statusList := range mappings {
		for _, mappedStatus := range statusList {
			if strings.ToLower(strings.TrimSpace(mappedStatus)) == normalizedStatus {
				return StatusType(statusType)
			}
		}
	}

	return StatusTypeUnknown
}

// getDefaultStatusType provides default status mapping
func (sm *StatusMapper) getDefaultStatusType(status string) StatusType {
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))

	// Default mappings for common statuses
	switch {
	case strings.Contains(normalizedStatus, "done") ||
		strings.Contains(normalizedStatus, "closed") ||
		strings.Contains(normalizedStatus, "resolved") ||
		strings.Contains(normalizedStatus, "deployed"):
		return StatusTypeDone

	case strings.Contains(normalizedStatus, "progress") ||
		strings.Contains(normalizedStatus, "development") ||
		strings.Contains(normalizedStatus, "review"):
		return StatusTypeInProgress

	case strings.Contains(normalizedStatus, "won't") ||
		strings.Contains(normalizedStatus, "cancelled") ||
		strings.Contains(normalizedStatus, "duplicate"):
		return StatusTypeWontDo

	case strings.Contains(normalizedStatus, "todo") ||
		strings.Contains(normalizedStatus, "open") ||
		strings.Contains(normalizedStatus, "backlog"):
		return StatusTypeTodo

	default:
		return StatusTypeUnknown
	}
}

// IsDone checks if a status represents completed work
func (sm *StatusMapper) IsDone(status string, teamKey string, boardID string) bool {
	return sm.NormalizeStatus(status, teamKey, boardID) == StatusTypeDone
}

// IsWontDo checks if a status represents work that won't be done
func (sm *StatusMapper) IsWontDo(status string, teamKey string, boardID string) bool {
	return sm.NormalizeStatus(status, teamKey, boardID) == StatusTypeWontDo
}

// IsInProgress checks if a status represents work in progress
func (sm *StatusMapper) IsInProgress(status string, teamKey string, boardID string) bool {
	return sm.NormalizeStatus(status, teamKey, boardID) == StatusTypeInProgress
}
