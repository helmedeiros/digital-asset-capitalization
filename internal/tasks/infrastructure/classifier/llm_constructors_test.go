package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/llama"
)

func TestNewComprehensiveClassificationChainWithLLM(t *testing.T) {
	// Asset and work-type classifiers can both be nil for the
	// constructor smoke test -- we only need to verify the chain is
	// built with the LLM classifier wired into its struct.
	chain := NewComprehensiveClassificationChainWithLLM(nil, nil, nil)
	require.NotNil(t, chain)
	concrete, ok := chain.(*ComprehensiveClassificationChainWithInheritance)
	require.True(t, ok, "constructor should return *ComprehensiveClassificationChainWithInheritance")
	assert.NotNil(t, concrete.taskLookup, "taskLookup map should be initialised")
}

func TestComprehensiveClassificationChainWithInheritance_SetLLMEnabled(t *testing.T) {
	chain := NewComprehensiveClassificationChainWithLLM(nil, nil, nil).(*ComprehensiveClassificationChainWithInheritance)
	chain.SetLLMEnabled(true)
	assert.True(t, chain.llmEnabled)
	chain.SetLLMEnabled(false)
	assert.False(t, chain.llmEnabled)
}

func TestComprehensiveClassifierAdapter_SetLLMEnabled_DelegatesWhenInheritanceChain(t *testing.T) {
	inner := NewComprehensiveClassificationChainWithLLM(nil, nil, nil).(*ComprehensiveClassificationChainWithInheritance)
	adapter := NewComprehensiveClassifierAdapter(inner).(*ComprehensiveClassifierAdapter)
	adapter.SetLLMEnabled(true)
	assert.True(t, inner.llmEnabled, "adapter should forward the toggle")
}

func TestComprehensiveClassifierAdapter_SetLLMEnabled_NoopForOtherChains(t *testing.T) {
	// The non-inheritance chain doesn't carry an llmEnabled field --
	// the adapter's type assertion fails and the call is a documented
	// silent no-op. Pin that it doesn't panic.
	inner := NewComprehensiveClassificationChain(nil, nil)
	adapter := NewComprehensiveClassifierAdapter(inner).(*ComprehensiveClassifierAdapter)
	require.NotPanics(t, func() { adapter.SetLLMEnabled(true) })
}

func TestNewOllamaEmbeddingAdapter(t *testing.T) {
	client, err := llama.NewClient(llama.Config{BaseURL: "http://example.invalid"})
	require.NoError(t, err)

	adapter := NewOllamaEmbeddingAdapter(client, "nomic-embed-text")
	require.NotNil(t, adapter)
	concrete, ok := adapter.(*OllamaEmbeddingAdapter)
	require.True(t, ok)
	assert.Same(t, client, concrete.client)
	assert.Equal(t, "nomic-embed-text", concrete.model)
}
