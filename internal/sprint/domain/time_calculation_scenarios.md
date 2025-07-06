# Multi-Sprint Story Time Allocation Scenarios

## Overview

This document outlines all possible scenarios for stories that span multiple sprints and how time allocation should be calculated within sprint boundaries.

## Core Principles

1. **Sprint Boundary Respect**: Only count time within the target sprint's date range
2. **Status Transition Accuracy**: Handle status changes that cross sprint boundaries
3. **Work Time Attribution**: Attribute actual work time to the correct sprint
4. **Proportional Allocation**: Distribute time fairly across sprints based on actual work periods

## Scenario Categories

### 1. Single Sprint Stories (Current Working Cases)

- **Scenario 1A**: Story starts and completes within one sprint

  - Expected: Full time allocation to the sprint
  - Current: ✅ Works correctly

- **Scenario 1B**: Story goes directly from "To Do" to "Done" within one sprint
  - Expected: Minimum 1 hour allocation
  - Current: ✅ Works correctly

### 2. Cross-Sprint Stories (Problem Cases)

#### **Scenario 2A**: Story starts in Sprint A, completes in Sprint B

```
Sprint A: 2024-03-01 to 2024-03-15
Sprint B: 2024-03-16 to 2024-03-30

Story Timeline:
- 2024-03-05: To Do → In Progress (within Sprint A)
- 2024-03-25: In Progress → Done (within Sprint B)

Current Problem:
- Sprint A allocation: 0 hours (story not completed)
- Sprint B allocation: 20 days (full lifecycle)

Expected Solution:
- Sprint A allocation: 10 days (Mar 5-15)
- Sprint B allocation: 10 days (Mar 16-25)
```

#### **Scenario 2B**: Story starts before Sprint A, completes in Sprint A

```
Sprint A: 2024-03-16 to 2024-03-30

Story Timeline:
- 2024-03-05: To Do → In Progress (before Sprint A)
- 2024-03-25: In Progress → Done (within Sprint A)

Current Problem:
- Sprint A allocation: 20 days (full lifecycle)

Expected Solution:
- Sprint A allocation: 10 days (Mar 16-25 only)
```

#### **Scenario 2C**: Story starts in Sprint A, completes after Sprint A

```
Sprint A: 2024-03-01 to 2024-03-15

Story Timeline:
- 2024-03-05: To Do → In Progress (within Sprint A)
- 2024-03-25: In Progress → Done (after Sprint A)

Current Problem:
- Sprint A allocation: 0 hours (story not completed)

Expected Solution:
- Sprint A allocation: 10 days (Mar 5-15 only)
```

### 3. Complex Multi-Sprint Stories

#### **Scenario 3A**: Story with multiple status transitions across sprints

```
Sprint A: 2024-03-01 to 2024-03-15
Sprint B: 2024-03-16 to 2024-03-30

Story Timeline:
- 2024-03-05: To Do → In Progress (Sprint A)
- 2024-03-10: In Progress → Blocked (Sprint A)
- 2024-03-20: Blocked → In Progress (Sprint B)
- 2024-03-25: In Progress → Done (Sprint B)

Expected Solution:
- Sprint A allocation: 5 days (Mar 5-10 only)
- Sprint B allocation: 5 days (Mar 20-25 only)
```

#### **Scenario 3B**: Story spans three sprints

```
Sprint A: 2024-03-01 to 2024-03-15
Sprint B: 2024-03-16 to 2024-03-30
Sprint C: 2024-04-01 to 2024-04-15

Story Timeline:
- 2024-03-05: To Do → In Progress (Sprint A)
- 2024-03-20: In Progress → Blocked (Sprint B)
- 2024-04-10: Blocked → In Progress (Sprint C)
- 2024-04-12: In Progress → Done (Sprint C)

Expected Solution:
- Sprint A allocation: 10 days (Mar 5-15)
- Sprint B allocation: 0 days (blocked entire period)
- Sprint C allocation: 2 days (Apr 10-12)
```

#### **Scenario 3C**: Story moves backward in status across sprints

```
Sprint A: 2024-03-01 to 2024-03-15
Sprint B: 2024-03-16 to 2024-03-30

Story Timeline:
- 2024-03-05: To Do → In Progress (Sprint A)
- 2024-03-10: In Progress → Under Review (Sprint A)
- 2024-03-20: Under Review → In Progress (Sprint B)
- 2024-03-25: In Progress → Done (Sprint B)

Expected Solution:
- Sprint A allocation: 10 days (Mar 5-15, both In Progress and Under Review count as work)
- Sprint B allocation: 10 days (Mar 16-25, both In Progress and Under Review count as work)
```

### 4. Edge Cases

#### **Scenario 4A**: Story completed same day it starts (cross-sprint)

```
Sprint A: 2024-03-01 to 2024-03-15
Sprint B: 2024-03-16 to 2024-03-30

Story Timeline:
- 2024-03-15 10:00: To Do → In Progress (Sprint A)
- 2024-03-16 14:00: In Progress → Done (Sprint B)

Expected Solution:
- Sprint A allocation: 14 hours (10:00-24:00 on Mar 15)
- Sprint B allocation: 14 hours (00:00-14:00 on Mar 16)
```

#### **Scenario 4B**: Story with no status changes but exists in sprint

```
Sprint A: 2024-03-01 to 2024-03-15

Story Timeline:
- Story exists in sprint but no status changes in changelog

Expected Solution:
- Sprint A allocation: 0 hours (no evidence of work)
```

## Technical Implementation Strategy

### 1. Time Calculation Abstraction

```go
type TimeCalculationStrategy interface {
    CalculateWorkingHours(ctx context.Context, issue JiraIssue, sprintBoundary SprintBoundary) (float64, error)
}
```

### 2. Sprint Boundary Value Object

```go
type SprintBoundary struct {
    StartDate time.Time
    EndDate   time.Time
}
```

### 3. Status Change Period

```go
type StatusChangePeriod struct {
    StartTime   time.Time
    EndTime     time.Time
    Status      string
    IsWorkTime  bool
}
```

### 4. Work Time Calculator

```go
type WorkTimeCalculator struct {
    strategy TimeCalculationStrategy
}
```

## Test Strategy

### Unit Tests

1. Test each scenario in isolation
2. Test boundary conditions (start/end of sprints)
3. Test edge cases (same day completion, no status changes)
4. Test time zone handling

### Integration Tests

1. Test with real sprint data
2. Test backward compatibility
3. Test performance with large datasets

## Implementation Phases

### Phase 1: Core Domain Logic

- Implement SprintBoundary value object
- Implement TimeCalculationStrategy interface
- Implement SprintBoundedTimeCalculator

### Phase 2: Integration

- Update SprintTimeAllocationUseCase
- Maintain backward compatibility
- Add configuration for strategy selection

### Phase 3: Testing & Validation

- Comprehensive test coverage
- Performance testing
- Real-world validation

## Success Criteria

1. **Accuracy**: Time allocation matches actual work periods within sprint boundaries
2. **Consistency**: Same story shows consistent time allocation across different sprint views
3. **Performance**: No significant performance degradation
4. **Maintainability**: Clean, testable code following SOLID principles
5. **Backward Compatibility**: Existing functionality remains unchanged when new feature is disabled
