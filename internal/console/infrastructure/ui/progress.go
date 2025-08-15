package ui

import (
	"fmt"
	"strings"
	"time"
)

// Progress status constants to avoid goconst warnings
const (
	ProgressStatusDone = "done"
)

// Spinner represents an animated spinner
type Spinner struct {
	frames   []string
	current  int
	interval time.Duration
	message  string
	color    Color
}

// SpinnerStyle defines different spinner styles
type SpinnerStyle int

const (
	SpinnerDots SpinnerStyle = iota
	SpinnerLine
	SpinnerBounce
	SpinnerCircle
	SpinnerArrow
	SpinnerClock
)

// NewSpinner creates a new spinner with the specified style
func NewSpinner(style SpinnerStyle, message string) *Spinner {
	s := &Spinner{
		message:  message,
		color:    ColorInfo,
		interval: 100 * time.Millisecond,
	}

	switch style {
	case SpinnerDots:
		s.frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		s.interval = 80 * time.Millisecond
	case SpinnerLine:
		s.frames = []string{"|", "/", "-", "\\"}
		s.interval = 100 * time.Millisecond
	case SpinnerBounce:
		s.frames = []string{"⠁", "⠂", "⠄", "⠂"}
		s.interval = 120 * time.Millisecond
	case SpinnerCircle:
		s.frames = []string{"◐", "◓", "◑", "◒"}
		s.interval = 150 * time.Millisecond
	case SpinnerArrow:
		s.frames = []string{"↑", "↗", "→", "↘", "↓", "↙", "←", "↖"}
		s.interval = 125 * time.Millisecond
	case SpinnerClock:
		s.frames = []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}
		s.interval = 200 * time.Millisecond
	default:
		s.frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		s.interval = 80 * time.Millisecond
	}

	return s
}

// Next returns the next frame of the spinner
func (s *Spinner) Next() string {
	frame := s.frames[s.current]
	s.current = (s.current + 1) % len(s.frames)

	if s.message != "" {
		return fmt.Sprintf("%s %s", Colorize(frame, s.color), s.message)
	}

	return Colorize(frame, s.color)
}

// Render renders the current frame
func (s *Spinner) Render() string {
	return s.Next()
}

// SetColor sets the spinner color
func (s *Spinner) SetColor(color Color) {
	s.color = color
}

// SetMessage sets the spinner message
func (s *Spinner) SetMessage(message string) {
	s.message = message
}

// GetInterval returns the spinner animation interval
func (s *Spinner) GetInterval() time.Duration {
	return s.interval
}

// ProgressBar represents a progress bar
type ProgressBar struct {
	total       int
	current     int
	width       int
	fillChar    string
	emptyChar   string
	brackets    bool
	showPercent bool
	message     string
	style       ProgressStyle
}

// ProgressStyle defines the appearance of progress bars
type ProgressStyle struct {
	FillColor  Color
	EmptyColor Color
	TextColor  Color
	ShowTitle  bool
	ShowETA    bool
}

// DefaultProgressStyle returns a default progress bar style
func DefaultProgressStyle() ProgressStyle {
	return ProgressStyle{
		FillColor:  ColorSuccess,
		EmptyColor: ColorMuted,
		TextColor:  ColorOutput,
		ShowTitle:  true,
		ShowETA:    false,
	}
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, width int) *ProgressBar {
	return &ProgressBar{
		total:       total,
		current:     0,
		width:       width,
		fillChar:    "█",
		emptyChar:   "░",
		brackets:    true,
		showPercent: true,
		style:       DefaultProgressStyle(),
	}
}

// Update updates the progress bar current value
func (pb *ProgressBar) Update(current int) {
	if current > pb.total {
		current = pb.total
	}
	if current < 0 {
		current = 0
	}
	pb.current = current
}

// Increment increments the progress bar by 1
func (pb *ProgressBar) Increment() {
	pb.Update(pb.current + 1)
}

// SetMessage sets the progress bar message
func (pb *ProgressBar) SetMessage(message string) {
	pb.message = message
}

