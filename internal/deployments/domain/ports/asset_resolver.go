package ports

import (
	"context"
)

// AssetInfo contains basic asset information resolved from tasks
type AssetInfo struct {
	Name      string
	TaskCount int
	TaskKeys  []string
}

// AssetResolver defines the interface for resolving assets from tasks
type AssetResolver interface {
	// ResolveAssetsForTasks resolves asset associations for given task keys
	// It uses dry-run classification to determine which assets are affected
	ResolveAssetsForTasks(ctx context.Context, taskKeys []string) ([]AssetInfo, error)

	// ResolveAssetsForTask resolves asset associations for a single task
	ResolveAssetsForTask(ctx context.Context, taskKey string) ([]string, error)
}
