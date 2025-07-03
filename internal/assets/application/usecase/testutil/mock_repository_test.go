package testutil

import (
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/stretchr/testify/assert"
)

func TestMockAssetRepository_Coverage(t *testing.T) {
	repo := NewMockAssetRepository()
	asset := &domain.Asset{ID: "1", Name: "A"}
	assert.NoError(t, repo.Save(asset))
	found, err := repo.FindByName("A")
	assert.NoError(t, err)
	assert.Equal(t, asset, found)
	all, err := repo.FindAll()
	assert.NoError(t, err)
	assert.Len(t, all, 1)
	foundByID, err := repo.FindByID("1")
	assert.NoError(t, err)
	assert.Equal(t, asset, foundByID)
	assert.NoError(t, repo.Delete("A"))
	_, err = repo.FindByName("A")
	assert.Error(t, err)
}
