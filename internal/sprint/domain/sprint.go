package domain

import (
	"time"
)

// Sprint represents a sprint in the system
type Sprint struct {
	ID        string
	Name      string
	Project   string
	Team      Team
	Status    SprintStatus
	StartDate string
	EndDate   string
}

// SprintStatus represents the current status of a sprint
type SprintStatus string

const (
	SprintStatusActive   SprintStatus = "active"
	SprintStatusComplete SprintStatus = "complete"
	SprintStatusFuture   SprintStatus = "future"
)

// IsActive checks if the sprint is currently active
func (s *Sprint) IsActive() bool {
	now := time.Now().Format("2006-01-02")
	return s.StartDate <= now && s.EndDate >= now
}

// IsCompleted checks if the sprint is completed
func (s *Sprint) IsCompleted() bool {
	return s.Status == SprintStatusComplete
}

// GetDuration returns the total duration of the sprint
func (s *Sprint) GetDuration() time.Duration {
	start, _ := time.Parse("2006-01-02", s.StartDate)
	end, _ := time.Parse("2006-01-02", s.EndDate)
	return end.Sub(start)
}

// GetRemainingTime returns the remaining time in the sprint
func (s *Sprint) GetRemainingTime() time.Duration {
	now := time.Now().Format("2006-01-02")
	if now > s.EndDate {
		return 0
	}
	end, _ := time.Parse("2006-01-02", s.EndDate)
	start, _ := time.Parse("2006-01-02", now)
	// Add 1 to include both start and end dates, but subtract 24 hours to account for timezone
	return end.Sub(start) - 24*time.Hour
}

// Team represents a group of team members
type Team struct {
	Team []string `json:"team"`
}

// IsTeamMember checks if a person is a member of the team
func (t *Team) IsTeamMember(person string) bool {
	for _, member := range t.Team {
		if member == person {
			return true
		}
	}
	return false
}

// TeamMap is a mapping of project keys to their respective teams
type TeamMap map[string]Team

// GetTeam returns a team for a given project key
func (tm TeamMap) GetTeam(projectKey string) (*Team, bool) {
	team, exists := tm[projectKey]
	if !exists {
		return nil, false
	}
	return &team, true
}
