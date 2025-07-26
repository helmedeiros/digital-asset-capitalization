package testutil

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// MockAssetService is a mock implementation of AssetService for testing
type MockAssetService struct {
	createAssetFunc             func(name, description string) error
	listAssetsFunc              func() ([]*domain.Asset, error)
	getAssetFunc                func(identifier string) (*domain.Asset, error)
	deleteAssetFunc             func(name string) error
	updateAssetFunc             func(name, description, why, benefits, how, metrics string) error
	updateDocumentationFunc     func(assetName string) error
	incrementTaskCountFunc      func(name string) error
	decrementTaskCountFunc      func(name string) error
	syncFromConfluenceFunc      func(spaceKey, label string, debug bool) (*domain.SyncResult, error)
	enrichAssetFunc             func(name, field string) error
	generateKeywordsFunc        func(name string) error
	assignTeamFunc              func(assetName, owningTeam string, contributingTeams []string) error
	getAssetTeamsFunc           func() ([]application.AssetTeamInfo, error)
	getAssetTeamInfoFunc        func(assetName string) (*application.AssetTeamInfo, error)
	addContributingTeamFunc     func(assetName, teamName string) error
	removeContributingTeamFunc  func(assetName, teamName string) error
}

// NewMockAssetService creates a new mock asset service
func NewMockAssetService() *MockAssetService {
	return &MockAssetService{}
}

// CreateAsset creates a new asset
func (m *MockAssetService) CreateAsset(name, description string) error {
	if m.createAssetFunc != nil {
		return m.createAssetFunc(name, description)
	}
	return nil
}

// ListAssets returns a list of all assets
func (m *MockAssetService) ListAssets() ([]*domain.Asset, error) {
	if m.listAssetsFunc != nil {
		return m.listAssetsFunc()
	}
	return []*domain.Asset{}, nil
}

// GetAsset returns an asset by name
func (m *MockAssetService) GetAsset(identifier string) (*domain.Asset, error) {
	if m.getAssetFunc != nil {
		return m.getAssetFunc(identifier)
	}
	return &domain.Asset{Name: identifier}, nil
}

// DeleteAsset deletes an asset by name
func (m *MockAssetService) DeleteAsset(name string) error {
	if m.deleteAssetFunc != nil {
		return m.deleteAssetFunc(name)
	}
	return nil
}

// UpdateAsset updates an asset's name and description
func (m *MockAssetService) UpdateAsset(name, description, why, benefits, how, metrics string) error {
	if m.updateAssetFunc != nil {
		return m.updateAssetFunc(name, description, why, benefits, how, metrics)
	}
	return nil
}

// UpdateDocumentation marks the documentation for an asset as updated
func (m *MockAssetService) UpdateDocumentation(assetName string) error {
	if m.updateDocumentationFunc != nil {
		return m.updateDocumentationFunc(assetName)
	}
	return nil
}

// IncrementTaskCount increments the task count for an asset
func (m *MockAssetService) IncrementTaskCount(name string) error {
	if m.incrementTaskCountFunc != nil {
		return m.incrementTaskCountFunc(name)
	}
	return nil
}

// DecrementTaskCount decrements the task count for an asset
func (m *MockAssetService) DecrementTaskCount(name string) error {
	if m.decrementTaskCountFunc != nil {
		return m.decrementTaskCountFunc(name)
	}
	return nil
}

// SyncFromConfluence fetches assets from Confluence and updates the local repository
func (m *MockAssetService) SyncFromConfluence(spaceKey, label string, debug bool) (*domain.SyncResult, error) {
	if m.syncFromConfluenceFunc != nil {
		return m.syncFromConfluenceFunc(spaceKey, label, debug)
	}
	return domain.NewSyncResult(), nil
}

// EnrichAsset enriches a specific field of an asset using LLaMA 3
func (m *MockAssetService) EnrichAsset(name, field string) error {
	if m.enrichAssetFunc != nil {
		return m.enrichAssetFunc(name, field)
	}
	return nil
}

// GenerateKeywords generates keywords for an asset using LLaMA
func (m *MockAssetService) GenerateKeywords(name string) error {
	if m.generateKeywordsFunc != nil {
		return m.generateKeywordsFunc(name)
	}
	return nil
}

// AssignTeam assigns owning and contributing teams to an asset
func (m *MockAssetService) AssignTeam(assetName, owningTeam string, contributingTeams []string) error {
	if m.assignTeamFunc != nil {
		return m.assignTeamFunc(assetName, owningTeam, contributingTeams)
	}
	return nil
}

// GetAssetTeams returns team assignments for all assets
func (m *MockAssetService) GetAssetTeams() ([]application.AssetTeamInfo, error) {
	if m.getAssetTeamsFunc != nil {
		return m.getAssetTeamsFunc()
	}
	return []application.AssetTeamInfo{}, nil
}

// GetAssetTeamInfo returns team assignments for a specific asset
func (m *MockAssetService) GetAssetTeamInfo(assetName string) (*application.AssetTeamInfo, error) {
	if m.getAssetTeamInfoFunc != nil {
		return m.getAssetTeamInfoFunc(assetName)
	}
	return &application.AssetTeamInfo{AssetName: assetName}, nil
}

// AddContributingTeam adds a contributing team to an asset
func (m *MockAssetService) AddContributingTeam(assetName, teamName string) error {
	if m.addContributingTeamFunc != nil {
		return m.addContributingTeamFunc(assetName, teamName)
	}
	return nil
}

// RemoveContributingTeam removes a contributing team from an asset
func (m *MockAssetService) RemoveContributingTeam(assetName, teamName string) error {
	if m.removeContributingTeamFunc != nil {
		return m.removeContributingTeamFunc(assetName, teamName)
	}
	return nil
}

// Setup methods for tests
func (m *MockAssetService) SetSyncFromConfluenceFunc(f func(spaceKey, label string, debug bool) (*domain.SyncResult, error)) {
	m.syncFromConfluenceFunc = f
}

func (m *MockAssetService) SetGetAssetFunc(f func(identifier string) (*domain.Asset, error)) {
	m.getAssetFunc = f
}

func (m *MockAssetService) SetListAssetsFunc(f func() ([]*domain.Asset, error)) {
	m.listAssetsFunc = f
}

// Reset resets all mock functions
func (m *MockAssetService) Reset() {
	m.createAssetFunc = nil
	m.listAssetsFunc = nil
	m.getAssetFunc = nil
	m.deleteAssetFunc = nil
	m.updateAssetFunc = nil
	m.updateDocumentationFunc = nil
	m.incrementTaskCountFunc = nil
	m.decrementTaskCountFunc = nil
	m.syncFromConfluenceFunc = nil
	m.enrichAssetFunc = nil
	m.generateKeywordsFunc = nil
	m.assignTeamFunc = nil
	m.getAssetTeamsFunc = nil
	m.getAssetTeamInfoFunc = nil
	m.addContributingTeamFunc = nil
	m.removeContributingTeamFunc = nil
}

// Compile time check to ensure MockAssetService implements AssetService
var _ application.AssetService = (*MockAssetService)(nil)
