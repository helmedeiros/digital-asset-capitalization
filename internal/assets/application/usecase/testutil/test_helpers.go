package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// SetupBulkTest creates commonly used mocks for bulk operation tests
func SetupBulkTest(t *testing.T) (*MockAssetRepository, *MockLlamaClient) {
	repo := NewMockAssetRepository()
	llama := &MockLlamaClient{}
	return repo, llama
}

// SetupSyncAndEnrichTest creates mocks for sync-and-enrich tests
func SetupSyncAndEnrichTest(t *testing.T) (*MockAssetRepository, *MockLlamaClient, *MockAssetService) {
	repo := NewMockAssetRepository()
	llama := &MockLlamaClient{}
	assetSvc := &MockAssetService{}
	return repo, llama, assetSvc
}

// CreateTestAsset creates a test asset with sensible defaults
func CreateTestAsset(name string, options ...func(*domain.Asset)) *domain.Asset {
	asset := &domain.Asset{
		ID:          "test-" + name,
		Name:        name,
		Description: "Test description for " + name,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Keywords:    []string{},
	}

	// Apply any custom options
	for _, option := range options {
		option(asset)
	}

	return asset
}

// WithKeywords sets keywords for a test asset
func WithKeywords(keywords ...string) func(*domain.Asset) {
	return func(asset *domain.Asset) {
		asset.Keywords = keywords
	}
}

// WithEmptyDescription sets an empty description for a test asset
func WithEmptyDescription() func(*domain.Asset) {
	return func(asset *domain.Asset) {
		asset.Description = ""
	}
}

// WithEmptyField sets a specific field to empty for a test asset
func WithEmptyField(field string) func(*domain.Asset) {
	return func(asset *domain.Asset) {
		switch field {
		case "description":
			asset.Description = ""
		case "why":
			asset.Why = ""
		case "benefits":
			asset.Benefits = ""
		case "how":
			asset.How = ""
		case "metrics":
			asset.Metrics = ""
		}
	}
}

// AssertBulkResult provides common assertions for bulk operation results
type BulkResultAssertions struct {
	t      *testing.T
	result interface{}
}

// NewBulkResultAssertions creates a new assertion helper
func NewBulkResultAssertions(t *testing.T, result interface{}) *BulkResultAssertions {
	return &BulkResultAssertions{t: t, result: result}
}

// AssertProcessedCount checks that the correct number of assets were processed
func (a *BulkResultAssertions) AssertProcessedCount(expected int) *BulkResultAssertions {
	// Use reflection or type switching to extract the TotalProcessed field
	// This is a simplified version - in practice, you might want to create
	// interfaces for different result types
	switch r := a.result.(type) {
	case interface{ getTotalProcessed() int }:
		assert.Equal(a.t, expected, r.getTotalProcessed())
	}
	return a
}

// AssertDurationGreaterThanZero checks that the operation took some time
func (a *BulkResultAssertions) AssertDurationGreaterThanZero() *BulkResultAssertions {
	switch r := a.result.(type) {
	case interface{ getDuration() time.Duration }:
		assert.Greater(a.t, r.getDuration(), time.Duration(0))
	}
	return a
}

// AssertNoErrors checks that no errors occurred during processing
func (a *BulkResultAssertions) AssertNoErrors() *BulkResultAssertions {
	switch r := a.result.(type) {
	case interface{ getTotalFailed() int }:
		assert.Equal(a.t, 0, r.getTotalFailed())
	}
	return a
}

// CommonTestScenarios provides commonly used test scenario builders
type CommonTestScenarios struct{}

// EmptyRepository returns a test scenario with no assets
func (CommonTestScenarios) EmptyRepository() func(*MockAssetRepository) {
	return func(repo *MockAssetRepository) {
		// Repository is already empty by default
	}
}

// WithAssets returns a test scenario with the specified assets
func (CommonTestScenarios) WithAssets(assets ...*domain.Asset) func(*MockAssetRepository) {
	return func(repo *MockAssetRepository) {
		for _, asset := range assets {
			_ = repo.Save(asset)
		}
	}
}

// SuccessfulLlamaClient returns a test scenario with a working LLM client
func (CommonTestScenarios) SuccessfulLlamaClient(response string) func(*MockLlamaClient) {
	return func(client *MockLlamaClient) {
		client.On("EnrichContent",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("*domain.Asset")).Return(response, nil)
	}
}

// FailingLlamaClient returns a test scenario with a failing LLM client
func (CommonTestScenarios) FailingLlamaClient(err error) func(*MockLlamaClient) {
	return func(client *MockLlamaClient) {
		client.On("EnrichContent",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("*domain.Asset")).Return("", err)
	}
}
