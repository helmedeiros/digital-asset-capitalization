package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSpinner(t *testing.T) {
	tests := []struct {
		name     string
		style    SpinnerStyle
		message  string
		expected struct {
			frameCount int
			interval   time.Duration
		}
	}{
		{
			name:    "dots spinner",
			style:   SpinnerDots,
			message: "Loading",
			expected: struct {
				frameCount int
				interval   time.Duration
			}{10, 80 * time.Millisecond},
		},
		{
			name:    "line spinner",
			style:   SpinnerLine,
			message: "Processing",
			expected: struct {
				frameCount int
				interval   time.Duration
			}{4, 100 * time.Millisecond},
		},
		{
			name:    "bounce spinner",
			style:   SpinnerBounce,
			message: "Working",
			expected: struct {
				frameCount int
				interval   time.Duration
			}{4, 120 * time.Millisecond},
		},
		{
			name:    "circle spinner",
			style:   SpinnerCircle,
			message: "Loading",
			expected: struct {
				frameCount int
				interval   time.Duration
			}{4, 150 * time.Millisecond},
		},
		{
			name:    "arrow spinner",
			style:   SpinnerArrow,
			message: "Processing",
			expected: struct {
				frameCount int
				interval   time.Duration
			}{8, 125 * time.Millisecond},
		},
		{
			name:    "clock spinner",
			style:   SpinnerClock,
			message: "Waiting",
			expected: struct {
				frameCount int
				interval   time.Duration
			}{12, 200 * time.Millisecond},
		},
		{
			name:    "default spinner",
			style:   SpinnerStyle(999),
			message: "Unknown",
			expected: struct {
				frameCount int
				interval   time.Duration
			}{10, 80 * time.Millisecond},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spinner := NewSpinner(tt.style, tt.message)
			assert.NotNil(t, spinner)
			assert.Equal(t, tt.message, spinner.message)
			assert.Equal(t, ColorInfo, spinner.color)
			assert.Len(t, spinner.frames, tt.expected.frameCount)
			assert.Equal(t, tt.expected.interval, spinner.interval)
		})
	}
}

func TestSpinnerNext(t *testing.T) {
	spinner := NewSpinner(SpinnerLine, "Testing")

	// Test cycling through frames
	firstFrame := spinner.Next()
	assert.Contains(t, firstFrame, "Testing")

	// Advance through all frames
	frames := []string{firstFrame}
	for i := 1; i < len(spinner.frames); i++ {
		frames = append(frames, spinner.Next())
	}

	// Should cycle back to first frame
	nextCycle := spinner.Next()
	assert.Contains(t, nextCycle, "Testing")

	// Test without message
	spinner.message = ""
	frameOnly := spinner.Next()
	assert.NotContains(t, frameOnly, "Testing")
}

func TestSpinnerRender(t *testing.T) {
	spinner := NewSpinner(SpinnerDots, "Loading")
	result := spinner.Render()
	assert.Contains(t, result, "Loading")
}

func TestSpinnerSetters(t *testing.T) {
	spinner := NewSpinner(SpinnerDots, "Initial")

	// Test SetColor
	spinner.SetColor(ColorError)
	assert.Equal(t, ColorError, spinner.color)

	// Test SetMessage
	spinner.SetMessage("Updated")
	assert.Equal(t, "Updated", spinner.message)

	// Test GetInterval
	assert.Equal(t, 80*time.Millisecond, spinner.GetInterval())
}

func TestNewProgressBar(t *testing.T) {
	pb := NewProgressBar(100, 20)

	assert.NotNil(t, pb)
	assert.Equal(t, 100, pb.total)
	assert.Equal(t, 0, pb.current)
	assert.Equal(t, 20, pb.width)
	assert.Equal(t, "█", pb.fillChar)
	assert.Equal(t, "░", pb.emptyChar)
	assert.True(t, pb.brackets)
	assert.True(t, pb.showPercent)
}

func TestProgressBarUpdate(t *testing.T) {
	pb := NewProgressBar(100, 20)

	// Normal update
	pb.Update(50)
	assert.Equal(t, 50, pb.current)

	// Update beyond total
	pb.Update(150)
	assert.Equal(t, 100, pb.current)

	// Negative update
	pb.Update(-10)
	assert.Equal(t, 0, pb.current)
}

func TestProgressBarIncrement(t *testing.T) {
	pb := NewProgressBar(100, 20)

	pb.Increment()
	assert.Equal(t, 1, pb.current)

	pb.current = 99
	pb.Increment()
	assert.Equal(t, 100, pb.current)

	// Should not go beyond total
	pb.Increment()
	assert.Equal(t, 100, pb.current)
}

func TestProgressBarSetMessage(t *testing.T) {
	pb := NewProgressBar(100, 20)

	pb.SetMessage("Downloading")
	assert.Equal(t, "Downloading", pb.message)
}

