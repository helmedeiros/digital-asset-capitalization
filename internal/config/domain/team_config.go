package domain

import (
	"fmt"
	"slices"
	"strings"
)

// TeamConfig represents the team configuration domain entity
type TeamConfig struct {
	teams                 map[string][]string
	nicknames             map[string][]string // project -> nicknames mapping
	tribes                map[string]string   // project -> tribe mapping
	companies             map[string]string   // project -> company mapping
	confluenceSpaces      map[string]string   // project -> confluence space mapping
	confluenceParentPages map[string]string   // project -> confluence parent page ID mapping
	excludedIssueTypes    map[string][]string // project -> excluded issue types for sprint allocation
}

// NewTeamConfig creates a new TeamConfig with validation
func NewTeamConfig(teams map[string][]string) (*TeamConfig, error) {
	config := &TeamConfig{
		teams:                 make(map[string][]string),
		nicknames:             make(map[string][]string),
		tribes:                make(map[string]string),
		companies:             make(map[string]string),
		confluenceSpaces:      make(map[string]string),
		confluenceParentPages: make(map[string]string),
		excludedIssueTypes:    make(map[string][]string),
	}

	for project, members := range teams {
		trimmedProject := strings.TrimSpace(project)
		if trimmedProject == "" {
			return nil, fmt.Errorf("project key cannot be empty")
		}

		var trimmedMembers []string
		memberSet := make(map[string]bool)

		for _, member := range members {
			trimmedMember := strings.TrimSpace(member)
			if trimmedMember == "" {
				return nil, fmt.Errorf("team member cannot be empty")
			}

			if memberSet[trimmedMember] {
				return nil, fmt.Errorf("duplicate team member '%s' in project '%s'", trimmedMember, trimmedProject)
			}

			memberSet[trimmedMember] = true
			trimmedMembers = append(trimmedMembers, trimmedMember)
		}

		config.teams[trimmedProject] = trimmedMembers
	}

	return config, nil
}

// NewTeamConfigWithNicknames creates a new TeamConfig with teams and nicknames
func NewTeamConfigWithNicknames(teams map[string][]string, nicknames map[string][]string) (*TeamConfig, error) {
	// First create config with teams
	config, err := NewTeamConfig(teams)
	if err != nil {
		return nil, err
	}

	// Then add nicknames with validation
	for project, nicks := range nicknames {
		trimmedProject := strings.TrimSpace(project)
		if trimmedProject == "" {
			return nil, fmt.Errorf("project key cannot be empty in nicknames")
		}

		// Ensure project exists in teams
		if _, exists := config.teams[trimmedProject]; !exists {
			return nil, fmt.Errorf("project '%s' has nicknames but no team defined", trimmedProject)
		}

		trimmedNicks := make([]string, 0, len(nicks))
		nickSet := make(map[string]bool)

		for _, nick := range nicks {
			// Convert to lowercase for case-insensitive matching
			trimmedNick := strings.ToLower(strings.TrimSpace(nick))
			if trimmedNick == "" {
				return nil, fmt.Errorf("nickname cannot be empty")
			}

			if nickSet[trimmedNick] {
				return nil, fmt.Errorf("duplicate nickname '%s' for project '%s'", trimmedNick, trimmedProject)
			}

			nickSet[trimmedNick] = true
			trimmedNicks = append(trimmedNicks, trimmedNick)
		}

		config.nicknames[trimmedProject] = trimmedNicks
	}

	return config, nil
}

// NewTeamConfigWithTribes creates a new TeamConfig with teams, nicknames, and tribes
func NewTeamConfigWithTribes(teams map[string][]string, nicknames map[string][]string, tribes map[string]string) (*TeamConfig, error) {
	return NewTeamConfigFull(teams, nicknames, tribes, nil)
}

