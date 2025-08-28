package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

func TestTeamResolverService_ResolveProjectIdentifier(t *testing.T) {
	teams := map[string][]string{
		"FN": {"Alice", "Bob"},
		"AD": {"Carol", "Dave"},
	}
	nicknames := map[string][]string{
		"FN": {"pricing", "fintech"},
		"AD": {"ads", "advertisement"},
	}

	teamConfig, err := domain.NewTeamConfigWithNicknames(teams, nicknames)
	require.NoError(t, err)

	resolver := NewTeamResolverService(teamConfig)

	tests := []struct {
		name       string
		identifier string
		want       string
		wantErr    bool
	}{
		{
			name:       "resolve project code",
			identifier: "FN",
			want:       "FN",
			wantErr:    false,
		},
		{
			name:       "resolve nickname",
			identifier: "pricing",
			want:       "FN",
			wantErr:    false,
		},
		{
			name:       "resolve case insensitive nickname",
			identifier: "PRICING",
			want:       "FN",
			wantErr:    false,
		},
		{
			name:       "resolve empty identifier",
			identifier: "",
			want:       "",
			wantErr:    false,
		},
		{
			name:       "resolve unknown identifier",
			identifier: "unknown",
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.ResolveProjectIdentifier(tt.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveProjectIdentifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTeamResolverService_ResolveMultipleIdentifiers(t *testing.T) {
	teams := map[string][]string{
		"FN": {"Alice", "Bob"},
		"AD": {"Carol", "Dave"},
	}
	nicknames := map[string][]string{
		"FN": {"pricing", "fintech"},
		"AD": {"ads"},
	}

	teamConfig, err := domain.NewTeamConfigWithNicknames(teams, nicknames)
	require.NoError(t, err)

	resolver := NewTeamResolverService(teamConfig)

	tests := []struct {
		name        string
		identifiers []string
		want        []string
		wantErr     bool
	}{
		{
			name:        "resolve multiple valid identifiers",
			identifiers: []string{"FN", "pricing", "ads"},
			want:        []string{"FN", "FN", "AD"},
			wantErr:     false,
		},
		{
			name:        "resolve with empty identifiers",
			identifiers: []string{"FN", "", "ads"},
			want:        []string{"FN", "AD"},
			wantErr:     false,
		},
		{
			name:        "resolve with unknown identifier",
			identifiers: []string{"FN", "unknown"},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "resolve empty list",
			identifiers: []string{},
			want:        []string{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.ResolveMultipleIdentifiers(tt.identifiers)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveMultipleIdentifiers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTeamResolverService_GetProjectWithNicknames(t *testing.T) {
	teams := map[string][]string{
		"FN": {"Alice", "Bob"},
		"AD": {"Carol", "Dave"},
	}
	nicknames := map[string][]string{
		"FN": {"pricing", "fintech"},
		"AD": {"ads"},
	}

	teamConfig, err := domain.NewTeamConfigWithNicknames(teams, nicknames)
	require.NoError(t, err)

	resolver := NewTeamResolverService(teamConfig)

	tests := []struct {
		name    string
		project string
		want    string
	}{
		{
			name:    "project with nicknames",
			project: "FN",
			want:    "FN (Pricing, Fintech)",
		},
		{
			name:    "project with single nickname",
			project: "AD",
			want:    "AD (Ads)",
		},
		{
			name:    "project without nicknames",
			project: "UNKNOWN",
			want:    "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.GetProjectWithNicknames(tt.project)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTeamResolverService_GetAllMappings(t *testing.T) {
	teams := map[string][]string{
		"FN": {"Alice", "Bob"},
		"AD": {"Carol", "Dave"},
	}
	nicknames := map[string][]string{
		"FN": {"pricing", "fintech"},
		"AD": {"ads"},
	}

	teamConfig, err := domain.NewTeamConfigWithNicknames(teams, nicknames)
	require.NoError(t, err)

	resolver := NewTeamResolverService(teamConfig)

	expected := map[string]string{
		"pricing": "FN",
		"fintech": "FN",
		"ads":     "AD",
	}

	mappings := resolver.GetAllMappings()
	assert.Equal(t, expected, mappings)
}

func TestTeamResolverService_UpdateTeamConfig(t *testing.T) {
	// Create initial config
	teams1 := map[string][]string{
		"FN": {"Alice", "Bob"},
	}
	nicknames1 := map[string][]string{
		"FN": {"pricing"},
	}

	teamConfig1, err := domain.NewTeamConfigWithNicknames(teams1, nicknames1)
	require.NoError(t, err)

	resolver := NewTeamResolverService(teamConfig1)

	// Test initial resolution
	result, err := resolver.ResolveProjectIdentifier("pricing")
	require.NoError(t, err)
	assert.Equal(t, "FN", result)

	// Update config
	teams2 := map[string][]string{
		"AD": {"Carol", "Dave"},
	}
	nicknames2 := map[string][]string{
		"AD": {"ads"},
	}

	teamConfig2, err := domain.NewTeamConfigWithNicknames(teams2, nicknames2)
	require.NoError(t, err)

	resolver.UpdateTeamConfig(teamConfig2)

	// Test resolution with updated config
	result, err = resolver.ResolveProjectIdentifier("ads")
	require.NoError(t, err)
	assert.Equal(t, "AD", result)

	// Old nickname should no longer work
	_, err = resolver.ResolveProjectIdentifier("pricing")
	assert.Error(t, err)
}
