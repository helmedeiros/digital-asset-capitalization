package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// Domain-specific errors
var (
	ErrEmptyName         = errors.New("asset name cannot be empty")
	ErrEmptyDescription  = errors.New("asset description cannot be empty")
	ErrInvalidVersion    = errors.New("invalid version")
	ErrNegativeTaskCount = errors.New("task count cannot be negative")
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
	// Platform represents the domain/platform for classification hints
	Platform string `json:"platform"`
	// Status represents the current state of the asset
	Status string `json:"status"`
	// LaunchDate is when the asset was rolled out to production
	LaunchDate time.Time `json:"launch_date"`
	// IsRolledOut100 indicates if the asset is fully rolled out
	IsRolledOut100 bool `json:"is_rolled_out_100"`
	// Keywords are terms to match against task titles/descriptions
	Keywords []string `json:"keywords"`
	// DocLink is the link to full Confluence documentation
	DocLink string `json:"doc_link"`
	// Why explains the purpose and motivation for this asset
	Why string `json:"why"`
	// Benefits describes the economic benefits of this asset
	Benefits string `json:"benefits"`
	// How explains how the asset works
	How string `json:"how"`
	// Metrics defines how we measure success for this asset
	Metrics string `json:"metrics"`
	// DateStarted is when the asset development started
	DateStarted time.Time `json:"date_started"`
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (a *Asset) UnmarshalJSON(data []byte) error {
	type Alias Asset
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.mu = sync.RWMutex{}
	return nil
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

// NewAsset creates a new Asset instance
func NewAssetWithDetails(name, description, why, benefits, how, metrics string) (*Asset, error) {
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
		Why:                 why,
		Benefits:            benefits,
		How:                 how,
		Metrics:             metrics,
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
func (a *Asset) UpdateDocumentation() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.LastDocUpdateAt = time.Now()
	a.Version++
	return nil
}

// IncrementTaskCount increments the task count for this asset
func (a *Asset) IncrementTaskCount() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.AssociatedTaskCount++
	a.Version++
	return nil
}

// DecrementTaskCount decrements the task count for this asset
func (a *Asset) DecrementTaskCount() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.AssociatedTaskCount == 0 {
		return ErrNegativeTaskCount
	}
	a.AssociatedTaskCount--
	a.Version++
	return nil
}

// GetTaskCount returns the current task count
func (a *Asset) GetTaskCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.AssociatedTaskCount
}

// GetVersion returns the current version
func (a *Asset) GetVersion() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Version
}

// generateID creates a unique ID for an asset based on its name
func generateID(name string) string {
	hash := sha256.New()
	hash.Write([]byte(name))
	hash.Write([]byte(time.Now().String()))
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

// SetDateStarted sets the date when the asset development started
func (a *Asset) SetDateStarted(date time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.DateStarted = date
	a.UpdatedAt = time.Now()
	a.Version++
	return nil
}