// NewTeamConfigFull creates a new TeamConfig with teams, nicknames, tribes, and companies
func NewTeamConfigFull(teams map[string][]string, nicknames map[string][]string, tribes map[string]string, companies map[string]string) (*TeamConfig, error) {
	// First create config with teams and nicknames
	config, err := NewTeamConfigWithNicknames(teams, nicknames)
	if err != nil {
		return nil, err
	}

	// Then add tribes
	for project, tribe := range tribes {
		trimmedProject := strings.TrimSpace(project)
		trimmedTribe := strings.TrimSpace(tribe)

		if trimmedProject == "" {
			continue
		}

		// Only set tribe if the project exists
		if _, exists := config.teams[trimmedProject]; exists && trimmedTribe != "" {
			config.tribes[trimmedProject] = trimmedTribe
		}
	}

	// Then add companies
	for project, company := range companies {
		trimmedProject := strings.TrimSpace(project)
		trimmedCompany := strings.TrimSpace(company)

		if trimmedProject == "" {
			continue
		}

		// Only set company if the project exists
		if _, exists := config.teams[trimmedProject]; exists && trimmedCompany != "" {
			config.companies[trimmedProject] = trimmedCompany
		}
	}

	return config, nil
}

// GetTribe returns the tribe for a given project
func (tc *TeamConfig) GetTribe(project string) string {
	return tc.tribes[project]
}

// SetTribe sets the tribe for a project
func (tc *TeamConfig) SetTribe(project, tribe string) error {
	trimmedProject := strings.TrimSpace(project)
	if trimmedProject == "" {
		return fmt.Errorf("project key cannot be empty")
	}

	// Ensure project exists in teams
	if _, exists := tc.teams[trimmedProject]; !exists {
		return fmt.Errorf("project '%s' does not exist", trimmedProject)
	}

	tc.tribes[trimmedProject] = strings.TrimSpace(tribe)
	return nil
}

// GetCompany returns the company for a given project
func (tc *TeamConfig) GetCompany(project string) string {
	return tc.companies[project]
}

// SetCompany sets the company for a project
func (tc *TeamConfig) SetCompany(project, company string) error {
	trimmedProject := strings.TrimSpace(project)
	if trimmedProject == "" {
		return fmt.Errorf("project key cannot be empty")
	}

	// Ensure project exists in teams
	if _, exists := tc.teams[trimmedProject]; !exists {
		return fmt.Errorf("project '%s' does not exist", trimmedProject)
	}

	tc.companies[trimmedProject] = strings.TrimSpace(company)
	return nil
}

// GetTeam returns the team members for a given project
func (tc *TeamConfig) GetTeam(project string) ([]string, bool) {
	team, exists := tc.teams[project]
	if !exists {
		return nil, false
	}
	// Return a copy to prevent external modification
	result := make([]string, len(team))
	copy(result, team)
	return result, true
}

// GetProjects returns all project keys
func (tc *TeamConfig) GetProjects() []string {
	projects := make([]string, 0, len(tc.teams))
	for project := range tc.teams {
		projects = append(projects, project)
	}
	return projects
}

// IsTeamMember checks if a person is a member of the team for a given project
func (tc *TeamConfig) IsTeamMember(project, member string) bool {
	team, exists := tc.teams[project]
	if !exists {
		return false
	}
	return slices.Contains(team, member)
}

// AddTeamMember adds a team member to a project
func (tc *TeamConfig) AddTeamMember(project, member string) error {
	trimmedProject := strings.TrimSpace(project)
	if trimmedProject == "" {
		return fmt.Errorf("project key cannot be empty")
	}

	trimmedMember := strings.TrimSpace(member)
	if trimmedMember == "" {
		return fmt.Errorf("team member cannot be empty")
	}

	team, exists := tc.teams[trimmedProject]
	if !exists {
		tc.teams[trimmedProject] = []string{trimmedMember}
		return nil
	}

	if slices.Contains(team, trimmedMember) {
		return fmt.Errorf("team member '%s' already exists in project '%s'", trimmedMember, trimmedProject)
	}

	tc.teams[trimmedProject] = append(team, trimmedMember)
	return nil
}

