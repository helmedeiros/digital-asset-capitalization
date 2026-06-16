package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmbeddingAssetClassifierWithHistory_WiresAllFields(t *testing.T) {
	historical := map[string][]string{"cap-asset-search": {"Search ranking task"}}
	epicNames := map[string]string{"COP-2": "Improve checkout"}
	epicHints := map[string]string{"COP-19": "cap-asset-carrier-optimization"}

	classifier := NewEmbeddingAssetClassifierWithHistory(nil, nil, nil, historical, epicNames, epicHints)
	require.NotNil(t, classifier)

	concrete, ok := classifier.(*EmbeddingAssetClassifier)
	require.True(t, ok, "constructor should return *EmbeddingAssetClassifier")
	assert.Equal(t, historical, concrete.historicalTasks)
	assert.Equal(t, epicNames, concrete.epicNames)
	assert.Equal(t, epicHints, concrete.epicAssetHint)
}

func TestNewEmbeddingAssetClassifierWithHistory_NilMapsAreFine(t *testing.T) {
	classifier := NewEmbeddingAssetClassifierWithHistory(nil, nil, nil, nil, nil, nil)
	require.NotNil(t, classifier)
	concrete, ok := classifier.(*EmbeddingAssetClassifier)
	require.True(t, ok)
	assert.Nil(t, concrete.historicalTasks)
	assert.Nil(t, concrete.epicNames)
	assert.Nil(t, concrete.epicAssetHint)
}
