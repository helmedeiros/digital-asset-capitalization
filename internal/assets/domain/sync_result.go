package domain

// SyncResult represents the result of a sync operation
type SyncResult struct {
	SyncedAssets    []*Asset
	NotSyncedAssets []*NotSyncedAsset
}

// NotSyncedAsset represents an asset that couldn't be synced due to missing information
type NotSyncedAsset struct {
	Name            string
	MissingFields   []string
	AvailableFields map[string]string
}

// NewSyncResult creates a new SyncResult instance
func NewSyncResult() *SyncResult {
	return &SyncResult{
		SyncedAssets:    make([]*Asset, 0),
		NotSyncedAssets: make([]*NotSyncedAsset, 0),
	}
}