// RemoveTeamMember removes a team member from a project
func (tc *TeamConfig) RemoveTeamMember(project, member string) error {
	team, exists := tc.teams[project]
	if !exists {
		return fmt.Errorf("project not found: '%s'", project)
	}

	memberIndex := slices.Index(team, member)
	if memberIndex == -1 {
		return fmt.Errorf("team member '%s' not found in project '%s'", member, project)
	}

	// Remove the member from the slice
	tc.teams[project] = slices.Delete(team, memberIndex, memberIndex+1)
	return nil
}

// IsEmpty checks if the team configuration is empty
func (tc *TeamConfig) IsEmpty() bool {
	return len(tc.teams) == 0
}

// SetTeam sets the entire team for a project, replacing existing members
func (tc *TeamConfig) SetTeam(project string, members []string) error {
	trimmedProject := strings.TrimSpace(project)
	if trimmedProject == "" {
		return fmt.Errorf("project key cannot be empty")
	}

	// Validate and trim members, check for duplicates
	memberSet := make(map[string]bool)
	trimmedMembers := make([]string, 0, len(members))

	for _, member := range members {
		trimmedMember := strings.TrimSpace(member)
		if trimmedMember == "" {
			return fmt.Errorf("team member cannot be empty")
		}

		if memberSet[trimmedMember] {
			return fmt.Errorf("duplicate team member '%s' in project '%s'", trimmedMember, trimmedProject)
		}

		memberSet[trimmedMember] = true
		trimmedMembers = append(trimmedMembers, trimmedMember)
	}

	tc.teams[trimmedProject] = trimmedMembers
	return nil
}

// ToMap returns a copy of the internal teams map
func (tc *TeamConfig) ToMap() map[string][]string {
	result := make(map[string][]string, len(tc.teams))
	for project, members := range tc.teams {
		membersCopy := make([]string, len(members))
		copy(membersCopy, members)
		result[project] = membersCopy
	}
	return result
}

// GetNicknames returns the nicknames for a given project
func (tc *TeamConfig) GetNicknames(project string) []string {
	nicks, exists := tc.nicknames[project]
	if !exists {
		return nil
	}
	// Return a copy to prevent external modification
	result := make([]string, len(nicks))
	copy(result, nicks)
	return result
}

// SetNicknames sets the nicknames for a project
func (tc *TeamConfig) SetNicknames(project string, nicknames []string) error {
	trimmedProject := strings.TrimSpace(project)
	if trimmedProject == "" {
		return fmt.Errorf("project key cannot be empty")
	}

	// Ensure project exists in teams
	if _, exists := tc.teams[trimmedProject]; !exists {
		return fmt.Errorf("project '%s' does not exist", trimmedProject)
	}

	// Validate and normalize nicknames
	trimmedNicks := make([]string, 0, len(nicknames))
	nickSet := make(map[string]bool)

	for _, nick := range nicknames {
		trimmedNick := strings.ToLower(strings.TrimSpace(nick))
		if trimmedNick == "" {
			return fmt.Errorf("nickname cannot be empty")
		}

		if nickSet[trimmedNick] {
			return fmt.Errorf("duplicate nickname '%s'", trimmedNick)
		}

		nickSet[trimmedNick] = true
		trimmedNicks = append(trimmedNicks, trimmedNick)
	}

	tc.nicknames[trimmedProject] = trimmedNicks
	return nil
}

