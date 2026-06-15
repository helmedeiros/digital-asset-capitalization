package application

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// TestEnrichAsset_AllFieldBranches drives EnrichAsset through every
// supported `field` value. The existing TestEnrichAsset table only
// covered the "description" branch, leaving why/benefits/how/metrics
// unexecuted in both the read- and write-switch.
func TestEnrichAsset_AllFieldBranches(t *testing.T) {
	cases := []struct {
		field  string
		seed   string
		seeder func(a *domain.Asset, v string)
		reader func(a *domain.Asset) string
	}{
		{"why", "original why",
			func(a *domain.Asset, v string) { a.Why = v },
			func(a *domain.Asset) string { return a.Why }},
		{"benefits", "original benefits",
			func(a *domain.Asset, v string) { a.Benefits = v },
			func(a *domain.Asset) string { return a.Benefits }},
		{"how", "original how",
			func(a *domain.Asset, v string) { a.How = v },
			func(a *domain.Asset) string { return a.How }},
		{"metrics", "original metrics",
			func(a *domain.Asset, v string) { a.Metrics = v },
			func(a *domain.Asset) string { return a.Metrics }},
	}

	for _, c := range cases {
		t.Run(c.field, func(t *testing.T) {
			asset := &domain.Asset{
				ID: "id-1", Name: "Search",
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
				Version: 7,
			}
			c.seeder(asset, c.seed)

			mockRepo := new(MockAssetRepository)
			mockLlama := new(MockLlamaClient)

			mockRepo.On("FindByName", "Search").Return(asset, nil)
			mockLlama.On("EnrichContent", c.seed, c.field, asset).Return("enriched "+c.field, nil)
			mockRepo.On("Save", mock.MatchedBy(func(a *domain.Asset) bool {
				return c.reader(a) == "enriched "+c.field && a.Version == 8
			})).Return(nil)

			svc := &AssetServiceImpl{repo: mockRepo, llama: mockLlama}
			require.NoError(t, svc.EnrichAsset("Search", c.field))
			assert.Equal(t, "enriched "+c.field, c.reader(asset))

			mockRepo.AssertExpectations(t)
			mockLlama.AssertExpectations(t)
		})
	}
}

// TestEnrichAsset_LlamaErrorWraps drives the EnrichContent failure
// branch — the asset is loaded successfully but the LLM call errors.
func TestEnrichAsset_LlamaErrorWraps(t *testing.T) {
	asset := &domain.Asset{ID: "id-1", Name: "Search", Description: "x", Version: 1}
	mockRepo := new(MockAssetRepository)
	mockLlama := new(MockLlamaClient)

	mockRepo.On("FindByName", "Search").Return(asset, nil)
	mockLlama.On("EnrichContent", "x", "description", asset).
		Return("", errors.New("model offline"))

	svc := &AssetServiceImpl{repo: mockRepo, llama: mockLlama}
	err := svc.EnrichAsset("Search", "description")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to enrich content")
	assert.Contains(t, err.Error(), "model offline")
}

// TestEnrichAsset_SaveErrorPropagates covers the final s.repo.Save
// failure branch — the asset and enrichment succeed but persistence
// fails.
func TestEnrichAsset_SaveErrorPropagates(t *testing.T) {
	asset := &domain.Asset{ID: "id-1", Name: "Search", Description: "x", Version: 1}
	mockRepo := new(MockAssetRepository)
	mockLlama := new(MockLlamaClient)

	mockRepo.On("FindByName", "Search").Return(asset, nil)
	mockLlama.On("EnrichContent", "x", "description", asset).Return("y", nil)
	mockRepo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(errors.New("disk full"))

	svc := &AssetServiceImpl{repo: mockRepo, llama: mockLlama}
	err := svc.EnrichAsset("Search", "description")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}
