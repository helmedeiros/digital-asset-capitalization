package prompt

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// Handler manages the interactive console prompt
type Handler struct {
	consoleService *application.ConsoleService
	reader         *bufio.Reader
	sessionContext *domain.Context
	promptStyle    Style
}

// Style defines the appearance of the console prompt
type Style struct {
	Prompt        string
	WelcomeMsg    string
	GoodbyeMsg    string
	ErrorPrefix   string
	SuccessPrefix string
	InfoPrefix    string
}

// DefaultStyle returns a default prompt style
func DefaultStyle() Style {
	return Style{
		Prompt:        "> ",
		WelcomeMsg:    "AssetCap AI Console\n\nAsk me anything about your digital assets, tasks, or investments.\nType 'help' for guidance or 'exit' to quit.",
		GoodbyeMsg:    "\nThanks for using AssetCap AI Console.",
		ErrorPrefix:   "  ❌ ",
		SuccessPrefix: "  ✅ ",
		InfoPrefix:    "  ℹ️  ",
	}
}

// NewHandler creates a new prompt handler
func NewHandler(consoleService *application.ConsoleService) *Handler {
	return &Handler{
		consoleService: consoleService,
		reader:         bufio.NewReader(os.Stdin),
		promptStyle:    DefaultStyle(),
	}
}

// Start begins the interactive console session
func (h *Handler) Start(ctx context.Context) error {
	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal...")
		cancel()
	}()

	// Start console session
	sessionContext, err := h.consoleService.StartSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to start console session: %w", err)
	}
	h.sessionContext = sessionContext

	// Display welcome message
	h.displayWelcome()

	// Main input loop
	for {
		select {
		case <-ctx.Done():
			h.displayGoodbye()
			return h.endSession(ctx)
		default:
			if err := h.processInput(ctx); err != nil {
				if err == ErrExitRequested {
					h.displayGoodbye()
					return h.endSession(ctx)
				}
				h.displayError(fmt.Sprintf("Error processing input: %v", err))
			}
		}
	}
}

// processInput handles a single input cycle
func (h *Handler) processInput(ctx context.Context) error {
	// Display input area with proper separation
	h.displayCompleteInputBox()

	// Read input with timeout
	inputChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		input, err := h.reader.ReadString('\n')
		if err != nil {
			errorChan <- err
			return
		}
		inputChan <- strings.TrimSpace(input)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errorChan:
		return err
	case input := <-inputChan:
		h.completeInputBoxWithInput(input)
		return h.handleInput(ctx, input)
	case <-time.After(10 * time.Minute):
		h.displayInfo("Session timeout due to inactivity (10 minutes)")
		return ErrExitRequested
	}
}

// handleInput processes user input
func (h *Handler) handleInput(ctx context.Context, input string) error {
	if input == "" {
		return nil
	}

	// Process the input through the console service
	result, err := h.consoleService.ProcessInput(ctx, h.sessionContext.SessionID, input)
	if err != nil {
		h.displayError(err.Error())
		return nil
	}

	// Handle the result
	if result.RequiresClarification {
		h.displayClarification(result.ClarificationPrompt, result.Options)
		return nil
	}

	if !result.Success {
		h.displayError(result.Error)
		if result.Help != "" {
			h.displayInfo(result.Help)
		}
		return nil
	}

	// Check for exit command
	if result.Command != nil && result.Command.Interpreted == "exit" {
		return ErrExitRequested
	}

	// Check for context commands
	if result.Command != nil && strings.HasPrefix(result.Command.Interpreted, "context") {
		return h.handleContextCommand(ctx, result)
	}

	// Display successful result
	h.displayResult(result)

	return nil
}

// handleContextCommand handles context-specific commands
func (h *Handler) handleContextCommand(ctx context.Context, result *application.ProcessResult) error {
	if result.Command == nil {
		return nil
	}

	parts := strings.Fields(result.Command.Interpreted)
	if len(parts) < 2 {
		return nil
	}

	action := parts[1]

	switch action {
	case "show":
		// Get updated context
		sessionContext, err := h.consoleService.GetSessionContext(ctx, h.sessionContext.SessionID)
		if err != nil {
			h.displayError(fmt.Sprintf("Failed to get context: %v", err))
			return nil
		}
		h.displayContext(sessionContext)
	case "clear":
		h.displaySuccess("Context cleared")
		// Refresh our local context reference
		sessionContext, err := h.consoleService.GetSessionContext(ctx, h.sessionContext.SessionID)
		if err == nil {
			h.sessionContext = sessionContext
		}
	}

	return nil
}

// displayCompleteInputBox shows a complete, visible input box
func (h *Handler) displayCompleteInputBox() {
	// Create a clear separation from output
	fmt.Println()
	fmt.Println(strings.Repeat("═", 80)) // Double line separator
	fmt.Println()

	// Show a complete input box that's always visible
	fmt.Println("INPUT:")
	fmt.Println("┌" + strings.Repeat("─", 78) + "┐")

	// Show the input line with prompt - but don't close it yet
	fmt.Print("│ " + h.promptStyle.Prompt)
}

