package usecase

import (
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// AllocationRecord holds the calculated values for a single issue
type AllocationRecord struct {
	IssueKey         string
	EngineeringHours *float64
	WorkStream       string
	TPDBusinessUnit  string
}

// PushDetail describes the outcome for a single field update
type PushDetail struct {
	IssueKey string
	Field    string
	OldValue string
	NewValue string
	Status   string // "updated", "skipped", "error"
	Reason   string
}

// PushResult aggregates all push details
type PushResult struct {
	Details      []PushDetail
	UpdatedCount int
	SkippedCount int
	ErrorCount   int
}

// PushAllocationUseCase pushes calculated allocation data back to JIRA custom fields
type PushAllocationUseCase struct {
	jiraPort ports.JiraPort
	dryRun   bool
}

// NewPushAllocationUseCase creates a new PushAllocationUseCase
func NewPushAllocationUseCase(jiraPort ports.JiraPort, dryRun bool) *PushAllocationUseCase {
	return &PushAllocationUseCase{
		jiraPort: jiraPort,
		dryRun:   dryRun,
	}
}

// Execute pushes allocation records to JIRA, only filling empty fields
func (uc *PushAllocationUseCase) Execute(records []AllocationRecord) (*PushResult, error) {
	result := &PushResult{}

	for _, rec := range records {
		current, err := uc.jiraPort.FetchCustomFields(rec.IssueKey)
		if err != nil {
			detail := PushDetail{
				IssueKey: rec.IssueKey,
				Field:    "all",
				Status:   "error",
				Reason:   fmt.Sprintf("failed to fetch current values: %v", err),
			}
			result.Details = append(result.Details, detail)
			result.ErrorCount++
			continue
		}

		update := ports.CustomFieldUpdate{}
		hasUpdates := false

		// Engineering Hours: only update if JIRA field is empty
		if rec.EngineeringHours != nil && current.EngineeringHours == nil {
			update.EngineeringHours = rec.EngineeringHours
			hasUpdates = true
			result.Details = append(result.Details, PushDetail{
				IssueKey: rec.IssueKey,
				Field:    "Engineering Hours",
				OldValue: "",
				NewValue: fmt.Sprintf("%.2f", *rec.EngineeringHours),
				Status:   uc.statusLabel("updated"),
			})
			result.UpdatedCount++
		} else if rec.EngineeringHours != nil && current.EngineeringHours != nil {
			result.Details = append(result.Details, PushDetail{
				IssueKey: rec.IssueKey,
				Field:    "Engineering Hours",
				OldValue: fmt.Sprintf("%.2f", *current.EngineeringHours),
				Status:   "skipped",
				Reason:   "already set",
			})
			result.SkippedCount++
		}

		// Work Stream: only update if JIRA field is empty
		if rec.WorkStream != "" && current.WorkStream == "" {
			ws := rec.WorkStream
			update.WorkStream = &ws
			hasUpdates = true
			result.Details = append(result.Details, PushDetail{
				IssueKey: rec.IssueKey,
				Field:    "Work Stream",
				OldValue: "",
				NewValue: rec.WorkStream,
				Status:   uc.statusLabel("updated"),
			})
			result.UpdatedCount++
		} else if rec.WorkStream != "" && current.WorkStream != "" {
			result.Details = append(result.Details, PushDetail{
				IssueKey: rec.IssueKey,
				Field:    "Work Stream",
				OldValue: current.WorkStream,
				Status:   "skipped",
				Reason:   "already set",
			})
			result.SkippedCount++
		}

		// TPD Business Unit: only update if JIRA field is empty
		if rec.TPDBusinessUnit != "" && len(current.TPDBusinessUnits) == 0 {
			buList := strings.Split(rec.TPDBusinessUnit, "; ")
			update.TPDBusinessUnits = buList
			hasUpdates = true
			result.Details = append(result.Details, PushDetail{
				IssueKey: rec.IssueKey,
				Field:    "TPD Business Unit",
				OldValue: "",
				NewValue: rec.TPDBusinessUnit,
				Status:   uc.statusLabel("updated"),
			})
			result.UpdatedCount++
		} else if rec.TPDBusinessUnit != "" && len(current.TPDBusinessUnits) > 0 {
			result.Details = append(result.Details, PushDetail{
				IssueKey: rec.IssueKey,
				Field:    "TPD Business Unit",
				OldValue: strings.Join(current.TPDBusinessUnits, "; "),
				Status:   "skipped",
				Reason:   "already set",
			})
			result.SkippedCount++
		}

		if !hasUpdates || uc.dryRun {
			continue
		}

		if err := uc.jiraPort.UpdateCustomFields(rec.IssueKey, update); err != nil {
			// Mark previously counted updates as errors
			for i := len(result.Details) - 1; i >= 0; i-- {
				d := &result.Details[i]
				if d.IssueKey == rec.IssueKey && d.Status == "updated" {
					d.Status = "error"
					d.Reason = fmt.Sprintf("update failed: %v", err)
					result.UpdatedCount--
					result.ErrorCount++
				}
			}
		}
	}

	return result, nil
}

func (uc *PushAllocationUseCase) statusLabel(real string) string {
	if uc.dryRun {
		return "will update"
	}
	return real
}
