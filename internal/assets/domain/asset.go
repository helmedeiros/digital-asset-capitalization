package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// Common errors that can occur when working with assets
var (
	ErrEmptyName        = errors.New("asset name cannot be empty")
	ErrEmptyDescription = errors.New("asset description cannot be empty")
)

// Asset represents a digital asset in the system
type Asset struct {
	// mu protects all fields below
	mu sync.RWMutex `json:"-"`
	// ID is a unique identifier for the asset
	ID string `json:"id"`
	// Name is the display name of the asset
	Name string `json:"name"`
	// Description provides detailed information about the asset
	Description string `json:"description"`
	// CreatedAt is when the asset was first created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when the asset was last modified
	UpdatedAt time.Time `json:"updated_at"`
	// LastDocUpdateAt is when the asset's documentation was last updated
	LastDocUpdateAt time.Time `json:"last_doc_update_at"`
	// AssociatedTaskCount tracks how many tasks are linked to this asset
	AssociatedTaskCount int `json:"associated_task_count"`
	// Version is used for optimistic locking
	Version int `json:"version"`
}

// NewAsset creates a new Asset instance
func NewAsset(name, description string) (*Asset, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	if description == "" {
		return nil, ErrEmptyDescription
	}

	now := time.Now()
	return &Asset{
		ID:                  generateID(name),
		Name:                name,
		Description:         description,
		CreatedAt:           now,
		UpdatedAt:           now,
		LastDocUpdateAt:     now,
		AssociatedTaskCount: 0,
		Version:             1,
	}, nil
}

// UpdateDescription updates the asset's description
func (a *Asset) UpdateDescription(description string) error {
	if description == "" {
		return ErrEmptyDescription
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Description = description
	a.UpdatedAt = time.Now()
	a.Version++
	return nil
}

// UpdateDocumentation marks the asset's documentation as updated
func (a *Asset) UpdateDocumentation() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.LastDocUpdateAt = time.Now()
	a.Version++
}

// IncrementTaskCount increases the count of tasks associated with this asset
func (a *Asset) IncrementTaskCount() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.AssociatedTaskCount++
	a.UpdatedAt = time.Now()
	a.Version++
}

// DecrementTaskCount decreases the count of tasks associated with this asset
func (a *Asset) DecrementTaskCount() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.AssociatedTaskCount > 0 {
		a.AssociatedTaskCount--
		a.UpdatedAt = time.Now()
		a.Version++
	}
}

// generateID creates a unique ID for the asset
func generateID(name string) string {
	hash := sha256.New()
	hash.Write([]byte(name))
	hash.Write([]byte(time.Now().String()))
	return hex.EncodeToString(hash.Sum(nil))[:16]
}
