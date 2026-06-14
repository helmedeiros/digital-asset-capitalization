package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
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

func TestComprehensiveClassificationChainWithInheritance_containsResearchKeywords(t *testing.T) {
	chain := &ComprehensiveClassificationChainWithInheritance{}

	assert.True(t, chain.containsResearchKeywords(&taskdomain.Task{Summary: "Spike: new SDK"}))
	assert.True(t, chain.containsResearchKeywords(&taskdomain.Task{
		Summary: "neutral",
		Labels:  []string{"discovery"},
	}))
	assert.False(t, chain.containsResearchKeywords(&taskdomain.Task{
		Summary: "ship", Labels: []string{"other"},
	}))
}

func TestComprehensiveClassificationChainWithInheritance_containsBugKeywords(t *testing.T) {
	chain := &ComprehensiveClassificationChainWithInheritance{}

	assert.True(t, chain.containsBugKeywords(&taskdomain.Task{Summary: "Fix the issue"}))
	assert.True(t, chain.containsBugKeywords(&taskdomain.Task{
		Summary: "neutral", Type: taskdomain.TaskTypeBug,
	}))
	assert.False(t, chain.containsBugKeywords(&taskdomain.Task{
		Summary: "ship feature", Type: taskdomain.TaskTypeStory,
	}))
}

func TestComprehensiveClassificationChainWithInheritance_containsAPIKeywords(t *testing.T) {
	chain := &ComprehensiveClassificationChainWithInheritance{}
	assert.True(t, chain.containsAPIKeywords(&taskdomain.Task{Summary: "Implement new service"}))
	assert.False(t, chain.containsAPIKeywords(&taskdomain.Task{Summary: "unrelated words"}))
}

func TestComprehensiveClassificationChain_generateWorkTypeReason(t *testing.T) {
	chain := &ComprehensiveClassificationChain{}

	assetWithName := &ports.AssetClassificationResult{
		Asset: &assetdomain.Asset{Name: "Payment Gateway"},
	}

	t.Run("discovery + research keywords", func(t *testing.T) {
		got := chain.generateWorkTypeReason(
			&taskdomain.Task{Summary: "Spike: explore"},
			nil, taskdomain.WorkTypeDiscovery)
		assert.Equal(t, "spike/research task detected", got)
	})

	t.Run("discovery without keywords falls through", func(t *testing.T) {
		got := chain.generateWorkTypeReason(
			&taskdomain.Task{Summary: "neutral"}, nil, taskdomain.WorkTypeDiscovery)
		assert.Equal(t, "discovery work classification", got)
	})

	t.Run("maintenance + bug keywords", func(t *testing.T) {
		got := chain.generateWorkTypeReason(
			&taskdomain.Task{Summary: "Fix bug"}, nil, taskdomain.WorkTypeMaintenance)
		assert.Equal(t, "bug fix or maintenance task", got)
	})

	t.Run("maintenance + asset name", func(t *testing.T) {
		got := chain.generateWorkTypeReason(
			&taskdomain.Task{Summary: "neutral"}, assetWithName, taskdomain.WorkTypeMaintenance)
		assert.Equal(t, "maintenance work for Payment Gateway", got)
	})

	t.Run("maintenance fallback", func(t *testing.T) {
		got := chain.generateWorkTypeReason(
			&taskdomain.Task{Summary: "neutral"}, nil, taskdomain.WorkTypeMaintenance)
		assert.Equal(t, "maintenance work classification", got)
	})

	t.Run("development + API keywords", func(t *testing.T) {
		got := chain.generateWorkTypeReason(
			&taskdomain.Task{Summary: "Build new endpoint"}, nil, taskdomain.WorkTypeDevelopment)
		assert.Equal(t, "new API or feature development", got)
	})

	t.Run("development + asset name", func(t *testing.T) {
		got := chain.generateWorkTypeReason(
			&taskdomain.Task{Summary: "neutral"}, assetWithName, taskdomain.WorkTypeDevelopment)
		assert.Equal(t, "development work for Payment Gateway", got)
	})

	t.Run("development fallback", func(t *testing.T) {
		got := chain.generateWorkTypeReason(
			&taskdomain.Task{Summary: "neutral"}, nil, taskdomain.WorkTypeDevelopment)
		assert.Equal(t, "development work classification", got)
	})

	t.Run("unknown work type falls to default", func(t *testing.T) {
		got := chain.generateWorkTypeReason(
			&taskdomain.Task{Summary: "neutral"}, nil, taskdomain.WorkType("unknown"))
		assert.Equal(t, "default work type classification", got)
	})
}

func TestComprehensiveClassificationChainWithInheritance_generateWorkTypeReason(t *testing.T) {
	chain := &ComprehensiveClassificationChainWithInheritance{}

	assetWithName := &ports.AssetClassificationResult{
		Asset: &assetdomain.Asset{Name: "Search"},
	}

	cases := []struct {
		name  string
		task  *taskdomain.Task
		asset *ports.AssetClassificationResult
		wt    taskdomain.WorkType
		want  string
	}{
		{"discovery + research", &taskdomain.Task{Summary: "Spike investigate"}, nil, taskdomain.WorkTypeDiscovery, "spike/research task detected"},
		{"discovery fallback", &taskdomain.Task{Summary: "neutral"}, nil, taskdomain.WorkTypeDiscovery, "discovery work classification"},
		{"maintenance + bug", &taskdomain.Task{Summary: "Fix it"}, nil, taskdomain.WorkTypeMaintenance, "bug fix or maintenance task"},
		{"maintenance + asset", &taskdomain.Task{Summary: "neutral"}, assetWithName, taskdomain.WorkTypeMaintenance, "maintenance work for Search"},
		{"maintenance fallback", &taskdomain.Task{Summary: "neutral"}, nil, taskdomain.WorkTypeMaintenance, "maintenance work classification"},
		{"development + API", &taskdomain.Task{Summary: "new endpoint"}, nil, taskdomain.WorkTypeDevelopment, "new API or feature development"},
		{"development + asset", &taskdomain.Task{Summary: "neutral"}, assetWithName, taskdomain.WorkTypeDevelopment, "development work for Search"},
		{"development fallback", &taskdomain.Task{Summary: "neutral"}, nil, taskdomain.WorkTypeDevelopment, "development work classification"},
		{"default branch", &taskdomain.Task{Summary: "x"}, nil, taskdomain.WorkType("other"), "default work type classification"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, chain.generateWorkTypeReason(c.task, c.asset, c.wt))
		})
	}
}