func TestProgressBarRender(t *testing.T) {
	pb := NewProgressBar(100, 20)

	// Test with 0 progress
	result := pb.Render()
	assert.Contains(t, result, "[")
	assert.Contains(t, result, "]")
	assert.Contains(t, result, "0.0%")
	assert.Contains(t, result, "(0/100)")

	// Test with 50% progress
	pb.Update(50)
	result = pb.Render()
	assert.Contains(t, result, "50.0%")
	assert.Contains(t, result, "(50/100)")

	// Test with message
	pb.SetMessage("Processing")
	result = pb.Render()
	assert.Contains(t, result, "Processing:")

	// Test with invalid total
	pb.total = 0
	assert.Equal(t, "", pb.Render())
}

func TestProgressBarIsComplete(t *testing.T) {
	pb := NewProgressBar(100, 20)

	assert.False(t, pb.IsComplete())

	pb.Update(100)
	assert.True(t, pb.IsComplete())

	pb.Update(101)
	assert.True(t, pb.IsComplete())
}

func TestProgressBarGetPercentage(t *testing.T) {
	pb := NewProgressBar(100, 20)

	assert.Equal(t, 0.0, pb.GetPercentage())

	pb.Update(25)
	assert.Equal(t, 25.0, pb.GetPercentage())

	pb.Update(100)
	assert.Equal(t, 100.0, pb.GetPercentage())

	// Test with zero total
	pb.total = 0
	assert.Equal(t, 0.0, pb.GetPercentage())
}

func TestDefaultProgressStyle(t *testing.T) {
	style := DefaultProgressStyle()

	assert.Equal(t, ColorSuccess, style.FillColor)
	assert.Equal(t, ColorMuted, style.EmptyColor)
	assert.Equal(t, ColorOutput, style.TextColor)
	assert.True(t, style.ShowTitle)
	assert.False(t, style.ShowETA)
}

func TestNewLoadingIndicator(t *testing.T) {
	li := NewLoadingIndicator("Loading data")

	assert.NotNil(t, li)
	assert.Equal(t, "Loading data", li.message)
	assert.Equal(t, 0, li.dots)
	assert.Equal(t, 3, li.maxDots)
	assert.Equal(t, ColorInfo, li.color)
}

func TestLoadingIndicatorNext(t *testing.T) {
	li := NewLoadingIndicator("Loading")

	// Test cycling through dots
	results := []string{}
	for i := 0; i <= li.maxDots; i++ {
		results = append(results, li.Next())
	}

	// Should have different number of dots
	assert.Contains(t, results[0], "Loading")
	assert.Contains(t, results[1], ".")
	assert.Contains(t, results[2], "..")
	assert.Contains(t, results[3], "...")

	// Should cycle back
	assert.Equal(t, 0, li.dots)
}

func TestLoadingIndicatorRender(t *testing.T) {
	li := NewLoadingIndicator("Processing")
	result := li.Render()
	assert.Contains(t, result, "Processing")
}

func TestNewStatusIndicator(t *testing.T) {
	tests := []struct {
		status        string
		expectedIcon  string
		expectedColor Color
	}{
		{"success", "✅", ColorSuccess},
		{"error", "❌", ColorError},
		{"warning", "⚠️", ColorWarning},
		{"info", "ℹ️", ColorInfo},
		{"loading", "ACTIVE", ColorInfo},
		{"pending", "PAUSE", ColorWarning},
		{"custom", "", ColorOutput},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			si := NewStatusIndicator(tt.status)
			assert.Equal(t, tt.status, si.status)
			assert.Equal(t, tt.expectedIcon, si.icon)
			assert.Equal(t, tt.expectedColor, si.color)
		})
	}
}

func TestStatusIndicatorRender(t *testing.T) {
	si := NewStatusIndicator("success")
	result := si.Render()
	assert.Contains(t, result, "✅")
	assert.Contains(t, result, "success")
}

func TestStatusIndicatorSetCustom(t *testing.T) {
	si := NewStatusIndicator("custom")
	si.SetCustom("SYNC", ColorPrimary)

	assert.Equal(t, "SYNC", si.icon)
	assert.Equal(t, ColorPrimary, si.color)
}

func TestNewMultiProgressBar(t *testing.T) {
	mpb := NewMultiProgressBar(30)

	assert.NotNil(t, mpb)
	assert.Equal(t, 30, mpb.width)
	assert.Empty(t, mpb.bars)
}

func TestMultiProgressBarAddBar(t *testing.T) {
	mpb := NewMultiProgressBar(30)

	bar1 := mpb.AddBar("Task 1", 100)
	bar2 := mpb.AddBar("Task 2", 200)

	assert.NotNil(t, bar1)
	assert.NotNil(t, bar2)
	assert.Len(t, mpb.bars, 2)
	assert.Equal(t, "Task 1", mpb.bars[0].Name)
	assert.Equal(t, "Task 2", mpb.bars[1].Name)
}

