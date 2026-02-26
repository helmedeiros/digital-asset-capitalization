package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	sprintDomain "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	sprintPorts "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// Mock implementations for testing
type mockSprintPort struct {
	mock.Mock
}

func (m *mockSprintPort) GetSprintsForProject(project string, states []string) ([]sprintPorts.Sprint, error) {
	args := m.Called(project, states)
	return args.Get(0).([]sprintPorts.Sprint), args.Error(1)
}

func (m *mockSprintPort) GetSprintsForProjectWithBoardInfo(project string, states []string) ([]sprintPorts.Sprint, []sprintPorts.BoardInfo, error) {
	args := m.Called(project, states)
	return args.Get(0).([]sprintPorts.Sprint), args.Get(1).([]sprintPorts.BoardInfo), args.Error(2)
}

func (m *mockSprintPort) GetIssuesForSprint(project, sprintID string) ([]sprintPorts.JiraIssue, error) {
	args := m.Called(project, sprintID)
	return args.Get(0).([]sprintPorts.JiraIssue), args.Error(1)
}

func (m *mockSprintPort) GetIssuesForTeamMember(member string) ([]sprintPorts.JiraIssue, error) {
	args := m.Called(member)
	return args.Get(0).([]sprintPorts.JiraIssue), args.Error(1)
}

func (m *mockSprintPort) GetSprintIssues(sprint *sprintDomain.Sprint) ([]sprintPorts.JiraIssue, error) {
	args := m.Called(sprint)
	return args.Get(0).([]sprintPorts.JiraIssue), args.Error(1)
}

func (m *mockSprintPort) GetTeamIssues(team *sprintDomain.Team) ([]sprintPorts.JiraIssue, error) {
	args := m.Called(team)
	return args.Get(0).([]sprintPorts.JiraIssue), args.Error(1)
}

func (m *mockSprintPort) GetIssuesForSprintOnBoard(project, sprintName string, boardID int) ([]sprintPorts.JiraIssue, error) {
	args := m.Called(project, sprintName, boardID)
	return args.Get(0).([]sprintPorts.JiraIssue), args.Error(1)
}

func (m *mockSprintPort) GetSprintByName(project, sprintName string) (*sprintPorts.Sprint, error) {
	args := m.Called(project, sprintName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sprintPorts.Sprint), args.Error(1)
}

type mockSelectionPort struct {
	mock.Mock
}

func (m *mockSelectionPort) SelectSprint(candidates []ports.SprintCandidate) (*sprintPorts.Sprint, error) {
	args := m.Called(candidates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sprintPorts.Sprint), args.Error(1)
}

