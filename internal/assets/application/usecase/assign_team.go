package usecase

import (
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// AssignTeamUseCase handles assigning teams to assets
type AssignTeamUseCase struct {
	repository ports.AssetRepository
}

// NewAssignTeamUseCase creates a new AssignTeamUseCase
func NewAssignTeamUseCase(repository ports.AssetRepository) *AssignTeamUseCase {
	return &AssignTeamUseCase{
		repository: repository,
	}
}

// AssignTeamInput contains the input parameters for assigning teams to an asset
type AssignTeamInput struct {
	AssetName         string   `json:"asset_name"`
	OwningTeam        string   `json:"owning_team,omitempty"`
	ContributingTeams []string `json:"contributing_teams,omitempty"`
}

// Execute assigns teams to an asset
func (uc *AssignTeamUseCase) Execute(input AssignTeamInput) error {
	if input.AssetName == "" {
		return fmt.Errorf("asset name is required")
	}

	// Find the asset
	asset, err := uc.repository.FindByName(input.AssetName)
	if err != nil {
		return fmt.Errorf("failed to find asset: %w", err)
	}

	// Update owning team if provided
	if input.OwningTeam != "" {
		if err := asset.SetOwningTeam(input.OwningTeam); err != nil {
			return fmt.Errorf("failed to set owning team: %w", err)
		}
	}

	// Update contributing teams if provided
	if len(input.ContributingTeams) > 0 {
		if err := asset.SetContributingTeams(input.ContributingTeams); err != nil {
			return fmt.Errorf("failed to set contributing teams: %w", err)
		}
	}

	// Save the updated asset
	if err := uc.repository.Save(asset); err != nil {
		return fmt.Errorf("failed to save asset: %w", err)
	}

	return nil
}

// GetAssetTeamsUseCase handles retrieving team assignments for assets
type GetAssetTeamsUseCase struct {
	repository ports.AssetRepository
}

// NewGetAssetTeamsUseCase creates a new GetAssetTeamsUseCase
func NewGetAssetTeamsUseCase(repository ports.AssetRepository) *GetAssetTeamsUseCase {
	return &GetAssetTeamsUseCase{
		repository: repository,
	}
}

// AssetTeamInfo contains team information for an asset
type AssetTeamInfo struct {
	AssetName         string   `json:"asset_name"`
	OwningTeam        string   `json:"owning_team"`
	ContributingTeams []string `json:"contributing_teams"`
}

// Execute retrieves team assignments for all assets
func (uc *GetAssetTeamsUseCase) Execute() ([]AssetTeamInfo, error) {
	assets, err := uc.repository.FindAll()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve assets: %w", err)
	}

	var result []AssetTeamInfo
	for _, asset := range assets {
		// Only include assets that have team assignments
		if asset.GetOwningTeam() != "" || len(asset.GetContributingTeams()) > 0 {
			info := AssetTeamInfo{
				AssetName:         asset.Name,
				OwningTeam:        asset.GetOwningTeam(),
				ContributingTeams: asset.GetContributingTeams(),
			}
			result = append(result, info)
		}
	}

	return result, nil
}

// GetAssetTeamInfoUseCase handles retrieving team assignments for a specific asset
type GetAssetTeamInfoUseCase struct {
	repository ports.AssetRepository
}

// NewGetAssetTeamInfoUseCase creates a new GetAssetTeamInfoUseCase
func NewGetAssetTeamInfoUseCase(repository ports.AssetRepository) *GetAssetTeamInfoUseCase {
	return &GetAssetTeamInfoUseCase{
		repository: repository,
	}
}

// Execute retrieves team assignments for a specific asset
func (uc *GetAssetTeamInfoUseCase) Execute(assetName string) (*AssetTeamInfo, error) {
	if assetName == "" {
		return nil, fmt.Errorf("asset name is required")
	}

	asset, err := uc.repository.FindByName(assetName)
	if err != nil {
		return nil, fmt.Errorf("failed to find asset: %w", err)
	}

	info := &AssetTeamInfo{
		AssetName:         asset.Name,
		OwningTeam:        asset.GetOwningTeam(),
		ContributingTeams: asset.GetContributingTeams(),
	}

	return info, nil
}

// AddContributingTeamUseCase handles adding a contributing team to an asset
type AddContributingTeamUseCase struct {
	repository ports.AssetRepository
}

// NewAddContributingTeamUseCase creates a new AddContributingTeamUseCase
func NewAddContributingTeamUseCase(repository ports.AssetRepository) *AddContributingTeamUseCase {
	return &AddContributingTeamUseCase{
		repository: repository,
	}
}

// Execute adds a contributing team to an asset
func (uc *AddContributingTeamUseCase) Execute(assetName, teamName string) error {
	if assetName == "" {
		return fmt.Errorf("asset name is required")
	}
	if teamName == "" {
		return fmt.Errorf("team name is required")
	}

	// Find the asset
	asset, err := uc.repository.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("failed to find asset: %w", err)
	}

	// Add the contributing team
	if err := asset.AddContributingTeam(teamName); err != nil {
		return fmt.Errorf("failed to add contributing team: %w", err)
	}

	// Save the updated asset
	if err := uc.repository.Save(asset); err != nil {
		return fmt.Errorf("failed to save asset: %w", err)
	}

	return nil
}

// RemoveContributingTeamUseCase handles removing a contributing team from an asset
type RemoveContributingTeamUseCase struct {
	repository ports.AssetRepository
}

// NewRemoveContributingTeamUseCase creates a new RemoveContributingTeamUseCase
func NewRemoveContributingTeamUseCase(repository ports.AssetRepository) *RemoveContributingTeamUseCase {
	return &RemoveContributingTeamUseCase{
		repository: repository,
	}
}

// Execute removes a contributing team from an asset
func (uc *RemoveContributingTeamUseCase) Execute(assetName, teamName string) error {
	if assetName == "" {
		return fmt.Errorf("asset name is required")
	}
	if teamName == "" {
		return fmt.Errorf("team name is required")
	}

	// Find the asset
	asset, err := uc.repository.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("failed to find asset: %w", err)
	}

	// Remove the contributing team
	if err := asset.RemoveContributingTeam(teamName); err != nil {
		return fmt.Errorf("failed to remove contributing team: %w", err)
	}

	// Save the updated asset
	if err := uc.repository.Save(asset); err != nil {
		return fmt.Errorf("failed to save asset: %w", err)
	}

	return nil
}

// ParseTeamsInput parses comma-separated team names
func ParseTeamsInput(teamsInput string) []string {
	if teamsInput == "" {
		return nil
	}
	
	teams := strings.Split(teamsInput, ",")
	var cleanTeams []string
	for _, team := range teams {
		if trimmed := strings.TrimSpace(team); trimmed != "" {
			cleanTeams = append(cleanTeams, trimmed)
		}
	}
	return cleanTeams
}