// Render renders the progress bar
func (pb *ProgressBar) Render() string {
	if pb.total <= 0 {
		return ""
	}

	// Calculate percentage
	percentage := float64(pb.current) / float64(pb.total) * 100

	// Calculate filled portion
	filled := int(float64(pb.width) * float64(pb.current) / float64(pb.total))
	if filled > pb.width {
		filled = pb.width
	}

	// Build the bar
	var bar strings.Builder

	if pb.brackets {
		bar.WriteString(Colorize("[", pb.style.TextColor))
	}

	// Filled portion
	bar.WriteString(Colorize(strings.Repeat(pb.fillChar, filled), pb.style.FillColor))

	// Empty portion
	bar.WriteString(Colorize(strings.Repeat(pb.emptyChar, pb.width-filled), pb.style.EmptyColor))

	if pb.brackets {
		bar.WriteString(Colorize("]", pb.style.TextColor))
	}

	// Add percentage if enabled
	result := bar.String()
	if pb.showPercent {
		percentText := fmt.Sprintf(" %6.1f%%", percentage)
		result += Colorize(percentText, pb.style.TextColor)
	}

	// Add current/total
	countText := fmt.Sprintf(" (%d/%d)", pb.current, pb.total)
	result += Colorize(countText, pb.style.TextColor)

	// Add message if present
	if pb.message != "" {
		result = Colorize(pb.message+": ", pb.style.TextColor) + result
	}

	return result
}

// IsComplete returns true if the progress bar is complete
func (pb *ProgressBar) IsComplete() bool {
	return pb.current >= pb.total
}

// GetPercentage returns the current percentage
func (pb *ProgressBar) GetPercentage() float64 {
	if pb.total <= 0 {
		return 0
	}
	return float64(pb.current) / float64(pb.total) * 100
}

// LoadingIndicator represents a simple loading indicator
type LoadingIndicator struct {
	message string
	dots    int
	maxDots int
	color   Color
}

// NewLoadingIndicator creates a new loading indicator
func NewLoadingIndicator(message string) *LoadingIndicator {
	return &LoadingIndicator{
		message: message,
		dots:    0,
		maxDots: 3,
		color:   ColorInfo,
	}
}

// Next returns the next state of the loading indicator
func (li *LoadingIndicator) Next() string {
	dots := strings.Repeat(".", li.dots)
	spaces := strings.Repeat(" ", li.maxDots-li.dots)

	li.dots = (li.dots + 1) % (li.maxDots + 1)

	return fmt.Sprintf("%s%s%s", Colorize(li.message, li.color), dots, spaces)
}

// Render renders the current state
func (li *LoadingIndicator) Render() string {
	return li.Next()
}

// MultiProgressBar manages multiple progress bars
type MultiProgressBar struct {
	bars  []*NamedProgressBar
	width int
	style ProgressStyle
}

// NamedProgressBar represents a progress bar with a name
type NamedProgressBar struct {
	Name string
	Bar  *ProgressBar
}

// NewMultiProgressBar creates a new multi-progress bar
func NewMultiProgressBar(width int) *MultiProgressBar {
	return &MultiProgressBar{
		bars:  make([]*NamedProgressBar, 0),
		width: width,
		style: DefaultProgressStyle(),
	}
}

// AddBar adds a new progress bar
func (mpb *MultiProgressBar) AddBar(name string, total int) *ProgressBar {
	bar := NewProgressBar(total, mpb.width)
	bar.style = mpb.style

	namedBar := &NamedProgressBar{
		Name: name,
		Bar:  bar,
	}

	mpb.bars = append(mpb.bars, namedBar)
	return bar
}

