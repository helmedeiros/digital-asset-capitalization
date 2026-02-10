package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Domain-specific errors
var (
	ErrEmptyName         = errors.New("asset name cannot be empty")
	ErrEmptyDescription  = errors.New("asset description cannot be empty")
	ErrInvalidVersion    = errors.New("invalid version")
	ErrNegativeTaskCount = errors.New("task count cannot be negative")
	ErrInvalidAssetID    = errors.New("asset ID must follow cap-asset-* format")
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
	// OwningTeam is the primary team responsible for this asset
	OwningTeam string `json:"owning_team,omitempty"`
	// ContributingTeams are teams that contribute to this asset
	ContributingTeams []string `json:"contributing_teams,omitempty"`
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
		Status:              "Planning",
		CreatedAt:           now,
		UpdatedAt:           now,
		LastDocUpdateAt:     now,
		AssociatedTaskCount: 0,
		Version:             1,
	}, nil
}

// NewAssetWithDetails creates a new Asset instance with additional details
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
		Status:              "Planning",
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

// GetID returns the asset ID safely
func (a *Asset) GetID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.ID
}

// generateID creates a unique ID for an asset based on its name in cap-asset-* format
func generateID(name string) string {
	// Convert name to lowercase and replace spaces with hyphens
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	// Remove special characters that aren't suitable for labels
	id = strings.ReplaceAll(id, "(", "")
	id = strings.ReplaceAll(id, ")", "")
	id = strings.ReplaceAll(id, "&", "and")
	id = strings.ReplaceAll(id, "/", "-")
	id = strings.ReplaceAll(id, ".", "-")

	return fmt.Sprintf("cap-asset-%s", id)
}

// ValidateAssetID checks if an asset ID follows the cap-asset-* format
func ValidateAssetID(id string) error {
	if !strings.HasPrefix(id, "cap-asset-") {
		return ErrInvalidAssetID
	}
	return nil
}

// IsValidAssetID returns true if the ID follows the cap-asset-* format
func IsValidAssetID(id string) bool {
	return strings.HasPrefix(id, "cap-asset-") && len(id) > len("cap-asset-")
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

// SetOwningTeam sets the primary team responsible for this asset
func (a *Asset) SetOwningTeam(team string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.OwningTeam = strings.TrimSpace(team)
	a.UpdatedAt = time.Now()
	a.Version++
	return nil
}

// GetOwningTeam returns the owning team
func (a *Asset) GetOwningTeam() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.OwningTeam
}

// AddContributingTeam adds a team to the contributing teams list
func (a *Asset) AddContributingTeam(team string) error {
	trimmedTeam := strings.TrimSpace(team)
	if trimmedTeam == "" {
		return errors.New("team name cannot be empty")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if team is already in the list
	for _, existingTeam := range a.ContributingTeams {
		if existingTeam == trimmedTeam {
			return nil // Team already exists, no error but no change
		}
	}

	// Don't add the owning team to contributing teams
	if trimmedTeam == a.OwningTeam {
		return nil
	}

	a.ContributingTeams = append(a.ContributingTeams, trimmedTeam)
	a.UpdatedAt = time.Now()
	a.Version++
	return nil
}

// RemoveContributingTeam removes a team from the contributing teams list
func (a *Asset) RemoveContributingTeam(team string) error {
	trimmedTeam := strings.TrimSpace(team)
	if trimmedTeam == "" {
		return errors.New("team name cannot be empty")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Find and remove the team
	for i, existingTeam := range a.ContributingTeams {
		if existingTeam == trimmedTeam {
			a.ContributingTeams = append(a.ContributingTeams[:i], a.ContributingTeams[i+1:]...)
			a.UpdatedAt = time.Now()
			a.Version++
			return nil
		}
	}

	return fmt.Errorf("team '%s' not found in contributing teams", trimmedTeam)
}

// GetContributingTeams returns a copy of the contributing teams list
func (a *Asset) GetContributingTeams() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]string, len(a.ContributingTeams))
	copy(result, a.ContributingTeams)
	return result
}

// SetContributingTeams sets the entire contributing teams list
func (a *Asset) SetContributingTeams(teams []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Filter out empty teams and duplicates
	var cleanTeams []string
	teamSet := make(map[string]bool)

	for _, team := range teams {
		trimmedTeam := strings.TrimSpace(team)
		if trimmedTeam == "" {
			continue
		}
		// Don't add the owning team to contributing teams
		if trimmedTeam == a.OwningTeam {
			continue
		}
		if !teamSet[trimmedTeam] {
			cleanTeams = append(cleanTeams, trimmedTeam)
			teamSet[trimmedTeam] = true
		}
	}

	a.ContributingTeams = cleanTeams
	a.UpdatedAt = time.Now()
	a.Version++
	return nil
}

// IsTeamAssociated checks if a team is either the owning team or a contributing team
func (a *Asset) IsTeamAssociated(team string) bool {
	trimmedTeam := strings.TrimSpace(team)
	if trimmedTeam == "" {
		return false
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check if it's the owning team
	if a.OwningTeam == trimmedTeam {
		return true
	}

	// Check if it's in contributing teams
	for _, contributingTeam := range a.ContributingTeams {
		if contributingTeam == trimmedTeam {
			return true
		}
	}

	return false
}

// GetAllAssociatedTeams returns both owning and contributing teams
func (a *Asset) GetAllAssociatedTeams() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var allTeams []string
	if a.OwningTeam != "" {
		allTeams = append(allTeams, a.OwningTeam)
	}
	allTeams = append(allTeams, a.ContributingTeams...)
	return allTeams
}
