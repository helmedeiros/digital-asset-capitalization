package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

func TestHonourExistingOpExLabel(t *testing.T) {
	t.Parallel()

	newUseCase := func(provider CapitalizationLabelProvider) *ClassifyTasksUseCase {
		uc := NewClassifyTasksUseCase(new(MockTaskRepository), new(MockTaskRepository), new(MockTaskClassifier), new(MockUserInput), testutil.NewMockAssetService(), nil)
		if provider != nil {
			uc.SetCapitalizationLabelProvider(provider)
		}
		return uc
	}
	provider := stubCapitalizationLabelProvider{opEx: map[string]string{"PRI": "IAS38-OpEx"}}

	result := func(project string, wt domain.WorkType, labels ...string) *ports.ComprehensiveClassificationResult {
		task, err := domain.NewTask("PRI-1", "summary", project, "S1", "jira")
		assert.NoError(t, err)
		task.Labels = labels
		return &ports.ComprehensiveClassificationResult{Task: task, WorkType: wt}
	}

	t.Run("downgrades development to maintenance when task already carries the OpEx label", func(t *testing.T) {
		r := result("PRI", domain.WorkTypeDevelopment, "operational", "IAS38-OpEx")
		newUseCase(provider).honourExistingOpExLabel([]*ports.ComprehensiveClassificationResult{r})
		assert.Equal(t, domain.WorkTypeMaintenance, r.WorkType)
	})

	t.Run("keeps discovery untouched", func(t *testing.T) {
		r := result("PRI", domain.WorkTypeDiscovery, "IAS38-OpEx")
		newUseCase(provider).honourExistingOpExLabel([]*ports.ComprehensiveClassificationResult{r})
		assert.Equal(t, domain.WorkTypeDiscovery, r.WorkType)
	})

	t.Run("leaves development alone without the OpEx label", func(t *testing.T) {
		r := result("PRI", domain.WorkTypeDevelopment, "IAS38-CapEx")
		newUseCase(provider).honourExistingOpExLabel([]*ports.ComprehensiveClassificationResult{r})
		assert.Equal(t, domain.WorkTypeDevelopment, r.WorkType)
	})

	t.Run("does nothing for projects without a configured label or provider", func(t *testing.T) {
		other := result("OTHER", domain.WorkTypeDevelopment, "IAS38-OpEx")
		newUseCase(provider).honourExistingOpExLabel([]*ports.ComprehensiveClassificationResult{other})
		assert.Equal(t, domain.WorkTypeDevelopment, other.WorkType)

		noProvider := result("PRI", domain.WorkTypeDevelopment, "IAS38-OpEx")
		newUseCase(nil).honourExistingOpExLabel([]*ports.ComprehensiveClassificationResult{noProvider})
		assert.Equal(t, domain.WorkTypeDevelopment, noProvider.WorkType)
	})
}