// Render renders all progress bars
func (mpb *MultiProgressBar) Render() string {
	if len(mpb.bars) == 0 {
		return ""
	}

	var result strings.Builder

	for i, namedBar := range mpb.bars {
		// Add bar name
		result.WriteString(Colorize(namedBar.Name+":", mpb.style.TextColor))
		result.WriteString("\n")

		// Add the progress bar
		result.WriteString("  " + namedBar.Bar.Render())

		// Add newline except for last bar
		if i < len(mpb.bars)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// IsAllComplete returns true if all progress bars are complete
func (mpb *MultiProgressBar) IsAllComplete() bool {
	for _, bar := range mpb.bars {
		if !bar.Bar.IsComplete() {
			return false
		}
	}
	return true
}

// GetOverallProgress returns the overall progress percentage
func (mpb *MultiProgressBar) GetOverallProgress() float64 {
	if len(mpb.bars) == 0 {
		return 0
	}

	totalProgress := 0.0
	for _, bar := range mpb.bars {
		totalProgress += bar.Bar.GetPercentage()
	}

	return totalProgress / float64(len(mpb.bars))
}

// StatusIndicator represents a status with icon and color
type StatusIndicator struct {
	status string
	color  Color
	icon   string
}

// NewStatusIndicator creates a new status indicator
func NewStatusIndicator(status string) *StatusIndicator {
	si := &StatusIndicator{status: status}
	si.setDefaults()
	return si
}

// setDefaults sets default icon and color based on status
func (si *StatusIndicator) setDefaults() {
	switch strings.ToLower(si.status) {
	case "success", "completed", ProgressStatusDone, "finished":
		si.color = ColorSuccess
		si.icon = "✅"
	case "error", "failed", "failure":
		si.color = ColorError
		si.icon = "❌"
	case "warning", "warn":
		si.color = ColorWarning
		si.icon = "⚠️"
	case "info", "information":
		si.color = ColorInfo
		si.icon = "ℹ️"
	case "loading", "processing", "running":
		si.color = ColorInfo
		si.icon = "ACTIVE"
	case "pending", "waiting":
		si.color = ColorWarning
		si.icon = "PAUSE"
	default:
		si.color = ColorOutput
		si.icon = ""
	}
}

// Render renders the status indicator
func (si *StatusIndicator) Render() string {
	return fmt.Sprintf("%s %s", si.icon, Colorize(si.status, si.color))
}

// SetCustom sets custom icon and color
func (si *StatusIndicator) SetCustom(icon string, color Color) {
	si.icon = icon
	si.color = color
}

// Stepper represents a step-by-step process indicator
type Stepper struct {
	steps       []Step
	currentStep int
	style       StepperStyle
}

// Step represents a single step
type Step struct {
	Name        string
	Description string
	Status      StepStatus
}

// StepStatus represents the status of a step
type StepStatus int

const (
	StepPending StepStatus = iota
	StepActive
	StepCompleted
	StepFailed
)

// StepperStyle defines the appearance of steppers
type StepperStyle struct {
	CompletedIcon  string
	ActiveIcon     string
	PendingIcon    string
	FailedIcon     string
	CompletedColor Color
	ActiveColor    Color
	PendingColor   Color
	FailedColor    Color
}

// DefaultStepperStyle returns a default stepper style
func DefaultStepperStyle() StepperStyle {
	return StepperStyle{
		CompletedIcon:  "✅",
		ActiveIcon:     "ACTIVE",
		PendingIcon:    "⭕",
		FailedIcon:     "❌",
		CompletedColor: ColorSuccess,
		ActiveColor:    ColorInfo,
		PendingColor:   ColorMuted,
		FailedColor:    ColorError,
	}
}

// NewStepper creates a new stepper
func NewStepper(stepNames []string) *Stepper {
	steps := make([]Step, len(stepNames))
	for i, name := range stepNames {
		steps[i] = Step{
			Name:   name,
			Status: StepPending,
		}
	}

	return &Stepper{
		steps:       steps,
		currentStep: 0,
		style:       DefaultStepperStyle(),
	}
}

// NextStep advances to the next step
func (s *Stepper) NextStep() {
	if s.currentStep < len(s.steps) {
		s.steps[s.currentStep].Status = StepCompleted
		s.currentStep++
		if s.currentStep < len(s.steps) {
			s.steps[s.currentStep].Status = StepActive
		}
	}
}

// FailCurrentStep marks the current step as failed
func (s *Stepper) FailCurrentStep() {
	if s.currentStep < len(s.steps) {
		s.steps[s.currentStep].Status = StepFailed
	}
}

// Render renders the stepper
func (s *Stepper) Render() string {
	var result strings.Builder

	for i, step := range s.steps {
		var icon string
		var color Color

		switch step.Status {
		case StepCompleted:
			icon = s.style.CompletedIcon
			color = s.style.CompletedColor
		case StepActive:
			icon = s.style.ActiveIcon
			color = s.style.ActiveColor
		case StepFailed:
			icon = s.style.FailedIcon
			color = s.style.FailedColor
		default: // StepPending
			icon = s.style.PendingIcon
			color = s.style.PendingColor
		}

		result.WriteString(fmt.Sprintf("%s %s", icon, Colorize(step.Name, color)))

		if step.Description != "" {
			result.WriteString(fmt.Sprintf(" - %s", MutedText(step.Description)))
		}

		if i < len(s.steps)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// IsComplete returns true if all steps are completed
func (s *Stepper) IsComplete() bool {
	for _, step := range s.steps {
		if step.Status != StepCompleted {
			return false
		}
	}
	return true
}

// HasFailed returns true if any step has failed
func (s *Stepper) HasFailed() bool {
	for _, step := range s.steps {
		if step.Status == StepFailed {
			return true
		}
	}
	return false
}