func TestSprintResolver_ResolveSprint(t *testing.T) {
	ctx := context.Background()

	t.Run("empty sprint name should return error", func(t *testing.T) {
		sprintPort := &mockSprintPort{}
		selectionPort := &mockSelectionPort{}
		resolver := NewSprintResolver(sprintPort, selectionPort)

		_, err := resolver.ResolveSprint(ctx, "PROJECT", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sprint name is required")
	})

	t.Run("exact match should return sprint ID directly", func(t *testing.T) {
		sprintPort := &mockSprintPort{}
		selectionPort := &mockSelectionPort{}
		resolver := NewSprintResolver(sprintPort, selectionPort)

		expectedSprint := &sprintPorts.Sprint{
			ID:   "123",
			Name: "Sprint 1",
		}

		sprintPort.On("GetSprintByName", "PROJECT", "Sprint 1").Return(expectedSprint, nil)

		result, err := resolver.ResolveSprint(ctx, "PROJECT", "Sprint 1")

		assert.NoError(t, err)
		assert.Equal(t, "Sprint 1", result)
		sprintPort.AssertExpectations(t)
		selectionPort.AssertExpectations(t)
	})

	t.Run("exact match failure should trigger fuzzy search", func(t *testing.T) {
		sprintPort := &mockSprintPort{}
		selectionPort := &mockSelectionPort{}
		resolver := NewSprintResolver(sprintPort, selectionPort)

		allSprints := []sprintPorts.Sprint{
			{ID: "456", Name: "🇵🇦 Panama", State: "closed"},
			{ID: "789", Name: "🇵🇸 Palestine State", State: "closed"},
		}

		sprintPort.On("GetSprintByName", "PROJECT", "Panama").Return(nil, errors.New("not found"))
		sprintPort.On("GetSprintsForProject", "PROJECT", []string{}).Return(allSprints, nil)

		result, err := resolver.ResolveSprint(ctx, "PROJECT", "Panama")

		assert.NoError(t, err)
		assert.Equal(t, "🇵🇦 Panama", result) // Should find Panama match
		sprintPort.AssertExpectations(t)
	})

	t.Run("multiple fuzzy matches should use interactive selection", func(t *testing.T) {
		sprintPort := &mockSprintPort{}
		selectionPort := &mockSelectionPort{}
		resolver := NewSprintResolver(sprintPort, selectionPort)

		allSprints := []sprintPorts.Sprint{
			{ID: "456", Name: "🇵🇦 Panama", State: "closed"},
			{ID: "789", Name: "Panama City", State: "active"},
		}

		selectedSprint := &sprintPorts.Sprint{ID: "789", Name: "Panama City", State: "active"}

		sprintPort.On("GetSprintByName", "PROJECT", "Panama").Return(nil, errors.New("not found"))
		sprintPort.On("GetSprintsForProject", "PROJECT", []string{}).Return(allSprints, nil)
		selectionPort.On("SelectSprint", mock.AnythingOfType("[]ports.SprintCandidate")).Return(selectedSprint, nil)

		result, err := resolver.ResolveSprint(ctx, "PROJECT", "Panama")

		assert.NoError(t, err)
		assert.Equal(t, "Panama City", result)
		sprintPort.AssertExpectations(t)
		selectionPort.AssertExpectations(t)
	})

	t.Run("no fuzzy matches should return error", func(t *testing.T) {
		sprintPort := &mockSprintPort{}
		selectionPort := &mockSelectionPort{}
		resolver := NewSprintResolver(sprintPort, selectionPort)

		allSprints := []sprintPorts.Sprint{
			{ID: "456", Name: "Completely Different", State: "closed"},
		}

		sprintPort.On("GetSprintByName", "PROJECT", "Panama").Return(nil, errors.New("not found"))
		sprintPort.On("GetSprintsForProject", "PROJECT", []string{}).Return(allSprints, nil)

		_, err := resolver.ResolveSprint(ctx, "PROJECT", "Panama")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no sprints matching 'Panama' found")
		sprintPort.AssertExpectations(t)
	})

	t.Run("user cancelled selection should return error", func(t *testing.T) {
		sprintPort := &mockSprintPort{}
		selectionPort := &mockSelectionPort{}
		resolver := NewSprintResolver(sprintPort, selectionPort)

		allSprints := []sprintPorts.Sprint{
			{ID: "456", Name: "🇵🇦 Panama", State: "closed"},
			{ID: "789", Name: "Panama City", State: "active"},
		}

		sprintPort.On("GetSprintByName", "PROJECT", "Panama").Return(nil, errors.New("not found"))
		sprintPort.On("GetSprintsForProject", "PROJECT", []string{}).Return(allSprints, nil)
		selectionPort.On("SelectSprint", mock.AnythingOfType("[]ports.SprintCandidate")).Return(nil, nil) // User cancelled

		_, err := resolver.ResolveSprint(ctx, "PROJECT", "Panama")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sprint selection cancelled by user")
		sprintPort.AssertExpectations(t)
		selectionPort.AssertExpectations(t)
	})
}

func TestSprintResolver_NormalizeSprintName(t *testing.T) {
	resolver := NewSprintResolver(nil, nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "flag emoji should be removed",
			input:    "🇵🇦 Panama",
			expected: "panama",
		},
		{
			name:     "multiple emojis should be removed",
			input:    "🎯🚀 Sprint One",
			expected: "sprint one",
		},
		{
			name:     "extra spaces should be normalized",
			input:    "  Sprint    Name  ",
			expected: "sprint name",
		},
		{
			name:     "mixed case should be lowercased",
			input:    "Panama CITY Sprint",
			expected: "panama city sprint",
		},
		{
			name:     "already normalized should remain unchanged",
			input:    "simple name",
			expected: "simple name",
		},
		{
			name:     "complex emoji and text mix",
			input:    "🇺🇸 🎯 Project    Alpha 🚀",
			expected: "project alpha",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := resolver.normalizeSprintName(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestSprintResolver_RemoveEmojis(t *testing.T) {
	resolver := NewSprintResolver(nil, nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "flag emojis should be removed",
			input:    "🇺🇸🇬🇧 Countries",
			expected: "Countries",
		},
		{
			name:     "common emojis should be removed",
			input:    "🎯🚀⭐ Sprint",
			expected: "Sprint",
		},
		{
			name:     "mixed content should preserve text",
			input:    "🇵🇦 Panama 🎯 Target",
			expected: "Panama  Target",
		},
		{
			name:     "no emojis should remain unchanged",
			input:    "Plain Text",
			expected: "Plain Text",
		},
		{
			name:     "only emojis should return empty",
			input:    "🎯🚀🇺🇸",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := resolver.removeEmojis(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestSprintResolver_CalculateSimilarity(t *testing.T) {
	resolver := NewSprintResolver(nil, nil)

	tests := []struct {
		name        string
		s1, s2      string
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "identical strings should return 1.0",
			s1:          "panama",
			s2:          "panama",
			expectedMin: 1.0,
			expectedMax: 1.0,
		},
		{
			name:        "completely different should return low similarity",
			s1:          "panama",
			s2:          "xyz",
			expectedMin: 0.0,
			expectedMax: 0.3,
		},
		{
			name:        "similar strings should return high similarity",
			s1:          "panama",
			s2:          "panam",
			expectedMin: 0.8,
			expectedMax: 1.0,
		},
		{
			name:        "partial match should return moderate similarity",
			s1:          "panama city",
			s2:          "panama",
			expectedMin: 0.5,
			expectedMax: 0.8,
		},
		{
			name:        "empty strings should return 0.0",
			s1:          "",
			s2:          "panama",
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := resolver.calculateSimilarity(test.s1, test.s2)
			assert.GreaterOrEqual(t, result, test.expectedMin, "similarity too low")
			assert.LessOrEqual(t, result, test.expectedMax, "similarity too high")
		})
	}
}

func TestSprintResolver_FindSprintCandidates(t *testing.T) {
	resolver := NewSprintResolver(nil, nil)

	sprints := []sprintPorts.Sprint{
		{ID: "1", Name: "🇵🇦 Panama"},
		{ID: "2", Name: "Panama City"},
		{ID: "3", Name: "🇵🇸 Palestine State"},
		{ID: "4", Name: "Completely Different"},
		{ID: "5", Name: "Pan American"},
	}

	t.Run("exact match after normalization", func(t *testing.T) {
		candidates := resolver.findSprintCandidates(sprints, "Panama")

		require.NotEmpty(t, candidates)
		// Should find both Panama entries
		found := false
		for _, candidate := range candidates {
			if candidate.Sprint.Name == "🇵🇦 Panama" && candidate.Reason == reasonExactMatch {
				found = true
				break
			}
		}
		assert.True(t, found, "should find exact match for Panama")
	})

	t.Run("contains match", func(t *testing.T) {
		candidates := resolver.findSprintCandidates(sprints, "Pan")

		require.NotEmpty(t, candidates)
		// Should find Panama and Pan American
		foundPanama := false
		foundPanAmerican := false
		for _, candidate := range candidates {
			if candidate.Sprint.Name == "🇵🇦 Panama" {
				foundPanama = true
				assert.Contains(t, candidate.Reason, "contains 'Pan'")
			}
			if candidate.Sprint.Name == "Pan American" {
				foundPanAmerican = true
			}
		}
		assert.True(t, foundPanama, "should find Panama")
		assert.True(t, foundPanAmerican, "should find Pan American")
	})

	t.Run("no matches", func(t *testing.T) {
		candidates := resolver.findSprintCandidates(sprints, "XYZ")

		assert.Empty(t, candidates)
	})

	t.Run("candidates should be sorted with exact matches first", func(t *testing.T) {
		candidates := resolver.findSprintCandidates(sprints, "Panama")

		require.NotEmpty(t, candidates)
		// First candidate should be exact match
		if len(candidates) > 1 {
			exactMatches := 0
			for i, candidate := range candidates {
				if candidate.Reason == reasonExactMatch {
					exactMatches++
					if i > 0 {
						// All exact matches should come before non-exact matches
						assert.Equal(t, reasonExactMatch, candidates[i-1].Reason, "exact matches should be sorted first")
					}
				}
			}
		}
	})
}

func TestSprintResolver_LevenshteinDistance(t *testing.T) {
	resolver := NewSprintResolver(nil, nil)

	tests := []struct {
		name     string
		s1, s2   string
		expected int
	}{
		{
			name:     "identical strings",
			s1:       "panama",
			s2:       "panama",
			expected: 0,
		},
		{
			name:     "one character difference",
			s1:       "panama",
			s2:       "panam",
			expected: 1,
		},
		{
			name:     "complete replacement",
			s1:       "abc",
			s2:       "xyz",
			expected: 3,
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			expected: 0,
		},
		{
			name:     "one empty string",
			s1:       "",
			s2:       "panama",
			expected: 6,
		},
		{
			name:     "insertion operations",
			s1:       "pan",
			s2:       "panama",
			expected: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := resolver.levenshteinDistance(test.s1, test.s2)
			assert.Equal(t, test.expected, result)
		})
	}
}
