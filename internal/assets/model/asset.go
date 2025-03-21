package model

import (
	"errors"
	"time"
)

var (
	ErrEmptyName        = errors.New("asset name cannot be empty")
	ErrEmptyDescription = errors.New("asset description cannot be empty")
)

type Asset struct {
	ID                  string
	Name                string
	Description         string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastDocUpdateAt     time.Time
	ContributionTypes   []string
	AssociatedTaskCount int
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
	}, nil
}

func (a *Asset) UpdateDescription(description string) error {
	if description == "" {
		return ErrEmptyDescription
	}
	a.Description = description
	a.UpdatedAt = time.Now()
	return nil
}

func (a *Asset) UpdateDocumentation() {
	a.LastDocUpdateAt = time.Now()
}

func (a *Asset) AddContributionType(contributionType string) {
	a.ContributionTypes = append(a.ContributionTypes, contributionType)
	a.UpdatedAt = time.Now()
}

func (a *Asset) IncrementTaskCount() {
	a.AssociatedTaskCount++
	a.UpdatedAt = time.Now()
}

func (a *Asset) DecrementTaskCount() {
	if a.AssociatedTaskCount > 0 {
		a.AssociatedTaskCount--
		a.UpdatedAt = time.Now()
	}
}

// generateID creates a unique ID for the asset based on its name
func generateID(name string) string {
	// TODO: Implement proper ID generation
	return name
}
