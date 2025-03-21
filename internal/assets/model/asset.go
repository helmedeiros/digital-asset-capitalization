package model

import (
	"crypto/sha256"
	"encoding/hex"
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
	mu sync.RWMutex
	// ID is a unique identifier for the asset
	ID string
	// Name is the display name of the asset
	Name string
	// Description provides detailed information about the asset
	Description string
	// CreatedAt is when the asset was first created
	CreatedAt time.Time
	// UpdatedAt is when the asset was last modified
	UpdatedAt time.Time
	// LastDocUpdateAt is when the asset's documentation was last updated
	LastDocUpdateAt time.Time
	// ContributionTypes tracks the types of work done on this asset
	ContributionTypes []string
	// AssociatedTaskCount tracks how many tasks are linked to this asset
	AssociatedTaskCount int
	// Version is used for optimistic locking
	Version int
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