// completeInputBoxWithInput completes the input box after user types inside it
func (h *Handler) completeInputBoxWithInput(input string) {
	// Calculate layout for the input that was typed in the box
	promptLen := len(h.promptStyle.Prompt) + 2 // "│ " + prompt
	availableSpace := 78 - promptLen - 1       // -1 for closing "│"

	if len(input) <= availableSpace {
		// Single line - pad the rest of the line and close
		padding := availableSpace - len(input)
		fmt.Print(strings.Repeat(" ", padding) + " │")
		fmt.Println()
	} else {
		// Multi-line - close current line and show overflow
		fmt.Print(" │")
		fmt.Println()

		// Show remaining text on additional lines
		remaining := input[availableSpace:]
		lineWidth := 76 // 78 - 2 (for borders)

		for len(remaining) > 0 {
			if len(remaining) <= lineWidth {
				// Last line
				fmt.Print("│ " + remaining + strings.Repeat(" ", lineWidth-len(remaining)) + " │")
				fmt.Println()
				remaining = ""
			} else {
				// Full line
				fmt.Print("│ " + remaining[:lineWidth] + " │")
				fmt.Println()
				remaining = remaining[lineWidth:]
			}
		}
	}

	// Close the box
	fmt.Println("└" + strings.Repeat("─", 78) + "┘")
	fmt.Println() // Space after input box
}

// displayWelcome shows the welcome message
func (h *Handler) displayWelcome() {
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(h.promptStyle.WelcomeMsg)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println()
}

// displayGoodbye shows the goodbye message
func (h *Handler) displayGoodbye() {
	fmt.Println()
	fmt.Println(h.promptStyle.GoodbyeMsg)
}

// displayError shows an error message
func (h *Handler) displayError(message string) {
	fmt.Printf("%s%s\n", h.promptStyle.ErrorPrefix, message)
}

// displaySuccess shows a success message
func (h *Handler) displaySuccess(message string) {
	fmt.Printf("%s%s\n", h.promptStyle.SuccessPrefix, message)
}

// displayInfo shows an info message
func (h *Handler) displayInfo(message string) {
	fmt.Printf("%s%s\n", h.promptStyle.InfoPrefix, message)
}

// displayClarification shows a clarification prompt
func (h *Handler) displayClarification(prompt string, options []string) {
	fmt.Println()
	fmt.Println("CLARIFICATION:")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("🤔 %s\n", prompt)
	if len(options) > 0 {
		fmt.Println("\nOptions:")
		for i, option := range options {
			fmt.Printf("  %d. %s\n", i+1, option)
		}
	}
	fmt.Println(strings.Repeat("-", 40))
}

// displayResult shows command execution results
func (h *Handler) displayResult(result *application.ProcessResult) {
	// Add output section header
	fmt.Println()
	fmt.Println("OUTPUT:")
	fmt.Println(strings.Repeat("-", 40))

	if result.Output == nil {
		h.displaySuccess("Command executed successfully")
		return
	}

	// Format output based on type
	switch output := result.Output.(type) {
	case map[string]interface{}:
		h.displayMapOutput(output)
	case []interface{}:
		h.displayListOutput(output)
	case string:
		fmt.Println(output)
	default:
		fmt.Printf("Result: %+v\n", output)
	}

	// Show execution time if significant
	if result.Duration > 100*time.Millisecond {
		h.displayInfo(fmt.Sprintf("Executed in %v", result.Duration.Round(time.Millisecond)))
	}

	// End output section
	fmt.Println(strings.Repeat("-", 40))
}

// displayMapOutput formats and displays map output
func (h *Handler) displayMapOutput(output map[string]interface{}) {
	for key, value := range output {
		switch key {
		case "message":
			if msg, ok := value.(string); ok {
				fmt.Println(msg)
			}
		case "commands":
			if commands, ok := value.([]interface{}); ok {
				h.displayCommandHelp(commands)
			}
		default:
			fmt.Printf("%s: %v\n", key, value)
		}
	}
}

// displayListOutput formats and displays list output
func (h *Handler) displayListOutput(output []interface{}) {
	for i, item := range output {
		fmt.Printf("%d. %v\n", i+1, item)
	}
}

// displayCommandHelp displays command help information
func (h *Handler) displayCommandHelp(commands []interface{}) {
	fmt.Println("Available commands:")
	for _, cmd := range commands {
		if cmdMap, ok := cmd.(map[string]interface{}); ok {
			if command, ok := cmdMap["command"].(string); ok {
				description, _ := cmdMap["description"].(string)
				fmt.Printf("  %-20s %s\n", command, description)

				if examples, ok := cmdMap["examples"].([]interface{}); ok && len(examples) > 0 {
					fmt.Printf("    Examples: %v\n", examples)
				}
			}
		}
	}
}

// displayContext shows the current session context
func (h *Handler) displayContext(context *domain.Context) {
	fmt.Println("Current Context:")
	fmt.Printf("  Session ID: %s\n", context.SessionID)
	fmt.Printf("  Duration: %s\n", context.GetSessionDuration().Round(time.Second))
	fmt.Printf("  Commands executed: %d\n", len(context.Commands))

	if context.CurrentProject != "" {
		fmt.Printf("  Project: %s\n", context.CurrentProject)
	}
	if context.CurrentSprint != "" {
		fmt.Printf("  Sprint: %s\n", context.CurrentSprint)
	}
	if context.CurrentSpace != "" {
		fmt.Printf("  Space: %s\n", context.CurrentSpace)
	}
	if len(context.RecentAssets) > 0 {
		fmt.Printf("  Recent assets: %v\n", context.RecentAssets)
	}
	if len(context.RecentTasks) > 0 {
		fmt.Printf("  Recent tasks: %v\n", context.RecentTasks)
	}
}

// endSession terminates the console session
func (h *Handler) endSession(ctx context.Context) error {
	if h.sessionContext != nil {
		return h.consoleService.EndSession(ctx, h.sessionContext.SessionID)
	}
	return nil
}

// ErrExitRequested indicates the user requested to exit
var ErrExitRequested = fmt.Errorf("exit requested")