// ResolveTeamIdentifier resolves a team identifier (project code or nickname) to the actual project code
func (tc *TeamConfig) ResolveTeamIdentifier(identifier string) (string, error) {
	if identifier == "" {
		return "", fmt.Errorf("identifier cannot be empty")
	}

	// First check if it's already a valid project code (case-sensitive)
	if _, exists := tc.teams[identifier]; exists {
		return identifier, nil
	}

	// Then check if it's a nickname (case-insensitive)
	lowerIdentifier := strings.ToLower(strings.TrimSpace(identifier))

	for project, nicks := range tc.nicknames {
		for _, nick := range nicks {
			if nick == lowerIdentifier {
				return project, nil
			}
		}
	}

	// Also check project codes case-insensitively as a fallback
	for project := range tc.teams {
		if strings.ToLower(project) == lowerIdentifier {
			return project, nil
		}
	}

	return "", fmt.Errorf("unknown project or nickname: %s", identifier)
}

// GetAllNicknameMappings returns a map of all nicknames to their project codes
func (tc *TeamConfig) GetAllNicknameMappings() map[string]string {
	result := make(map[string]string)

	for project, nicks := range tc.nicknames {
		for _, nick := range nicks {
			result[nick] = project
		}
	}

	return result
}

// ToMapWithNicknames returns both teams and nicknames maps
func (tc *TeamConfig) ToMapWithNicknames() (map[string][]string, map[string][]string) {
	teams := tc.ToMap()

	nicknames := make(map[string][]string, len(tc.nicknames))
	for project, nicks := range tc.nicknames {
		nicksCopy := make([]string, len(nicks))
		copy(nicksCopy, nicks)
		nicknames[project] = nicksCopy
	}

	return teams, nicknames
}

// ToFullMap returns teams, nicknames, tribes, and companies maps
func (tc *TeamConfig) ToFullMap() (teams map[string][]string, nicknames map[string][]string, tribes map[string]string, companies map[string]string) {
	teams, nicknames = tc.ToMapWithNicknames()

	tribes = make(map[string]string, len(tc.tribes))
	for project, tribe := range tc.tribes {
		tribes[project] = tribe
	}

	companies = make(map[string]string, len(tc.companies))
	for project, company := range tc.companies {
		companies[project] = company
	}

	return teams, nicknames, tribes, companies
}

// NewTeamConfigComplete creates a new TeamConfig with all fields including confluence settings
func NewTeamConfigComplete(teams map[string][]string, nicknames map[string][]string, tribes map[string]string, companies map[string]string, confluenceSpaces map[string]string, confluenceParentPages map[string]string) (*TeamConfig, error) {
	return NewTeamConfigWithExcludedTypes(teams, nicknames, tribes, companies, confluenceSpaces, confluenceParentPages, nil)
}

// NewTeamConfigWithExcludedTypes creates a new TeamConfig with all fields including excluded issue types
func NewTeamConfigWithExcludedTypes(teams map[string][]string, nicknames map[string][]string, tribes map[string]string, companies map[string]string, confluenceSpaces map[string]string, confluenceParentPages map[string]string, excludedIssueTypes map[string][]string) (*TeamConfig, error) {
	config, err := NewTeamConfigFull(teams, nicknames, tribes, companies)
	if err != nil {
		return nil, err
	}

	for project, space := range confluenceSpaces {
		trimmedProject := strings.TrimSpace(project)
		trimmedSpace := strings.TrimSpace(space)
		if trimmedProject == "" {
			continue
		}
		if _, exists := config.teams[trimmedProject]; exists && trimmedSpace != "" {
			config.confluenceSpaces[trimmedProject] = trimmedSpace
		}
	}

	for project, pageID := range confluenceParentPages {
		trimmedProject := strings.TrimSpace(project)
		trimmedPageID := strings.TrimSpace(pageID)
		if trimmedProject == "" {
			continue
		}
		if _, exists := config.teams[trimmedProject]; exists && trimmedPageID != "" {
			config.confluenceParentPages[trimmedProject] = trimmedPageID
		}
	}

	for project, types := range excludedIssueTypes {
		trimmedProject := strings.TrimSpace(project)
		if trimmedProject == "" {
			continue
		}
		if _, exists := config.teams[trimmedProject]; exists && len(types) > 0 {
			config.excludedIssueTypes[trimmedProject] = types
		}
	}

	return config, nil
}

