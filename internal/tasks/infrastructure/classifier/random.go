package classifier

import (
	"math/rand"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// RandomClassifier implements TaskClassifier using random classification
type RandomClassifier struct {
	rng *rand.Rand
}

// NewRandomClassifier creates a new random classifier
func NewRandomClassifier() *RandomClassifier {
	return &RandomClassifier{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ClassifyTask randomly assigns a work type to a task
func (c *RandomClassifier) ClassifyTask(task *domain.Task) (domain.WorkType, error) {
	workTypes := []domain.WorkType{
		domain.WorkTypeMaintenance,
		domain.WorkTypeDiscovery,
		domain.WorkTypeDevelopment,
	}

	return workTypes[c.rng.Intn(len(workTypes))], nil
}

// ClassifyTasks randomly assigns work types to multiple tasks
func (c *RandomClassifier) ClassifyTasks(tasks []*domain.Task) (map[string]domain.WorkType, error) {
	result := make(map[string]domain.WorkType)

	for _, task := range tasks {
		workType, err := c.ClassifyTask(task)
		if err != nil {
			return nil, err
		}
		result[task.Key] = workType
	}

	return result, nil
}
