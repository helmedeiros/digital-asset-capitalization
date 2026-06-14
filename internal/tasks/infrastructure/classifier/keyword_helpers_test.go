package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"

	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

func TestComprehensiveClassificationChain_containsResearchKeywords(t *testing.T) {
	chain := &ComprehensiveClassificationChain{}

	t.Run("summary or description match returns true", func(t *testing.T) {
		assert.True(t, chain.containsResearchKeywords(&taskdomain.Task{
			Summary: "Spike: try the new SDK",
		}))
		assert.True(t, chain.containsResearchKeywords(&taskdomain.Task{
			Description: "Investigation into latency",
		}))
	})

	t.Run("label match returns true even with neutral summary", func(t *testing.T) {
		assert.True(t, chain.containsResearchKeywords(&taskdomain.Task{
			Summary: "ordinary work",
			Labels:  []string{"poc"},
		}))
	})

	t.Run("nothing matches returns false", func(t *testing.T) {
		assert.False(t, chain.containsResearchKeywords(&taskdomain.Task{
			Summary: "ship feature",
			Labels:  []string{"unrelated"},
		}))
	})
}

func TestComprehensiveClassificationChain_containsBugKeywords(t *testing.T) {
	chain := &ComprehensiveClassificationChain{}

	t.Run("content keyword match returns true", func(t *testing.T) {
		assert.True(t, chain.containsBugKeywords(&taskdomain.Task{Summary: "Fix the issue"}))
	})

	t.Run("Bug type returns true even without matching keywords", func(t *testing.T) {
		assert.True(t, chain.containsBugKeywords(&taskdomain.Task{
			Summary: "neutral text", Type: taskdomain.TaskTypeBug,
		}))
	})

	t.Run("neither match returns false", func(t *testing.T) {
		assert.False(t, chain.containsBugKeywords(&taskdomain.Task{
			Summary: "ship feature", Type: taskdomain.TaskTypeStory,
		}))
	})
}

func TestComprehensiveClassificationChain_containsAPIKeywords(t *testing.T) {
	chain := &ComprehensiveClassificationChain{}
	assert.True(t, chain.containsAPIKeywords(&taskdomain.Task{Summary: "Add new endpoint"}))
	assert.False(t, chain.containsAPIKeywords(&taskdomain.Task{Summary: "no relevant words"}))
}
