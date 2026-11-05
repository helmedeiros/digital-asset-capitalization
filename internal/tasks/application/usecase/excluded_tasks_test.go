package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

type stubExcludedLabels []string

func (s stubExcludedLabels) GetExcludedLabels(string) []string { return s }

func TestDropExcludedTasks(t *testing.T) {
	t.Parallel()

	tasks := []*domain.Task{
		{Key: "T-1", Labels: []string{"cap-asset-pricing"}},
		{Key: "T-2", Labels: []string{"operational", "assetcap-exclude"}},
		{Key: "T-3"},
	}

	kept, excluded := dropExcludedTasks(stubExcludedLabels{"assetcap-exclude"}, "T", tasks)
	assert.Equal(t, []*domain.Task{tasks[0], tasks[2]}, kept)
	assert.Equal(t, []string{"T-2"}, excluded)

	kept, excluded = dropExcludedTasks(stubExcludedLabels{}, "T", tasks)
	assert.Len(t, kept, 3, "no labels configured")
	assert.Empty(t, excluded)

	kept, excluded = dropExcludedTasks(nil, "T", tasks)
	assert.Len(t, kept, 3, "no provider")
	assert.Empty(t, excluded)
}
