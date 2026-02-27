package usecase

import (
	"fmt"
	"strings"
	"unicode"

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

// Execute pushes allocation records to JIRA, only filling empty fields.
// Each field is pushed individually so one unsupported field does not block others.
func (uc *PushAllocationUseCase) Execute(records []AllocationRecord) (*PushResult, error) {
	result := &PushResult{}

	for _, rec := range records {
		current, err := uc.jiraPort.FetchCustomFields(rec.IssueKey)
		if err != nil {
			result.Details = append(result.Details, PushDetail{
				IssueKey: rec.IssueKey,
				Field:    "all",
				Status:   "error",
				Reason:   fmt.Sprintf("failed to fetch current values: %v", err),
			})
			result.ErrorCount++
			continue
		}

		// Engineering Hours
		if rec.EngineeringHours != nil && current.EngineeringHours == nil {
			uc.pushField(result, rec.IssueKey, "Engineering Hours",
				"", fmt.Sprintf("%.2f", *rec.EngineeringHours),
				ports.CustomFieldUpdate{EngineeringHours: rec.EngineeringHours})
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

		// Work Stream
		if rec.WorkStream != "" && current.WorkStream == "" {
			ws := titleCase(rec.WorkStream)
			uc.pushField(result, rec.IssueKey, "Work Stream",
				"", ws,
				ports.CustomFieldUpdate{WorkStream: &ws})
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

		// TPD Business Unit
		if rec.TPDBusinessUnit != "" && len(current.TPDBusinessUnits) == 0 {
			buList := strings.Split(rec.TPDBusinessUnit, "; ")
			uc.pushField(result, rec.IssueKey, "TPD Business Unit",
				"", rec.TPDBusinessUnit,
				ports.CustomFieldUpdate{TPDBusinessUnits: buList})
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
	}

	return result, nil
}

// pushField pushes a single field update and records the result
func (uc *PushAllocationUseCase) pushField(result *PushResult, issueKey, field, oldValue, newValue string, update ports.CustomFieldUpdate) {
	if uc.dryRun {
		result.Details = append(result.Details, PushDetail{
			IssueKey: issueKey,
			Field:    field,
			OldValue: oldValue,
			NewValue: newValue,
			Status:   "will update",
		})
		result.UpdatedCount++
		return
	}

	if err := uc.jiraPort.UpdateCustomFields(issueKey, update); err != nil {
		result.Details = append(result.Details, PushDetail{
			IssueKey: issueKey,
			Field:    field,
			OldValue: oldValue,
			NewValue: newValue,
			Status:   "error",
			Reason:   fmt.Sprintf("update failed: %v", err),
		})
		result.ErrorCount++
	} else {
		result.Details = append(result.Details, PushDetail{
			IssueKey: issueKey,
			Field:    field,
			OldValue: oldValue,
			NewValue: newValue,
			Status:   "updated",
		})
		result.UpdatedCount++
	}
}

// titleCase capitalises the first letter of a string (e.g. "product" -> "Product")
func titleCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
