package domain

import (
	"testing"
	"time"

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

func TestTeamConfig_ToMap(t *testing.T) {
	t.Run("should return empty map for empty config", func(t *testing.T) {
		config, err := NewTeamConfig(map[string][]string{})
		require.NoError(t, err)

		result := config.ToMap()
		assert.Empty(t, result, "Should return empty map")
		assert.NotNil(t, result, "Should not return nil")
	})

	t.Run("should return copy of teams map", func(t *testing.T) {
		originalTeams := map[string][]string{
			"PROJECT-A": {"Alice", "Bob"},
			"PROJECT-B": {"Charlie", "David", "Eve"},
			"PROJECT-C": {"Frank"},
		}

		config, err := NewTeamConfig(originalTeams)
		require.NoError(t, err)

		result := config.ToMap()

		// Should have same content
		assert.Equal(t, originalTeams, result, "Should return same content as original")
		assert.Len(t, result, 3, "Should have 3 projects")
		assert.Equal(t, []string{"Alice", "Bob"}, result["PROJECT-A"])
		assert.Equal(t, []string{"Charlie", "David", "Eve"}, result["PROJECT-B"])
		assert.Equal(t, []string{"Frank"}, result["PROJECT-C"])
	})

	t.Run("should return independent copy - modifications don't affect original", func(t *testing.T) {
		originalTeams := map[string][]string{
			"PROJECT-A": {"Alice", "Bob"},
		}

		config, err := NewTeamConfig(originalTeams)
		require.NoError(t, err)

		result := config.ToMap()

		// Modify the returned map
		result["PROJECT-A"] = append(result["PROJECT-A"], "Charlie")
		result["NEW-PROJECT"] = []string{"Dave"}

		// Original should be unchanged
		originalResult := config.ToMap()
		assert.Equal(t, []string{"Alice", "Bob"}, originalResult["PROJECT-A"], "Original should be unchanged")
		assert.NotContains(t, originalResult, "NEW-PROJECT", "Original should not have new project")
	})

	t.Run("should return independent copy - slice modifications don't affect original", func(t *testing.T) {
		originalTeams := map[string][]string{
			"PROJECT-A": {"Alice", "Bob"},
		}

		config, err := NewTeamConfig(originalTeams)
		require.NoError(t, err)

		result := config.ToMap()

		// Modify the slice directly
		result["PROJECT-A"][0] = "Modified"

		// Original should be unchanged
		originalResult := config.ToMap()
		assert.Equal(t, "Alice", originalResult["PROJECT-A"][0], "Original slice should be unchanged")
	})

	t.Run("should work after team modifications", func(t *testing.T) {
		config, err := NewTeamConfig(map[string][]string{
			"PROJECT-A": {"Alice"},
		})
		require.NoError(t, err)

		// Add team member
		err = config.AddTeamMember("PROJECT-A", "Bob")
		require.NoError(t, err)

		// Add new project
		err = config.AddTeamMember("PROJECT-B", "Charlie")
		require.NoError(t, err)

		result := config.ToMap()
		assert.Len(t, result, 2, "Should have 2 projects")
		assert.Contains(t, result["PROJECT-A"], "Alice")
		assert.Contains(t, result["PROJECT-A"], "Bob")
		assert.Equal(t, []string{"Charlie"}, result["PROJECT-B"])
	})

	t.Run("should handle single member teams", func(t *testing.T) {
		config, err := NewTeamConfig(map[string][]string{
			"SINGLE": {"OnlyMember"},
		})
		require.NoError(t, err)

		result := config.ToMap()
		assert.Equal(t, []string{"OnlyMember"}, result["SINGLE"])
	})

	t.Run("should preserve order within teams", func(t *testing.T) {
		// Note: Go maps don't guarantee order, but slices do
		config, err := NewTeamConfig(map[string][]string{
			"PROJECT-A": {"Alice", "Bob", "Charlie", "David"},
		})
		require.NoError(t, err)

		result := config.ToMap()
		expected := []string{"Alice", "Bob", "Charlie", "David"}
		assert.Equal(t, expected, result["PROJECT-A"], "Should preserve order within team")
	})
}

func TestTeamConfig_SetTeam(t *testing.T) {
	config, err := NewTeamConfig(map[string][]string{})
	require.NoError(t, err)

	t.Run("successful team setting", func(t *testing.T) {
		err := config.SetTeam("PROJECT-A", []string{"Alice", "Bob"})
		require.NoError(t, err)

		team, exists := config.GetTeam("PROJECT-A")
		assert.True(t, exists)
		expected := []string{"Alice", "Bob"}
		assert.Equal(t, expected, team)
	})

	t.Run("empty project key", func(t *testing.T) {
		err := config.SetTeam("", []string{"Alice"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project key cannot be empty")
	})

	t.Run("whitespace project key", func(t *testing.T) {
		err := config.SetTeam("   ", []string{"Alice"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project key cannot be empty")
	})

	t.Run("empty team member", func(t *testing.T) {
		err := config.SetTeam("PROJECT-B", []string{"Alice", ""})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team member cannot be empty")
	})

	t.Run("duplicate team member", func(t *testing.T) {
		err := config.SetTeam("PROJECT-C", []string{"Alice", "Alice"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate team member")
	})

	t.Run("trim whitespace from project and members", func(t *testing.T) {
		err := config.SetTeam("  PROJECT-D  ", []string{"  Alice  ", "  Bob  "})
		require.NoError(t, err)

		team, exists := config.GetTeam("PROJECT-D")
		assert.True(t, exists)
		expected := []string{"Alice", "Bob"}
		assert.Equal(t, expected, team)
	})

	t.Run("overwrite existing team", func(t *testing.T) {
		err := config.SetTeam("PROJECT-E", []string{"Alice"})
		require.NoError(t, err)

		err = config.SetTeam("PROJECT-E", []string{"Bob", "Charlie"})
		require.NoError(t, err)

		team, exists := config.GetTeam("PROJECT-E")
		assert.True(t, exists)
		expected := []string{"Bob", "Charlie"}
		assert.Equal(t, expected, team)
	})
}

// Tests for nickname functionality

func TestNewTeamConfigWithNicknames(t *testing.T) {
	tests := []struct {
		name      string
		teams     map[string][]string
		nicknames map[string][]string
		wantErr   bool
	}{
		{
			name: "valid teams and nicknames",
			teams: map[string][]string{
				"FN": {"Alice", "Bob"},
				"AD": {"Carol", "Dave"},
			},
			nicknames: map[string][]string{
				"FN": {"pricing", "fintech"},
				"AD": {"ads", "advertisement"},
			},
			wantErr: false,
		},
		{
			name: "nickname for non-existent team",
			teams: map[string][]string{
				"FN": {"Alice", "Bob"},
			},
			nicknames: map[string][]string{
				"UNKNOWN": {"test"},
			},
			wantErr: true,
		},
		{
			name: "duplicate nicknames for same project",
			teams: map[string][]string{
				"FN": {"Alice", "Bob"},
			},
			nicknames: map[string][]string{
				"FN": {"pricing", "pricing"},
			},
			wantErr: true,
		},
		{
			name: "empty nickname",
			teams: map[string][]string{
				"FN": {"Alice", "Bob"},
			},
			nicknames: map[string][]string{
				"FN": {""},
			},
			wantErr: true,
		},
		{
			name: "teams without nicknames",
			teams: map[string][]string{
				"FN": {"Alice", "Bob"},
			},
			nicknames: map[string][]string{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTeamConfigWithNicknames(tt.teams, tt.nicknames)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTeamConfigWithNicknames() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTeamConfig_ResolveTeamIdentifier(t *testing.T) {
	teams := map[string][]string{
		"FN": {"Alice", "Bob"},
		"AD": {"Carol", "Dave"},
	}
	nicknames := map[string][]string{
		"FN": {"pricing", "fintech"},
		"AD": {"ads", "advertisement"},
	}

	config, err := NewTeamConfigWithNicknames(teams, nicknames)
	require.NoError(t, err)

	tests := []struct {
		name       string
		identifier string
		want       string
		wantErr    bool
	}{
		{
			name:       "resolve existing project code",
			identifier: "FN",
			want:       "FN",
			wantErr:    false,
		},
		{
			name:       "resolve lowercase nickname",
			identifier: "pricing",
			want:       "FN",
			wantErr:    false,
		},
		{
			name:       "resolve uppercase nickname",
			identifier: "PRICING",
			want:       "FN",
			wantErr:    false,
		},
		{
			name:       "resolve mixed case nickname",
			identifier: "Pricing",
			want:       "FN",
			wantErr:    false,
		},
		{
			name:       "resolve another project nickname",
			identifier: "ads",
			want:       "AD",
			wantErr:    false,
		},
		{
			name:       "resolve project code case insensitive",
			identifier: "fn",
			want:       "FN",
			wantErr:    false,
		},
		{
			name:       "unknown identifier",
			identifier: "unknown",
			want:       "",
			wantErr:    true,
		},
		{
			name:       "empty identifier",
			identifier: "",
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := config.ResolveTeamIdentifier(tt.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveTeamIdentifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveTeamIdentifier() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTeamConfig_SetNicknames(t *testing.T) {
	teams := map[string][]string{
		"FN": {"Alice", "Bob"},
	}

	config, err := NewTeamConfig(teams)
	require.NoError(t, err)

	tests := []struct {
		name      string
		project   string
		nicknames []string
		wantErr   bool
	}{
		{
			name:      "set valid nicknames",
			project:   "FN",
			nicknames: []string{"pricing", "fintech"},
			wantErr:   false,
		},
		{
			name:      "set nicknames for non-existent project",
			project:   "UNKNOWN",
			nicknames: []string{"test"},
			wantErr:   true,
		},
		{
			name:      "set duplicate nicknames",
			project:   "FN",
			nicknames: []string{"pricing", "pricing"},
			wantErr:   true,
		},
		{
			name:      "set empty nickname",
			project:   "FN",
			nicknames: []string{""},
			wantErr:   true,
		},
		{
			name:      "empty project",
			project:   "",
			nicknames: []string{"test"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.SetNicknames(tt.project, tt.nicknames)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetNicknames() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTeamConfig_GetNicknames(t *testing.T) {
	teams := map[string][]string{
		"FN": {"Alice", "Bob"},
		"AD": {"Carol", "Dave"},
	}
	nicknames := map[string][]string{
		"FN": {"pricing", "fintech"},
	}

	config, err := NewTeamConfigWithNicknames(teams, nicknames)
	require.NoError(t, err)

	tests := []struct {
		name    string
		project string
		want    []string
	}{
		{
			name:    "get existing nicknames",
			project: "FN",
			want:    []string{"pricing", "fintech"},
		},
		{
			name:    "get nicknames for project without nicknames",
			project: "AD",
			want:    nil,
		},
		{
			name:    "get nicknames for non-existent project",
			project: "UNKNOWN",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.GetNicknames(tt.project)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTeamConfig_GetAllNicknameMappings(t *testing.T) {
	teams := map[string][]string{
		"FN": {"Alice", "Bob"},
		"AD": {"Carol", "Dave"},
	}
	nicknames := map[string][]string{
		"FN": {"pricing", "fintech"},
		"AD": {"ads"},
	}

	config, err := NewTeamConfigWithNicknames(teams, nicknames)
	require.NoError(t, err)

	mappings := config.GetAllNicknameMappings()
	expected := map[string]string{
		"pricing": "FN",
		"fintech": "FN",
		"ads":     "AD",
	}

	assert.Equal(t, expected, mappings)
}

func TestNewTeamConfigWithTribes(t *testing.T) {
	teams := map[string][]string{
		"FN":  {"alice", "bob"},
		"COP": {"charlie"},
	}
	nicknames := map[string][]string{
		"FN": {"pricing"},
	}
	tribes := map[string]string{
		"FN":  "Engineering",
		"COP": "Platform",
	}

	config, err := NewTeamConfigWithTribes(teams, nicknames, tribes)
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Verify teams
	fnTeam, exists := config.GetTeam("FN")
	assert.True(t, exists)
	assert.Equal(t, []string{"alice", "bob"}, fnTeam)

	// Verify nicknames
	assert.Equal(t, []string{"pricing"}, config.GetNicknames("FN"))

	// Verify tribes
	assert.Equal(t, "Engineering", config.GetTribe("FN"))
	assert.Equal(t, "Platform", config.GetTribe("COP"))
}

func TestNewTeamConfigWithTribes_IgnoresInvalidProjects(t *testing.T) {
	teams := map[string][]string{
		"FN": {"alice"},
	}
	nicknames := map[string][]string{}
	tribes := map[string]string{
		"FN":      "Engineering",
		"INVALID": "SomeTribe", // This project doesn't exist
	}

	config, err := NewTeamConfigWithTribes(teams, nicknames, tribes)
	require.NoError(t, err)

	// Valid project should have tribe
	assert.Equal(t, "Engineering", config.GetTribe("FN"))
	// Invalid project should not have tribe
	assert.Equal(t, "", config.GetTribe("INVALID"))
}

func TestGetTribe(t *testing.T) {
	teams := map[string][]string{
		"FN":  {"alice"},
		"COP": {"bob"},
	}
	tribes := map[string]string{
		"FN": "Engineering",
	}

	config, err := NewTeamConfigWithTribes(teams, nil, tribes)
	require.NoError(t, err)

	// Project with tribe
	assert.Equal(t, "Engineering", config.GetTribe("FN"))
	// Project without tribe
	assert.Equal(t, "", config.GetTribe("COP"))
	// Non-existent project
	assert.Equal(t, "", config.GetTribe("NONEXISTENT"))
}

func TestSetTribe(t *testing.T) {
	teams := map[string][]string{
		"FN": {"alice"},
	}

	config, err := NewTeamConfig(teams)
	require.NoError(t, err)

	// Set tribe for existing project
	err = config.SetTribe("FN", "Engineering")
	require.NoError(t, err)
	assert.Equal(t, "Engineering", config.GetTribe("FN"))

	// Update tribe
	err = config.SetTribe("FN", "NewTribe")
	require.NoError(t, err)
	assert.Equal(t, "NewTribe", config.GetTribe("FN"))
}

func TestSetTribe_Errors(t *testing.T) {
	teams := map[string][]string{
		"FN": {"alice"},
	}

	config, err := NewTeamConfig(teams)
	require.NoError(t, err)

	// Empty project key
	err = config.SetTribe("", "Engineering")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project key cannot be empty")

	// Non-existent project
	err = config.SetTribe("INVALID", "Engineering")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestToFullMap(t *testing.T) {
	teams := map[string][]string{
		"FN":  {"alice", "bob"},
		"COP": {"charlie"},
	}
	nicknames := map[string][]string{
		"FN": {"pricing", "fintech"},
	}
	tribes := map[string]string{
		"FN":  "Engineering",
		"COP": "Platform",
	}
	companies := map[string]string{
		"FN":  "TechCorp",
		"COP": "PartnerInc",
	}

	config, err := NewTeamConfigFull(teams, nicknames, tribes, companies)
	require.NoError(t, err)

	resultTeams, resultNicknames, resultTribes, resultCompanies := config.ToFullMap()

	// Verify teams
	assert.Equal(t, []string{"alice", "bob"}, resultTeams["FN"])
	assert.Equal(t, []string{"charlie"}, resultTeams["COP"])

	// Verify nicknames
	assert.Equal(t, []string{"pricing", "fintech"}, resultNicknames["FN"])

	// Verify tribes
	assert.Equal(t, "Engineering", resultTribes["FN"])
	assert.Equal(t, "Platform", resultTribes["COP"])

	// Verify companies
	assert.Equal(t, "TechCorp", resultCompanies["FN"])
	assert.Equal(t, "PartnerInc", resultCompanies["COP"])
}

func TestGetCompany(t *testing.T) {
	teams := map[string][]string{
		"FN":  {"alice", "bob"},
		"COP": {"charlie"},
	}
	nicknames := map[string][]string{}
	tribes := map[string]string{}
	companies := map[string]string{
		"FN": "TechCorp",
	}

	config, err := NewTeamConfigFull(teams, nicknames, tribes, companies)
	require.NoError(t, err)

	t.Run("returns company for project with company", func(t *testing.T) {
		company := config.GetCompany("FN")
		assert.Equal(t, "TechCorp", company)
	})

	t.Run("returns empty for project without company", func(t *testing.T) {
		company := config.GetCompany("COP")
		assert.Equal(t, "", company)
	})

	t.Run("returns empty for non-existent project", func(t *testing.T) {
		company := config.GetCompany("NONEXISTENT")
		assert.Equal(t, "", company)
	})
}

func TestSetCompany(t *testing.T) {
	t.Run("sets company for existing project", func(t *testing.T) {
		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		config, err := NewTeamConfig(teams)
		require.NoError(t, err)

		err = config.SetCompany("FN", "TechCorp")
		require.NoError(t, err)

		assert.Equal(t, "TechCorp", config.GetCompany("FN"))
	})

	t.Run("returns error for non-existent project", func(t *testing.T) {
		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		config, err := NewTeamConfig(teams)
		require.NoError(t, err)

		err = config.SetCompany("NONEXISTENT", "TechCorp")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("returns error for empty project", func(t *testing.T) {
		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		config, err := NewTeamConfig(teams)
		require.NoError(t, err)

		err = config.SetCompany("", "TechCorp")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("trims whitespace from company name", func(t *testing.T) {
		teams := map[string][]string{
			"FN": {"alice", "bob"},
		}
		config, err := NewTeamConfig(teams)
		require.NoError(t, err)

		err = config.SetCompany("FN", "  TechCorp  ")
		require.NoError(t, err)

		assert.Equal(t, "TechCorp", config.GetCompany("FN"))
	})
}

func TestNewTeamConfigFull(t *testing.T) {
	t.Run("creates config with all fields", func(t *testing.T) {
		teams := map[string][]string{
			"FN":  {"alice", "bob"},
			"COP": {"charlie"},
		}
		nicknames := map[string][]string{
			"FN": {"pricing"},
		}
		tribes := map[string]string{
			"FN": "Engineering",
		}
		companies := map[string]string{
			"FN":  "TechCorp",
			"COP": "PartnerInc",
		}

		config, err := NewTeamConfigFull(teams, nicknames, tribes, companies)
		require.NoError(t, err)

		// Verify teams
		fnTeam, exists := config.GetTeam("FN")
		require.True(t, exists)
		assert.Equal(t, []string{"alice", "bob"}, fnTeam)

		// Verify nicknames
		assert.Equal(t, []string{"pricing"}, config.GetNicknames("FN"))

		// Verify tribes
		assert.Equal(t, "Engineering", config.GetTribe("FN"))

		// Verify companies
		assert.Equal(t, "TechCorp", config.GetCompany("FN"))
		assert.Equal(t, "PartnerInc", config.GetCompany("COP"))
	})

	t.Run("ignores company for non-existent project", func(t *testing.T) {
		teams := map[string][]string{
			"FN": {"alice"},
		}
		companies := map[string]string{
			"NONEXISTENT": "SomeCorp",
		}

		config, err := NewTeamConfigFull(teams, nil, nil, companies)
		require.NoError(t, err)

		// Company for non-existent project should not be set
		assert.Equal(t, "", config.GetCompany("NONEXISTENT"))
	})
}

func TestGetExcludedIssueTypes(t *testing.T) {
	teams := map[string][]string{
		"COP": {"alice"},
		"FN":  {"bob"},
	}
	excludedTypes := map[string][]string{
		"COP": {"Experiment", "Spike"},
	}

	config, err := NewTeamConfigWithExcludedTypes(teams, nil, nil, nil, nil, nil, excludedTypes)
	require.NoError(t, err)

	t.Run("returns excluded types for project with config", func(t *testing.T) {
		types := config.GetExcludedIssueTypes("COP")
		assert.Equal(t, []string{"Experiment", "Spike"}, types)
	})

	t.Run("returns nil for project without excluded types", func(t *testing.T) {
		types := config.GetExcludedIssueTypes("FN")
		assert.Nil(t, types)
	})

	t.Run("returns nil for non-existent project", func(t *testing.T) {
		types := config.GetExcludedIssueTypes("NONEXISTENT")
		assert.Nil(t, types)
	})
}

func TestSetExcludedIssueTypes(t *testing.T) {
	t.Run("sets excluded types for existing project", func(t *testing.T) {
		teams := map[string][]string{"COP": {"alice"}}
		config, err := NewTeamConfig(teams)
		require.NoError(t, err)

		err = config.SetExcludedIssueTypes("COP", []string{"Experiment"})
		require.NoError(t, err)
		assert.Equal(t, []string{"Experiment"}, config.GetExcludedIssueTypes("COP"))
	})

	t.Run("returns error for non-existent project", func(t *testing.T) {
		teams := map[string][]string{"COP": {"alice"}}
		config, err := NewTeamConfig(teams)
		require.NoError(t, err)

		err = config.SetExcludedIssueTypes("NONEXISTENT", []string{"Experiment"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("returns error for empty project", func(t *testing.T) {
		teams := map[string][]string{"COP": {"alice"}}
		config, err := NewTeamConfig(teams)
		require.NoError(t, err)

		err = config.SetExcludedIssueTypes("", []string{"Experiment"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestNewTeamConfigWithExcludedTypes(t *testing.T) {
	t.Run("creates config with excluded issue types", func(t *testing.T) {
		teams := map[string][]string{
			"COP": {"alice"},
			"FN":  {"bob"},
		}
		excludedTypes := map[string][]string{
			"COP": {"Experiment"},
		}

		config, err := NewTeamConfigWithExcludedTypes(teams, nil, nil, nil, nil, nil, excludedTypes)
		require.NoError(t, err)

		assert.Equal(t, []string{"Experiment"}, config.GetExcludedIssueTypes("COP"))
		assert.Nil(t, config.GetExcludedIssueTypes("FN"))
	})

	t.Run("ignores excluded types for non-existent project", func(t *testing.T) {
		teams := map[string][]string{"COP": {"alice"}}
		excludedTypes := map[string][]string{
			"NONEXISTENT": {"Experiment"},
		}

		config, err := NewTeamConfigWithExcludedTypes(teams, nil, nil, nil, nil, nil, excludedTypes)
		require.NoError(t, err)

		assert.Nil(t, config.GetExcludedIssueTypes("NONEXISTENT"))
	})

	t.Run("round-trips through ToCompleteMapWithExcludedTypes", func(t *testing.T) {
		teams := map[string][]string{"COP": {"alice"}}
		excludedTypes := map[string][]string{"COP": {"Experiment", "Spike"}}

		config, err := NewTeamConfigWithExcludedTypes(teams, nil, nil, nil, nil, nil, excludedTypes)
		require.NoError(t, err)

		_, _, _, _, _, _, resultExcluded := config.ToCompleteMapWithExcludedTypes()
		assert.Equal(t, []string{"Experiment", "Spike"}, resultExcluded["COP"])
	})
}

// --- Team Timeline Tests ---

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func datePtr(year int, month time.Month, day int) *time.Time {
	t := date(year, month, day)
	return &t
}

func TestTeamMemberPeriod_IsActiveAt(t *testing.T) {
	tests := []struct {
		name   string
		period TeamMemberPeriod
		at     time.Time
		want   bool
	}{
		{
			name:   "active member at date after joined",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 1, 1)},
			at:     date(2024, 6, 15),
			want:   true,
		},
		{
			name:   "active member on join date",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 1, 1)},
			at:     date(2024, 1, 1),
			want:   true,
		},
		{
			name:   "before join date",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 6, 1)},
			at:     date(2024, 1, 1),
			want:   false,
		},
		{
			name:   "after left date",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 1, 1), Left: datePtr(2024, 6, 30)},
			at:     date(2024, 7, 15),
			want:   false,
		},
		{
			name:   "on left date",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 1, 1), Left: datePtr(2024, 6, 30)},
			at:     date(2024, 6, 30),
			want:   true,
		},
		{
			name:   "between joined and left",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 1, 1), Left: datePtr(2024, 6, 30)},
			at:     date(2024, 3, 15),
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.period.IsActiveAt(tt.at))
		})
	}
}

func TestTeamMemberPeriod_IsActiveDuring(t *testing.T) {
	tests := []struct {
		name   string
		period TeamMemberPeriod
		start  time.Time
		end    time.Time
		want   bool
	}{
		{
			name:   "active member overlaps entire period",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 1, 1)},
			start:  date(2024, 3, 1),
			end:    date(2024, 3, 15),
			want:   true,
		},
		{
			name:   "left before period starts",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 1, 1), Left: datePtr(2024, 2, 28)},
			start:  date(2024, 3, 1),
			end:    date(2024, 3, 15),
			want:   false,
		},
		{
			name:   "joined after period ends",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 4, 1)},
			start:  date(2024, 3, 1),
			end:    date(2024, 3, 15),
			want:   false,
		},
		{
			name:   "left during period - partial overlap",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 1, 1), Left: datePtr(2024, 3, 5)},
			start:  date(2024, 3, 1),
			end:    date(2024, 3, 15),
			want:   true,
		},
		{
			name:   "joined during period - partial overlap",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 3, 10)},
			start:  date(2024, 3, 1),
			end:    date(2024, 3, 15),
			want:   true,
		},
		{
			name:   "left on period start date",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 1, 1), Left: datePtr(2024, 3, 1)},
			start:  date(2024, 3, 1),
			end:    date(2024, 3, 15),
			want:   true,
		},
		{
			name:   "joined on period end date",
			period: TeamMemberPeriod{Member: "Alice", Joined: date(2024, 3, 15)},
			start:  date(2024, 3, 1),
			end:    date(2024, 3, 15),
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.period.IsActiveDuring(tt.start, tt.end))
		})
	}
}

func TestTeamConfig_GetTeamForPeriod(t *testing.T) {
	teams := map[string][]string{
		"PROJECT-A": {"Alice", "Bob", "Charlie", "Dave"},
		"PROJECT-B": {"Eve"},
	}

	timeline := map[string][]TeamMemberPeriod{
		"PROJECT-A": {
			{Member: "Alice", Joined: date(2024, 1, 1)},                               // still active
			{Member: "Bob", Joined: date(2024, 1, 1), Left: datePtr(2024, 6, 30)},     // left mid-year
			{Member: "Charlie", Joined: date(2024, 1, 1), Left: datePtr(2024, 3, 31)}, // left Q1
			{Member: "Dave", Joined: date(2024, 7, 1)},                                // joined mid-year
			{Member: "Eve", Joined: date(2024, 10, 1), Left: datePtr(2024, 12, 31)},   // short stint
		},
	}

	config, err := NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, nil, nil, timeline)
	require.NoError(t, err)

	t.Run("Q1 period includes Alice Bob Charlie", func(t *testing.T) {
		members, exists := config.GetTeamForPeriod("PROJECT-A", date(2024, 1, 1), date(2024, 3, 31))
		assert.True(t, exists)
		assert.ElementsMatch(t, []string{"Alice", "Bob", "Charlie"}, members)
	})

	t.Run("Q2 period includes Alice Bob", func(t *testing.T) {
		members, exists := config.GetTeamForPeriod("PROJECT-A", date(2024, 4, 1), date(2024, 6, 30))
		assert.True(t, exists)
		assert.ElementsMatch(t, []string{"Alice", "Bob"}, members)
	})

	t.Run("Q3 period includes Alice Dave", func(t *testing.T) {
		members, exists := config.GetTeamForPeriod("PROJECT-A", date(2024, 7, 1), date(2024, 9, 30))
		assert.True(t, exists)
		assert.ElementsMatch(t, []string{"Alice", "Dave"}, members)
	})

	t.Run("Q4 period includes Alice Dave Eve", func(t *testing.T) {
		members, exists := config.GetTeamForPeriod("PROJECT-A", date(2024, 10, 1), date(2024, 12, 31))
		assert.True(t, exists)
		assert.ElementsMatch(t, []string{"Alice", "Dave", "Eve"}, members)
	})

	t.Run("far future includes only active members", func(t *testing.T) {
		members, exists := config.GetTeamForPeriod("PROJECT-A", date(2025, 6, 1), date(2025, 6, 15))
		assert.True(t, exists)
		assert.ElementsMatch(t, []string{"Alice", "Dave"}, members)
	})

	t.Run("project without timeline falls back to flat team", func(t *testing.T) {
		members, exists := config.GetTeamForPeriod("PROJECT-B", date(2024, 1, 1), date(2024, 3, 31))
		assert.True(t, exists)
		assert.Equal(t, []string{"Eve"}, members)
	})

	t.Run("non-existent project", func(t *testing.T) {
		members, exists := config.GetTeamForPeriod("UNKNOWN", date(2024, 1, 1), date(2024, 3, 31))
		assert.False(t, exists)
		assert.Nil(t, members)
	})
}

func TestTeamConfig_AddMemberWithDates(t *testing.T) {
	teams := map[string][]string{"PROJECT-A": {"Alice"}}
	config, err := NewTeamConfig(teams)
	require.NoError(t, err)

	t.Run("add member with join date", func(t *testing.T) {
		err := config.AddMemberWithDates("PROJECT-A", "Bob", date(2024, 6, 1))
		require.NoError(t, err)

		timeline := config.GetTeamTimeline("PROJECT-A")
		require.Len(t, timeline, 1)
		assert.Equal(t, "Bob", timeline[0].Member)
		assert.Equal(t, date(2024, 6, 1), timeline[0].Joined)
		assert.Nil(t, timeline[0].Left)
	})

	t.Run("add duplicate active member fails", func(t *testing.T) {
		err := config.AddMemberWithDates("PROJECT-A", "Bob", date(2024, 7, 1))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already has an active period")
	})

	t.Run("empty project", func(t *testing.T) {
		err := config.AddMemberWithDates("", "Bob", date(2024, 6, 1))
		assert.Error(t, err)
	})

	t.Run("empty member", func(t *testing.T) {
		err := config.AddMemberWithDates("PROJECT-A", "", date(2024, 6, 1))
		assert.Error(t, err)
	})

	t.Run("non-existent project", func(t *testing.T) {
		err := config.AddMemberWithDates("UNKNOWN", "Bob", date(2024, 6, 1))
		assert.Error(t, err)
	})
}

func TestTeamConfig_SetMemberLeft(t *testing.T) {
	teams := map[string][]string{"PROJECT-A": {"Alice"}}
	timeline := map[string][]TeamMemberPeriod{
		"PROJECT-A": {
			{Member: "Alice", Joined: date(2024, 1, 1)},
			{Member: "Bob", Joined: date(2024, 1, 1), Left: datePtr(2024, 6, 30)},
		},
	}

	config, err := NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, nil, nil, timeline)
	require.NoError(t, err)

	t.Run("set left date for active member", func(t *testing.T) {
		err := config.SetMemberLeft("PROJECT-A", "Alice", date(2024, 12, 31))
		require.NoError(t, err)

		tl := config.GetTeamTimeline("PROJECT-A")
		for _, p := range tl {
			if p.Member == "Alice" {
				require.NotNil(t, p.Left)
				assert.Equal(t, date(2024, 12, 31), *p.Left)
			}
		}
	})

	t.Run("cannot set left for already departed member", func(t *testing.T) {
		err := config.SetMemberLeft("PROJECT-A", "Bob", date(2024, 12, 31))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no active period found")
	})

	t.Run("empty project", func(t *testing.T) {
		err := config.SetMemberLeft("", "Alice", date(2024, 12, 31))
		assert.Error(t, err)
	})

	t.Run("project without timeline", func(t *testing.T) {
		err := config.SetMemberLeft("UNKNOWN", "Alice", date(2024, 12, 31))
		assert.Error(t, err)
	})
}

func TestTeamConfig_DeriveActiveTeamFromTimeline(t *testing.T) {
	teams := map[string][]string{"PROJECT-A": {"Alice", "Bob", "Charlie"}}
	timeline := map[string][]TeamMemberPeriod{
		"PROJECT-A": {
			{Member: "Alice", Joined: date(2024, 1, 1)},
			{Member: "Bob", Joined: date(2024, 1, 1), Left: datePtr(2025, 6, 30)},
			{Member: "Charlie", Joined: date(2024, 3, 1)},
		},
	}

	config, err := NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, nil, nil, timeline)
	require.NoError(t, err)

	active := config.DeriveActiveTeamFromTimeline("PROJECT-A")
	assert.ElementsMatch(t, []string{"Alice", "Charlie"}, active)
}

func TestTeamConfig_HasTeamTimeline(t *testing.T) {
	teams := map[string][]string{
		"PROJECT-A": {"Alice"},
		"PROJECT-B": {"Bob"},
	}
	timeline := map[string][]TeamMemberPeriod{
		"PROJECT-A": {{Member: "Alice", Joined: date(2024, 1, 1)}},
	}

	config, err := NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, nil, nil, timeline)
	require.NoError(t, err)

	assert.True(t, config.HasTeamTimeline("PROJECT-A"))
	assert.False(t, config.HasTeamTimeline("PROJECT-B"))
	assert.False(t, config.HasTeamTimeline("UNKNOWN"))
}

func TestNewTeamConfigWithTimelines(t *testing.T) {
	t.Run("creates config with timelines", func(t *testing.T) {
		teams := map[string][]string{"PROJECT-A": {"Alice"}}
		timeline := map[string][]TeamMemberPeriod{
			"PROJECT-A": {{Member: "Alice", Joined: date(2024, 1, 1)}},
		}

		config, err := NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, nil, nil, timeline)
		require.NoError(t, err)
		assert.True(t, config.HasTeamTimeline("PROJECT-A"))
	})

	t.Run("ignores timeline for non-existent project", func(t *testing.T) {
		teams := map[string][]string{"PROJECT-A": {"Alice"}}
		timeline := map[string][]TeamMemberPeriod{
			"UNKNOWN": {{Member: "Bob", Joined: date(2024, 1, 1)}},
		}

		config, err := NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, nil, nil, timeline)
		require.NoError(t, err)
		assert.False(t, config.HasTeamTimeline("UNKNOWN"))
	})

	t.Run("nil timeline is fine", func(t *testing.T) {
		teams := map[string][]string{"PROJECT-A": {"Alice"}}

		config, err := NewTeamConfigWithTimelines(teams, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.False(t, config.HasTeamTimeline("PROJECT-A"))

		// Falls back to flat team
		members, exists := config.GetTeamForPeriod("PROJECT-A", date(2024, 1, 1), date(2024, 3, 31))
		assert.True(t, exists)
		assert.Equal(t, []string{"Alice"}, members)
	})
}
