package domain

import (
	"fmt"
	"time"
)

// TeamMember represents a team member from JIRA
type TeamMember struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

// ProjectTeamData represents team information for a specific project
type ProjectTeamData struct {
	ProjectKey string       `json:"projectKey"`
	Members    []TeamMember `json:"members"`
}

// Validate validates the project team data
func (p *ProjectTeamData) Validate() error {
	if p.ProjectKey == "" {
		return fmt.Errorf("project key is required")
	}

	if len(p.Members) == 0 {
		return fmt.Errorf("no team members found for project %s", p.ProjectKey)
	}

	// Validate each member
	for i, member := range p.Members {
		if member.Name == "" && member.DisplayName == "" {
			return fmt.Errorf("member at index %d has no name or display name", i)
		}
	}

	return nil
}

// TeamSyncResult represents the result of synchronizing team data from JIRA
type TeamSyncResult struct {
	ProjectKey     string          `json:"projectKey"`
	Members        []TeamMember    `json:"members"`
	SyncedAt       time.Time       `json:"syncedAt"`
	Source         string          `json:"source"` // "jira", "manual", etc.
	TotalMembers   int             `json:"totalMembers"`
	AddedMembers   []string        `json:"addedMembers"`
	RemovedMembers []string        `json:"removedMembers"`
	Errors         []TeamSyncError `json:"errors,omitempty"`
}

// TeamSyncError represents an error that occurred during team synchronization
type TeamSyncError struct {
	Message string `json:"message"`
	Type    string `json:"type"` // "authentication", "network", "permission", etc.
}

// NewTeamSyncResult creates a new team sync result
func NewTeamSyncResult(projectKey string, members []TeamMember, source string) *TeamSyncResult {
	return &TeamSyncResult{
		ProjectKey:   projectKey,
		Members:      members,
		SyncedAt:     time.Now(),
		Source:       source,
		TotalMembers: len(members),
		Errors:       make([]TeamSyncError, 0),
	}
}

// AddError adds an error to the sync result
func (r *TeamSyncResult) AddError(message, errorType string) {
	r.Errors = append(r.Errors, TeamSyncError{
		Message: message,
		Type:    errorType,
	})
}

// HasErrors returns true if there are any errors in the sync result
func (r *TeamSyncResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// GetMemberNames extracts display names from team members for compatibility with existing structure
func (r *TeamSyncResult) GetMemberNames() []string {
	names := make([]string, len(r.Members))
	for i, member := range r.Members {
		// Prefer DisplayName, fallback to Name if empty
		if member.DisplayName != "" {
			names[i] = member.DisplayName
		} else {
			names[i] = member.Name
		}
	}
	return names
}

// Validate validates the sync result
func (r *TeamSyncResult) Validate() error {
	if r.ProjectKey == "" {
		return fmt.Errorf("project key is required")
	}

	if len(r.Members) == 0 {
		return fmt.Errorf("no team members found for project %s", r.ProjectKey)
	}

	// Validate each member
	for i, member := range r.Members {
		if member.Name == "" && member.DisplayName == "" {
			return fmt.Errorf("member at index %d has no name or display name", i)
		}
	}

	return nil
}

// String returns a string representation of the sync result
func (r *TeamSyncResult) String() string {
	status := "success"
	if r.HasErrors() {
		status = fmt.Sprintf("completed with %d errors", len(r.Errors))
	}

	return fmt.Sprintf("TeamSync[%s]: %d members from %s (%s)",
		r.ProjectKey, r.TotalMembers, r.Source, status)
}
