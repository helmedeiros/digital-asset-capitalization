package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

var (
	ErrEmptyName                 = errors.New("asset name cannot be empty")
	ErrEmptyDescription          = errors.New("asset description cannot be empty")
	ErrInvalidContributionType   = errors.New("invalid contribution type")
	ErrDuplicateContributionType = errors.New("contribution type already exists")
	ErrEmptyContributionType     = errors.New("contribution type cannot be empty")
)

// ValidContributionTypes defines the allowed contribution types
var ValidContributionTypes = map[string]bool{
	"discovery":   true,
	"development": true,
	"maintenance": true,
}

type Asset struct {
	ID                  string
	Name                string
	Description         string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastDocUpdateAt     time.Time
	ContributionTypes   []string
	AssociatedTaskCount int
	Version             int
}

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

func (a *Asset) UpdateDescription(description string) error {
	if description == "" {
		return ErrEmptyDescription
	}
	a.Description = description
	a.UpdatedAt = time.Now()
	a.Version++
	return nil
}

func (a *Asset) UpdateDocumentation() {
	a.LastDocUpdateAt = time.Now()
	a.Version++
}

func (a *Asset) AddContributionType(contributionType string) error {
	if contributionType == "" {
		return ErrEmptyContributionType
	}
	if !ValidContributionTypes[contributionType] {
		return ErrInvalidContributionType
	}
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

func (a *Asset) IncrementTaskCount() {
	a.AssociatedTaskCount++
	a.UpdatedAt = time.Now()
	a.Version++
}

func (a *Asset) DecrementTaskCount() {
	if a.AssociatedTaskCount > 0 {
		a.AssociatedTaskCount--
		a.UpdatedAt = time.Now()
		a.Version++
	}
}

// generateID creates a unique ID for the asset based on its name and timestamp
func generateID(name string) string {
	hash := sha256.New()
	hash.Write([]byte(name))
	hash.Write([]byte(time.Now().String()))
	return hex.EncodeToString(hash.Sum(nil))[:16]
}
