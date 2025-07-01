package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTeamConfig(t *testing.T) {
	tests := []struct {
		name    string
		teams   map[string][]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid team configuration",
			teams: map[string][]string{
				"FN": {"helio.medeiros", "julio.medeiros"},
				"MZ": {"alice.smith", "bob.jones"},
			},
			wantErr: false,
		},
		{
			name:    "empty configuration",
			teams:   map[string][]string{},
			wantErr: false,
		},
		{
			name: "empty project key",
			teams: map[string][]string{
				"":   {"helio.medeiros"},
				"FN": {"julio.medeiros"},
			},
			wantErr: true,
			errMsg:  "project key cannot be empty",
		},
		{
			name: "empty team member",
			teams: map[string][]string{
				"FN": {"helio.medeiros", "", "julio.medeiros"},
			},
			wantErr: true,
			errMsg:  "team member cannot be empty",
		},
		{
			name: "duplicate team members in same project",
			teams: map[string][]string{
				"FN": {"helio.medeiros", "helio.medeiros"},
			},
			wantErr: true,
			errMsg:  "duplicate team member",
		},
		{
			name: "whitespace in project key",
			teams: map[string][]string{
				"  FN  ": {"helio.medeiros"},
			},
			wantErr: false, // Should be trimmed
		},
		{
			name: "whitespace in team member",
			teams: map[string][]string{
				"FN": {"  helio.medeiros  "},
			},
			wantErr: false, // Should be trimmed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewTeamConfig(tt.teams)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
			}
		})
	}
}

func TestTeamConfig_GetTeam(t *testing.T) {
	teams := map[string][]string{
		"FN": {"helio.medeiros", "julio.medeiros"},
		"MZ": {"alice.smith", "bob.jones"},
	}

	config, err := NewTeamConfig(teams)
	require.NoError(t, err)

	t.Run("existing project", func(t *testing.T) {
		team, exists := config.GetTeam("FN")
		assert.True(t, exists)
		assert.Equal(t, []string{"helio.medeiros", "julio.medeiros"}, team)
	})

	t.Run("non-existing project", func(t *testing.T) {
		team, exists := config.GetTeam("XYZ")
		assert.False(t, exists)
		assert.Nil(t, team)
	})
}

func TestTeamConfig_GetProjects(t *testing.T) {
	teams := map[string][]string{
		"FN": {"helio.medeiros", "julio.medeiros"},
		"MZ": {"alice.smith", "bob.jones"},
	}

	config, err := NewTeamConfig(teams)
	require.NoError(t, err)

	projects := config.GetProjects()
	assert.Len(t, projects, 2)
	assert.Contains(t, projects, "FN")
	assert.Contains(t, projects, "MZ")
}

func TestTeamConfig_IsTeamMember(t *testing.T) {
	teams := map[string][]string{
		"FN": {"helio.medeiros", "julio.medeiros"},
		"MZ": {"alice.smith", "bob.jones"},
	}

	config, err := NewTeamConfig(teams)
	require.NoError(t, err)

	t.Run("existing member in project", func(t *testing.T) {
		assert.True(t, config.IsTeamMember("FN", "helio.medeiros"))
		assert.True(t, config.IsTeamMember("MZ", "alice.smith"))
	})

	t.Run("non-existing member in project", func(t *testing.T) {
		assert.False(t, config.IsTeamMember("FN", "alice.smith"))
		assert.False(t, config.IsTeamMember("MZ", "helio.medeiros"))
	})

	t.Run("non-existing project", func(t *testing.T) {
		assert.False(t, config.IsTeamMember("XYZ", "helio.medeiros"))
	})
}

func TestTeamConfig_AddTeamMember(t *testing.T) {
	teams := map[string][]string{
		"FN": {"helio.medeiros"},
	}

	config, err := NewTeamConfig(teams)
	require.NoError(t, err)

	t.Run("add to existing project", func(t *testing.T) {
		err := config.AddTeamMember("FN", "julio.medeiros")
		require.NoError(t, err)

		team, exists := config.GetTeam("FN")
		assert.True(t, exists)
		assert.Contains(t, team, "julio.medeiros")
	})

	t.Run("add to new project", func(t *testing.T) {
		err := config.AddTeamMember("MZ", "alice.smith")
		require.NoError(t, err)

		team, exists := config.GetTeam("MZ")
		assert.True(t, exists)
		assert.Equal(t, []string{"alice.smith"}, team)
	})

	t.Run("add duplicate member", func(t *testing.T) {
		err := config.AddTeamMember("FN", "helio.medeiros")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("add empty member", func(t *testing.T) {
		err := config.AddTeamMember("FN", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("add to empty project", func(t *testing.T) {
		err := config.AddTeamMember("", "user")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestTeamConfig_RemoveTeamMember(t *testing.T) {
	teams := map[string][]string{
		"FN": {"helio.medeiros", "julio.medeiros"},
	}

	config, err := NewTeamConfig(teams)
	require.NoError(t, err)

	t.Run("remove existing member", func(t *testing.T) {
		err := config.RemoveTeamMember("FN", "julio.medeiros")
		require.NoError(t, err)

		team, exists := config.GetTeam("FN")
		assert.True(t, exists)
		assert.NotContains(t, team, "julio.medeiros")
		assert.Contains(t, team, "helio.medeiros")
	})

	t.Run("remove non-existing member", func(t *testing.T) {
		err := config.RemoveTeamMember("FN", "alice.smith")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("remove from non-existing project", func(t *testing.T) {
		err := config.RemoveTeamMember("XYZ", "helio.medeiros")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project not found")
	})
}

func TestTeamConfig_IsEmpty(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config, err := NewTeamConfig(map[string][]string{})
		require.NoError(t, err)
		assert.True(t, config.IsEmpty())
	})

	t.Run("non-empty config", func(t *testing.T) {
		teams := map[string][]string{
			"FN": {"helio.medeiros"},
		}
		config, err := NewTeamConfig(teams)
		require.NoError(t, err)
		assert.False(t, config.IsEmpty())
	})
}
