package application

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// TeamResolverService handles resolution of team identifiers (project codes or nicknames)
type TeamResolverService struct {
	teamConfig *domain.TeamConfig
}

// NewTeamResolverService creates a new team resolver service
func NewTeamResolverService(teamConfig *domain.TeamConfig) *TeamResolverService {
	return &TeamResolverService{
		teamConfig: teamConfig,
	}
}

// ResolveProjectIdentifier resolves any identifier (project code or nickname) to the actual project code
// This ensures that external systems (JIRA, Confluence) always receive valid project codes
func (s *TeamResolverService) ResolveProjectIdentifier(identifier string) (string, error) {
	if identifier == "" {
		return "", nil // Allow empty identifiers (for optional fields)
	}

	return s.teamConfig.ResolveTeamIdentifier(identifier)
}

// ResolveMultipleIdentifiers resolves multiple identifiers to their project codes
func (s *TeamResolverService) ResolveMultipleIdentifiers(identifiers []string) ([]string, error) {
	resolved := make([]string, 0, len(identifiers))

	for _, id := range identifiers {
		if id == "" {
			continue // Skip empty identifiers
		}

		resolvedID, err := s.ResolveProjectIdentifier(id)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve '%s': %w", id, err)
		}
		resolved = append(resolved, resolvedID)
	}

	return resolved, nil
}

// GetProjectWithNicknames returns a formatted string showing project and its nicknames
func (s *TeamResolverService) GetProjectWithNicknames(project string) string {
	nicknames := s.teamConfig.GetNicknames(project)
	if len(nicknames) == 0 {
		return project
	}

	// Format as "FN (Pricing, Fintech)"
	nicknamesStr := ""
	for i, nick := range nicknames {
		if i > 0 {
			nicknamesStr += ", "
		}
		// Capitalize first letter for display
		if len(nick) > 0 {
			nicknamesStr += string(nick[0]-32) + nick[1:]
		}
	}

	return fmt.Sprintf("%s (%s)", project, nicknamesStr)
}

// GetAllMappings returns all nickname to project mappings for display
func (s *TeamResolverService) GetAllMappings() map[string]string {
	return s.teamConfig.GetAllNicknameMappings()
}

// UpdateTeamConfig updates the underlying team configuration
func (s *TeamResolverService) UpdateTeamConfig(teamConfig *domain.TeamConfig) {
	s.teamConfig = teamConfig
}
