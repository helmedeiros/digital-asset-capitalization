package infrastructure

import (
	"context"
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain/ports"
)

// TimeAllocationAdapter adapts sprint allocation data from AssetCap CLI to investment calculator
type TimeAllocationAdapter struct{}

// NewTimeAllocationAdapter creates a new time allocation adapter
func NewTimeAllocationAdapter() *TimeAllocationAdapter {
	return &TimeAllocationAdapter{}
}

// GetSprintAllocations gets engineer time allocations for a sprint
func (ta *TimeAllocationAdapter) GetSprintAllocations(ctx context.Context, project, sprint string) ([]ports.EngineerAllocation, error) {
	// Execute the assetcap sprint allocate command
	cmd := exec.CommandContext(ctx, "./assetcap", "sprint", "allocate", "--project", project, "--sprint", sprint)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute sprint allocate command: %w", err)
	}

	return ta.parseSprintAllocationOutput(string(output), sprint)
}

// GetAssetAllocations gets engineer time allocations for an asset across sprints
func (ta *TimeAllocationAdapter) GetAssetAllocations(_ context.Context, _ string) ([]ports.EngineerAllocation, error) {
	// For now, we'll need to search through all available sprint data
	// This could be optimized by having the main app provide this data directly
	return nil, fmt.Errorf("asset-specific allocations not yet implemented - use GetSprintAllocations for each sprint")
}

// GetTaskAllocations gets engineer time allocations for specific tasks
func (ta *TimeAllocationAdapter) GetTaskAllocations(_ context.Context, _ []string) ([]ports.EngineerAllocation, error) {
	// This would require parsing task-specific data from the sprint allocations
	return nil, fmt.Errorf("task-specific allocations not yet implemented")
}

// parseSprintAllocationOutput parses the CSV output from sprint allocate command
func (ta *TimeAllocationAdapter) parseSprintAllocationOutput(output, sprint string) ([]ports.EngineerAllocation, error) {
	reader := csv.NewReader(strings.NewReader(output))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV output: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("invalid CSV output: expected header and at least one data row")
	}

	// Parse header to find engineer columns
	header := records[0]
	engineerStartIndex := ta.findEngineerColumnsStart(header)

	if engineerStartIndex == -1 {
		return nil, fmt.Errorf("could not find engineer columns in CSV header")
	}

	var allocations []ports.EngineerAllocation

	// Parse data rows
	for i, record := range records[1:] {
		if len(record) < engineerStartIndex {
			continue // Skip incomplete rows
		}

		// Extract task information
		taskKey := ta.getFieldValue(record, 1, "")
		taskTitle := ta.getFieldValue(record, 3, "")
		workType := ta.getFieldValue(record, 4, "")
		assetName := ta.getFieldValue(record, 5, "")
		startDateStr := ta.getFieldValue(record, 7, "")
		endDateStr := ta.getFieldValue(record, 8, "")

		// Parse dates
		startDate, _ := time.Parse("2006-01-02", startDateStr)
		endDate, _ := time.Parse("2006-01-02", endDateStr)

		// Extract engineer allocations
		for j := engineerStartIndex; j < len(record) && j < len(header); j++ {
			engineerName := header[j]
			allocationStr := strings.TrimSpace(record[j])

			if allocationStr == "" || allocationStr == "0.00%" {
				continue // Skip zero allocations
			}

			// Parse allocation percentage
			allocationStr = strings.TrimSuffix(allocationStr, "%")
			allocation, err := strconv.ParseFloat(allocationStr, 64)
			if err != nil {
				fmt.Printf("Warning: could not parse allocation '%s' for engineer %s in row %d\n", allocationStr, engineerName, i+2)
				continue
			}

			allocations = append(allocations, ports.EngineerAllocation{
				EngineerName: engineerName,
				TaskKey:      taskKey,
				TaskTitle:    taskTitle,
				WorkType:     workType,
				AssetName:    assetName,
				Sprint:       sprint,
				Allocation:   allocation,
				StartDate:    startDate,
				EndDate:      endDate,
			})
		}
	}

	return allocations, nil
}

// findEngineerColumnsStart finds the index where engineer columns start in the CSV header
func (ta *TimeAllocationAdapter) findEngineerColumnsStart(header []string) int {
	// Look for common engineer names or the pattern after standard columns
	standardColumns := []string{"sprint", "issueKey", "issueType", "issueTitle", "workType", "assetName", "status", "dateStarted", "dateCompleted"}

	// If we have the expected standard columns, engineers start after them
	if len(header) > len(standardColumns) {
		// Verify we have the standard columns
		allMatch := true
		for i, expected := range standardColumns {
			if i >= len(header) || header[i] != expected {
				allMatch = false
				break
			}
		}
		if allMatch {
			return len(standardColumns)
		}
	}

	// Fallback: look for names that look like engineer names
	for i, col := range header {
		if ta.looksLikeEngineerName(col) {
			return i
		}
	}

	return -1
}

// looksLikeEngineerName determines if a column name looks like an engineer name
func (ta *TimeAllocationAdapter) looksLikeEngineerName(name string) bool {
	// Simple heuristic: contains space and starts with capital letter
	parts := strings.Split(name, " ")
	return len(parts) >= 2 && len(name) > 3 && name[0] >= 'A' && name[0] <= 'Z'
}

// getFieldValue safely gets a field value from a record
func (ta *TimeAllocationAdapter) getFieldValue(record []string, index int, _ string) string {
	if index < len(record) {
		return strings.TrimSpace(record[index])
	}
	return ""
}
