package usecase

import (
	"slices"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// dropExcludedTasks splits tasks into those that stay and the keys of those
// carrying one of the project's excluded labels. Without a provider or
// configured labels every task stays.
func dropExcludedTasks(provider ports.ExcludedLabelProvider, project string, tasks []*domain.Task) ([]*domain.Task, []string) {
	if provider == nil {
		return tasks, nil
	}
	labels := provider.GetExcludedLabels(project)
	if len(labels) == 0 {
		return tasks, nil
	}
	kept := make([]*domain.Task, 0, len(tasks))
	var excluded []string
	for _, task := range tasks {
		if hasExcludedLabel(task.Labels, labels) {
			excluded = append(excluded, task.Key)
			continue
		}
		kept = append(kept, task)
	}
	return kept, excluded
}

func hasExcludedLabel(have, excluded []string) bool {
	for _, label := range have {
		if slices.Contains(excluded, label) {
			return true
		}
	}
	return false
}