func TestMultiProgressBarRender(t *testing.T) {
	mpb := NewMultiProgressBar(30)

	// Empty render
	assert.Equal(t, "", mpb.Render())

	// Add bars and render
	bar1 := mpb.AddBar("Download", 100)
	bar2 := mpb.AddBar("Process", 50)

	bar1.Update(50)
	bar2.Update(25)

	result := mpb.Render()
	assert.Contains(t, result, "Download:")
	assert.Contains(t, result, "Process:")
}

func TestMultiProgressBarIsAllComplete(t *testing.T) {
	mpb := NewMultiProgressBar(30)

	bar1 := mpb.AddBar("Task 1", 100)
	bar2 := mpb.AddBar("Task 2", 100)

	assert.False(t, mpb.IsAllComplete())

	bar1.Update(100)
	assert.False(t, mpb.IsAllComplete())

	bar2.Update(100)
	assert.True(t, mpb.IsAllComplete())
}

func TestMultiProgressBarGetOverallProgress(t *testing.T) {
	mpb := NewMultiProgressBar(30)

	// Empty bars
	assert.Equal(t, 0.0, mpb.GetOverallProgress())

	// Add bars
	bar1 := mpb.AddBar("Task 1", 100)
	bar2 := mpb.AddBar("Task 2", 100)

	assert.Equal(t, 0.0, mpb.GetOverallProgress())

	bar1.Update(50)
	bar2.Update(30)
	assert.Equal(t, 40.0, mpb.GetOverallProgress())

	bar1.Update(100)
	bar2.Update(100)
	assert.Equal(t, 100.0, mpb.GetOverallProgress())
}

func TestDefaultStepperStyle(t *testing.T) {
	style := DefaultStepperStyle()

	assert.Equal(t, "✅", style.CompletedIcon)
	assert.Equal(t, "ACTIVE", style.ActiveIcon)
	assert.Equal(t, "⭕", style.PendingIcon)
	assert.Equal(t, "❌", style.FailedIcon)
	assert.Equal(t, ColorSuccess, style.CompletedColor)
	assert.Equal(t, ColorInfo, style.ActiveColor)
	assert.Equal(t, ColorMuted, style.PendingColor)
	assert.Equal(t, ColorError, style.FailedColor)
}

func TestNewStepper(t *testing.T) {
	steps := []string{"Step 1", "Step 2", "Step 3"}
	stepper := NewStepper(steps)

	assert.NotNil(t, stepper)
	assert.Len(t, stepper.steps, 3)
	assert.Equal(t, 0, stepper.currentStep)

	for i, step := range stepper.steps {
		assert.Equal(t, steps[i], step.Name)
		assert.Equal(t, StepPending, step.Status)
	}
}

func TestStepperNextStep(t *testing.T) {
	stepper := NewStepper([]string{"Step 1", "Step 2", "Step 3"})

	// First step should become completed, second should be active
	stepper.NextStep()
	assert.Equal(t, StepCompleted, stepper.steps[0].Status)
	assert.Equal(t, StepActive, stepper.steps[1].Status)
	assert.Equal(t, 1, stepper.currentStep)

	// Move to last step
	stepper.NextStep()
	assert.Equal(t, StepCompleted, stepper.steps[1].Status)
	assert.Equal(t, StepActive, stepper.steps[2].Status)
	assert.Equal(t, 2, stepper.currentStep)

	// Complete last step
	stepper.NextStep()
	assert.Equal(t, StepCompleted, stepper.steps[2].Status)
	assert.Equal(t, 3, stepper.currentStep)

	// Try to go beyond
	stepper.NextStep()
	assert.Equal(t, 3, stepper.currentStep)
}

func TestStepperFailCurrentStep(t *testing.T) {
	stepper := NewStepper([]string{"Step 1", "Step 2"})

	stepper.NextStep() // Move to step 1
	stepper.FailCurrentStep()

	assert.Equal(t, StepFailed, stepper.steps[1].Status)
}

func TestStepperRender(t *testing.T) {
	stepper := NewStepper([]string{"Download", "Process", "Complete"})
	stepper.steps[0].Description = "Downloading files"

	result := stepper.Render()
	assert.Contains(t, result, "Download")
	assert.Contains(t, result, "Process")
	assert.Contains(t, result, "Complete")
	assert.Contains(t, result, "Downloading files")
}

func TestStepperIsComplete(t *testing.T) {
	stepper := NewStepper([]string{"Step 1", "Step 2"})

	assert.False(t, stepper.IsComplete())

	// Move to first step - completes step 0 and activates step 1
	stepper.NextStep()
	assert.False(t, stepper.IsComplete())

	// Move to second step - completes step 1, activates nothing (no more steps)
	stepper.NextStep()
	assert.True(t, stepper.IsComplete())
}

func TestStepperHasFailed(t *testing.T) {
	stepper := NewStepper([]string{"Step 1", "Step 2"})

	assert.False(t, stepper.HasFailed())

	stepper.NextStep()
	stepper.FailCurrentStep()

	assert.True(t, stepper.HasFailed())
}
