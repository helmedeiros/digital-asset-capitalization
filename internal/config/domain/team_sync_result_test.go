package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTeamSyncResult(t *testing.T) {
	t.Parallel()
	members := []TeamMember{
		{Name: "John Doe", Email: "john@example.com", DisplayName: "John Doe"},
		{Name: "Jane Smith", Email: "jane@example.com", DisplayName: "Jane Smith"},
	}

	result := NewTeamSyncResult("TEST", members, "jira")

	assert.Equal(t, "TEST", result.ProjectKey)
	assert.Equal(t, members, result.Members)
	assert.Equal(t, "jira", result.Source)
	assert.Equal(t, 2, result.TotalMembers)
	assert.WithinDuration(t, time.Now(), result.SyncedAt, time.Second)
	assert.Empty(t, result.Errors)
	assert.False(t, result.HasErrors())
}

func TestTeamSyncResult_AddError(t *testing.T) {
	t.Parallel()
	result := NewTeamSyncResult("TEST", []TeamMember{}, "jira")

	result.AddError("Connection failed", "network")
	result.AddError("Permission denied", "permission")

	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors, 2)
	assert.Equal(t, "Connection failed", result.Errors[0].Message)
	assert.Equal(t, "network", result.Errors[0].Type)
	assert.Equal(t, "Permission denied", result.Errors[1].Message)
	assert.Equal(t, "permission", result.Errors[1].Type)
}

func TestTeamSyncResult_GetMemberNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		members  []TeamMember
		expected []string
	}{
		{
			name: "prefers display name over name",
			members: []TeamMember{
				{Name: "john.doe", DisplayName: "John Doe"},
				{Name: "jane.smith", DisplayName: "Jane Smith"},
			},
			expected: []string{"John Doe", "Jane Smith"},
		},
		{
			name: "falls back to name when display name is empty",
			members: []TeamMember{
				{Name: "john.doe", DisplayName: ""},
				{Name: "jane.smith", DisplayName: "Jane Smith"},
			},
			expected: []string{"john.doe", "Jane Smith"},
		},
		{
			name:     "empty members",
			members:  []TeamMember{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewTeamSyncResult("TEST", tt.members, "jira")
			names := result.GetMemberNames()
			assert.Equal(t, tt.expected, names)
		})
	}
}

func TestTeamSyncResult_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		result  *TeamSyncResult
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid result",
			result: &TeamSyncResult{
				ProjectKey: "TEST",
				Members: []TeamMember{
					{Name: "John Doe", DisplayName: "John Doe"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty project key",
			result: &TeamSyncResult{
				ProjectKey: "",
				Members: []TeamMember{
					{Name: "John Doe", DisplayName: "John Doe"},
				},
			},
			wantErr: true,
			errMsg:  "project key is required",
		},
		{
			name: "no members",
			result: &TeamSyncResult{
				ProjectKey: "TEST",
				Members:    []TeamMember{},
			},
			wantErr: true,
			errMsg:  "no team members found for project TEST",
		},
		{
			name: "member with no name or display name",
			result: &TeamSyncResult{
				ProjectKey: "TEST",
				Members: []TeamMember{
					{Name: "", DisplayName: "", Email: "test@example.com"},
				},
			},
			wantErr: true,
			errMsg:  "member at index 0 has no name or display name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTeamSyncResult_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		result   *TeamSyncResult
		expected string
	}{
		{
			name: "success without errors",
			result: &TeamSyncResult{
				ProjectKey:   "TEST",
				TotalMembers: 3,
				Source:       "jira",
				Errors:       []TeamSyncError{},
			},
			expected: "TeamSync[TEST]: 3 members from jira (success)",
		},
		{
			name: "with errors",
			result: &TeamSyncResult{
				ProjectKey:   "TEST",
				TotalMembers: 2,
				Source:       "jira",
				Errors: []TeamSyncError{
					{Message: "Network error", Type: "network"},
					{Message: "Permission denied", Type: "permission"},
				},
			},
			expected: "TeamSync[TEST]: 2 members from jira (completed with 2 errors)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
