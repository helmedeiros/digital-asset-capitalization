package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// Common errors that can occur when working with assets
var (
	ErrEmptyName                 = errors.New("asset name cannot be empty")
	ErrEmptyDescription          = errors.New("asset description cannot be empty")
	ErrInvalidContributionType   = errors.New("invalid contribution type")
	ErrDuplicateContributionType = errors.New("contribution type already exists")
	ErrEmptyContributionType     = errors.New("contribution type cannot be empty")
)

// ValidContributionTypes defines the allowed contribution types for assets.
// These types represent different kinds of work that can be done on an asset:
// - discovery: Initial research and requirements gathering
// - development: Implementation of new features
// - maintenance: Bug fixes and improvements to existing features
var ValidContributionTypes = map[string]bool{
	"discovery":   true,
	"development": true,
	"maintenance": true,
}

// Asset represents a digital asset in the system.
// It tracks various aspects of the asset including its metadata,
// contribution types, and associated tasks.
type Asset struct {
	// mu protects all fields below
	mu sync.RWMutex `json:"-"`
	// ID is a unique identifier for the asset
	ID string `json:"id"`
	// Name is the display name of the asset
	Name string `json:"name"`
	// Description provides detailed information about the asset
	Description string `json:"description"`
	// Why provides additional context or justification for the asset
	Why string `json:"why"`
	// Benefits describes the advantages or positive outcomes of the asset
	Benefits string `json:"benefits"`
	// How describes the process or method used to create or implement the asset
	How string `json:"how"`
	// Metrics are key performance indicators or measurements for the asset
	Metrics string `json:"metrics"`
	// CreatedAt is when the asset was first created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when the asset was last modified
	UpdatedAt time.Time `json:"updated_at"`
	// LastDocUpdateAt is when the asset's documentation was last updated
	LastDocUpdateAt time.Time `json:"last_doc_update_at"`
	// ContributionTypes tracks the types of work done on this asset
	ContributionTypes []string `json:"contribution_types"`
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
}

// NewAsset creates a new Asset instance with the given name and description.
// It validates the input parameters and initializes all fields with appropriate default values.
// Returns an error if the name or description is empty.
func NewAsset(name, description string) (*Asset, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	if description == "" {
		return nil, ErrEmptyDescription
	}

	now := time.Now()
	return &Asset{
		ID:                generateID(name),
		Name:              name,
		Description:       description,
		CreatedAt:         now,
		UpdatedAt:         now,
		LastDocUpdateAt:   now,
		ContributionTypes: make([]string, 0),
		Version:           1,
	}, nil
}

// NewAsset creates a new Asset instance with the given name and description.
// It validates the input parameters and initializes all fields with appropriate default values.
// Returns an error if the name or description is empty.
func NewAssetWithDetails(name, description, why, benefits, how, metrics string) (*Asset, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	if description == "" {
		return nil, ErrEmptyDescription
	}

	now := time.Now()
	return &Asset{
		ID:                generateID(name),
		Name:              name,
		Description:       description,
		Why:               why,
		Benefits:          benefits,
		How:               how,
		Metrics:           metrics,
		CreatedAt:         now,
		UpdatedAt:         now,
		LastDocUpdateAt:   now,
		ContributionTypes: make([]string, 0),
		Version:           1,
	}, nil
}

// UpdateDescription updates the asset's description.
// Returns an error if the new description is empty.
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

// UpdateDocumentation marks the asset's documentation as updated.
// This should be called whenever the asset's documentation is modified.
func (a *Asset) UpdateDocumentation() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.LastDocUpdateAt = time.Now()
	a.Version++
}

// AddContributionType adds a new contribution type to the asset.
// Returns an error if:
// - The contribution type is empty
// - The contribution type is not in the list of valid types
// - The contribution type is already added to this asset
func (a *Asset) AddContributionType(contributionType string) error {
	if contributionType == "" {
		return ErrEmptyContributionType
	}
	if !ValidContributionTypes[contributionType] {
		return ErrInvalidContributionType
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, t := range a.ContributionTypes {
		if t == contributionType {
			return ErrDuplicateContributionType
		}
	}
	a.ContributionTypes = append(a.ContributionTypes, contributionType)
	a.UpdatedAt = time.Now()
	a.Version++
	return nil
}

// IncrementTaskCount increases the count of tasks associated with this asset.
// This should be called when a new task is linked to the asset.
func (a *Asset) IncrementTaskCount() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.AssociatedTaskCount++
	a.UpdatedAt = time.Now()
	a.Version++
}

// DecrementTaskCount decreases the count of tasks associated with this asset.
// This should be called when a task is unlinked from the asset.
// The count will not go below zero.
func (a *Asset) DecrementTaskCount() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.AssociatedTaskCount > 0 {
		a.AssociatedTaskCount--
		a.UpdatedAt = time.Now()
		a.Version++
	}
}

// generateID creates a unique ID for the asset based on its name and timestamp.
// The ID is a 16-character hexadecimal string generated using SHA-256.
func generateID(name string) string {
	hash := sha256.New()
	hash.Write([]byte(name))
	hash.Write([]byte(time.Now().String()))
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

// MarshalJSON implements the json.Marshaler interface.
// It ensures thread-safe marshaling of the Asset struct.
func (a *Asset) MarshalJSON() ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	type Alias Asset
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It ensures thread-safe unmarshaling of the Asset struct.
func (a *Asset) UnmarshalJSON(data []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	type Alias Asset
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return nil
}