// GetConfluenceSpace returns the Confluence space for a given project
func (tc *TeamConfig) GetConfluenceSpace(project string) string {
	return tc.confluenceSpaces[project]
}

// SetConfluenceSpace sets the Confluence space for a project
func (tc *TeamConfig) SetConfluenceSpace(project, space string) error {
	trimmedProject := strings.TrimSpace(project)
	if trimmedProject == "" {
		return fmt.Errorf("project key cannot be empty")
	}
	if _, exists := tc.teams[trimmedProject]; !exists {
		return fmt.Errorf("project '%s' does not exist", trimmedProject)
	}
	tc.confluenceSpaces[trimmedProject] = strings.TrimSpace(space)
	return nil
}

// GetConfluenceParentPage returns the Confluence parent page ID for a given project
func (tc *TeamConfig) GetConfluenceParentPage(project string) string {
	return tc.confluenceParentPages[project]
}

// SetConfluenceParentPage sets the Confluence parent page ID for a project
func (tc *TeamConfig) SetConfluenceParentPage(project, pageID string) error {
	trimmedProject := strings.TrimSpace(project)
	if trimmedProject == "" {
		return fmt.Errorf("project key cannot be empty")
	}
	if _, exists := tc.teams[trimmedProject]; !exists {
		return fmt.Errorf("project '%s' does not exist", trimmedProject)
	}
	tc.confluenceParentPages[trimmedProject] = strings.TrimSpace(pageID)
	return nil
}

// ToCompleteMap returns all six maps including confluence settings
func (tc *TeamConfig) ToCompleteMap() (teams map[string][]string, nicknames map[string][]string, tribes map[string]string, companies map[string]string, confluenceSpaces map[string]string, confluenceParentPages map[string]string) {
	teams, nicknames, tribes, companies = tc.ToFullMap()

	confluenceSpaces = make(map[string]string, len(tc.confluenceSpaces))
	for project, space := range tc.confluenceSpaces {
		confluenceSpaces[project] = space
	}

	confluenceParentPages = make(map[string]string, len(tc.confluenceParentPages))
	for project, pageID := range tc.confluenceParentPages {
		confluenceParentPages[project] = pageID
	}

	return
}

// ToCompleteMapWithExcludedTypes returns all maps including excluded issue types
func (tc *TeamConfig) ToCompleteMapWithExcludedTypes() (teams map[string][]string, nicknames map[string][]string, tribes map[string]string, companies map[string]string, confluenceSpaces map[string]string, confluenceParentPages map[string]string, excludedIssueTypes map[string][]string) {
	teams, nicknames, tribes, companies, confluenceSpaces, confluenceParentPages = tc.ToCompleteMap()

	excludedIssueTypes = make(map[string][]string, len(tc.excludedIssueTypes))
	for project, types := range tc.excludedIssueTypes {
		typesCopy := make([]string, len(types))
		copy(typesCopy, types)
		excludedIssueTypes[project] = typesCopy
	}

	return
}

// GetExcludedIssueTypes returns the excluded issue types for a given project
func (tc *TeamConfig) GetExcludedIssueTypes(project string) []string {
	types, exists := tc.excludedIssueTypes[project]
	if !exists {
		return nil
	}
	result := make([]string, len(types))
	copy(result, types)
	return result
}

// SetExcludedIssueTypes sets the excluded issue types for a project
func (tc *TeamConfig) SetExcludedIssueTypes(project string, types []string) error {
	trimmedProject := strings.TrimSpace(project)
	if trimmedProject == "" {
		return fmt.Errorf("project key cannot be empty")
	}
	if _, exists := tc.teams[trimmedProject]; !exists {
		return fmt.Errorf("project '%s' does not exist", trimmedProject)
	}
	tc.excludedIssueTypes[trimmedProject] = types
	return nil
}
