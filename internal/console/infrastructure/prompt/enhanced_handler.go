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
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/infrastructure/ui"
)

// Command constants to avoid goconst warnings
const (
	CommandClear = "clear"
)

// EnhancedHandler provides a Claude Code-style console interface
type EnhancedHandler struct {
	consoleService ConsoleService
	reader         *bufio.Reader
	sessionContext *domain.Context
	promptSession  *ui.PromptSession
	palette        ui.ColorPalette
	spinner        *ui.Spinner
	spinnerStop    chan struct{} // closed by stopProcessingIndicator to signal exit
	spinnerDone    chan struct{} // closed by the animate goroutine on exit
	smartFormatter *ui.SmartFormatter
	maxWidth       int
	commandHistory []string
	historyIndex   int
}

// EnhancedStyle extends the basic style with UI enhancements
type EnhancedStyle struct {
	Style           // Embed the basic style
	ShowTimestamps  bool
	ShowProgress    bool
	MaxWidth        int
	UseColors       bool
	ClaudeCodeStyle bool
}

// DefaultEnhancedStyle returns a Claude Code-style configuration
func DefaultEnhancedStyle() EnhancedStyle {
	return EnhancedStyle{
		Style:           DefaultStyle(),
		ShowTimestamps:  true,
		ShowProgress:    true,
		MaxWidth:        80,
		UseColors:       true,
		ClaudeCodeStyle: true,
	}
}

// NewEnhancedHandler creates a new enhanced prompt handler
func NewEnhancedHandler(consoleService *application.ConsoleService) *EnhancedHandler {
	style := DefaultEnhancedStyle()

	return &EnhancedHandler{
		consoleService: consoleService,
		reader:         bufio.NewReader(os.Stdin),
		promptSession:  ui.NewPromptSession(style.MaxWidth),
		palette:        ui.DefaultPalette(),
		smartFormatter: ui.NewSmartFormatter(),
		maxWidth:       style.MaxWidth,
		commandHistory: make([]string, 0, 100), // Keep last 100 commands
		historyIndex:   -1,
	}
}

// Start begins the enhanced interactive console session
func (h *EnhancedHandler) Start(ctx context.Context) error {
	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println(ui.WarningText("\nReceived interrupt signal..."))
		cancel()
	}()

	// Start console session
	sessionContext, err := h.consoleService.StartSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to start console session: %w", err)
	}
	h.sessionContext = sessionContext

	// Display enhanced welcome
	h.displayEnhancedWelcome()

	// Main input loop
	for {
		select {
		case <-ctx.Done():
			h.displayEnhancedGoodbye()
			return h.endSession(ctx)
		default:
			if err := h.processEnhancedInput(ctx); err != nil {
				if err == ErrExitRequested {
					h.displayEnhancedGoodbye()
					return h.endSession(ctx)
				}
				h.displayEnhancedError(fmt.Sprintf("Error processing input: %v", err))
			}
		}
	}
}

// processEnhancedInput handles a single input cycle with Claude Code styling
func (h *EnhancedHandler) processEnhancedInput(ctx context.Context) error {
	// Display interactive prompt box with proper cursor positioning
	h.displayInteractivePrompt()

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
		// No need to complete prompt box - we'll handle it in handleEnhancedInput
		return h.handleEnhancedInput(ctx, input)
	case <-time.After(10 * time.Minute):
		h.displayEnhancedInfo("Session timeout due to inactivity (10 minutes)")
		return ErrExitRequested
	}
}

// handleEnhancedInput processes user input with enhanced UI feedback
func (h *EnhancedHandler) handleEnhancedInput(ctx context.Context, input string) error {
	if input == "" {
		return nil
	}

	// Add to command history
	h.addToCommandHistory(input)

	// Replace the active prompt with the completed styled version
	h.completePromptWithInput(input)

	// Handle special commands
	if h.handleSpecialCommands(input) {
		return nil
	}

	// Start processing indicator
	h.startProcessingIndicator("Processing your request...")

	// Record start time
	startTime := time.Now()

	// Process the input through the console service
	result, err := h.consoleService.ProcessInput(ctx, h.sessionContext.SessionID, input)
	duration := time.Since(startTime)

	// Stop processing indicator
	h.stopProcessingIndicator()

	if err != nil {
		h.addToHistory(input, err.Error(), duration, false)
		h.displayEnhancedError(err.Error())
		return nil
	}

	// Handle the result
	if result.RequiresClarification {
		h.displayEnhancedClarification(result.ClarificationPrompt, result.Options)
		return nil
	}

	if !result.Success {
		h.addToHistory(input, result.Error, duration, false)
		h.displayEnhancedError(result.Error)
		if result.Help != "" {
			h.displayEnhancedInfo(result.Help)
		}
		return nil
	}

	// Check for exit command
	if result.Command != nil && result.Command.Interpreted == "exit" {
		return ErrExitRequested
	}

	// Check for context commands
	if result.Command != nil && strings.HasPrefix(result.Command.Interpreted, "context") {
		return h.handleEnhancedContextCommand(ctx, result)
	}

	// Prepare output for history
	outputText := h.formatResultForHistory(result)
	h.addToHistory(input, outputText, duration, true)

	// Display successful result with enhanced formatting
	h.displayEnhancedResult(result)

	return nil
}

// displayInteractivePrompt displays the prompt with cursor positioned inside
func (h *EnhancedHandler) displayInteractivePrompt() {
	// Simple prompt without box drawing
	fmt.Print(ui.PromptText("> "))
}

// completePromptWithInput shows the completed prompt with user input
func (h *EnhancedHandler) completePromptWithInput(input string) {
	// Move cursor up one line to overwrite the active prompt
	fmt.Print("\033[1A")

	// Clear the entire line
	fmt.Print("\r" + strings.Repeat(" ", h.maxWidth) + "\r")

	// Render the completed prompt with muted styling (like Claude Code)
	fmt.Printf("%s %s\n", ui.MutedText(">"), ui.MutedText(input))
}

// addToHistory adds an entry to the prompt session history
func (h *EnhancedHandler) addToHistory(input, output string, duration time.Duration, success bool) {
	h.promptSession.AddEntry(input, output, duration, success)
}

// startProcessingIndicator shows a processing spinner
func (h *EnhancedHandler) startProcessingIndicator(message string) {
	h.spinner = ui.NewSpinner(ui.SpinnerDots, message)
	h.spinner.SetColor(ui.ColorInfo)
	h.spinnerStop = make(chan struct{})
	h.spinnerDone = make(chan struct{})

	// Show initial spinner state
	fmt.Printf("\n%s", h.spinner.Render())

	// Start a goroutine to animate the spinner. We pass the spinner and
	// channels by value so the goroutine never reads h.spinner concurrently
	// with the field being cleared in stopProcessingIndicator.
	go animateSpinner(h.spinner, h.spinnerStop, h.spinnerDone)
}

// stopProcessingIndicator stops the processing spinner. Safe to call when
// no spinner is running.
func (h *EnhancedHandler) stopProcessingIndicator() {
	if h.spinner == nil {
		return
	}

	close(h.spinnerStop)
	<-h.spinnerDone

	// Clear the spinner line. Safe to do now that the animate goroutine
	// has exited and won't race with us.
	fmt.Print("\r" + strings.Repeat(" ", h.maxWidth) + "\r")
	h.spinner = nil
	h.spinnerStop = nil
	h.spinnerDone = nil
}

// animateSpinner animates the given spinner until stop is closed, then
// signals done. It does not touch any handler state, which is what makes
// it race-free against stopProcessingIndicator.
func animateSpinner(spinner *ui.Spinner, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	if spinner == nil {
		return
	}

	ticker := time.NewTicker(spinner.GetInterval())
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			fmt.Printf("\r%s", spinner.Render())
		}
	}
}

// displayEnhancedWelcome shows an enhanced welcome message
func (h *EnhancedHandler) displayEnhancedWelcome() {
	welcome := h.promptSession.RenderWelcome("AssetCap")
	fmt.Println(welcome)
	fmt.Println()
}

// displayEnhancedGoodbye shows an enhanced goodbye message
func (h *EnhancedHandler) displayEnhancedGoodbye() {
	fmt.Println()
	farewell := ui.BoldText(ui.PrimaryText("Thanks for using AssetCap AI Console!"))
	fmt.Println(farewell)

	// Show session summary
	history := h.promptSession.GetHistory()
	if len(history) > 0 {
		summary := fmt.Sprintf("Session completed with %d commands processed.", len(history))
		fmt.Println(ui.MutedText(summary))
	}
}

// displayEnhancedError shows an enhanced error message
func (h *EnhancedHandler) displayEnhancedError(message string) {
	fmt.Println()
	fmt.Println(ui.ErrorText("ERROR:"))

	// Indent error message with better formatting
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			fmt.Printf("   %s\n", ui.ErrorText(line))
		}
	}
	fmt.Println()
}

// displayEnhancedInfo shows an enhanced info message
func (h *EnhancedHandler) displayEnhancedInfo(message string) {
	fmt.Printf("   %s %s\n", ui.InfoText("INFO:"), message)
}

// displayEnhancedSuccess shows an enhanced success message
func (h *EnhancedHandler) displayEnhancedSuccess(message string) {
	fmt.Printf("   %s %s\n", ui.SuccessText("OK:"), message)
}

// displayEnhancedClarification shows an enhanced clarification prompt
func (h *EnhancedHandler) displayEnhancedClarification(prompt string, options []string) {
	fmt.Println()

	// Create a nice clarification box
	clarificationBox := ui.Box(
		fmt.Sprintf("CLARIFICATION: %s", prompt),
		ui.ColorWarning,
	)
	fmt.Println(clarificationBox)

	if len(options) > 0 {
		fmt.Println()
		fmt.Println(ui.Subheader("Options:"))
		optionsList := ui.NumberedList(options)
		fmt.Print(optionsList)
	}
	fmt.Println()
}

// displayEnhancedResult shows command execution results with enhanced formatting
func (h *EnhancedHandler) displayEnhancedResult(result *application.ProcessResult) {
	fmt.Println()

	// Output header
	fmt.Println(ui.Header("OUTPUT"))
	fmt.Println(ui.MutedText(strings.Repeat("─", 40)))
	fmt.Println()

	if result.Output == nil {
		h.displayEnhancedSuccess("Command executed successfully")
		fmt.Println()
		return
	}

	// Format output based on type with enhanced styling
	switch output := result.Output.(type) {
	case map[string]interface{}:
		h.displayEnhancedMapOutput(output)
	case []interface{}:
		h.displayEnhancedListOutput(output)
	case string:
		fmt.Println(output)
	default:
		fmt.Printf("Result: %+v\n", output)
	}

	// Show execution time if significant
	if result.Duration > 100*time.Millisecond {
		durationText := fmt.Sprintf("Completed in %v", result.Duration.Round(time.Millisecond))
		fmt.Println()
		fmt.Printf("   %s\n", ui.MutedText(durationText))
	}

	fmt.Println()
	fmt.Println(ui.MutedText(strings.Repeat("─", 40)))
	fmt.Println()
}

// displayEnhancedMapOutput formats and displays map output with enhanced styling
func (h *EnhancedHandler) displayEnhancedMapOutput(output map[string]interface{}) {
	// Check if this looks like tabular data
	if h.isTabularData(output) {
		h.displayAsTable(output)
		return
	}

	// Check for single asset detail view
	if h.isSingleAssetDetail(output) {
		h.displayAssetDetail(output)
		return
	}

	// Display as enhanced key-value pairs
	pairs := make([]ui.KeyValuePair, 0, len(output))

	// Sort keys for consistent display
	keys := ui.SortMapKeys(output)

	for _, key := range keys {
		value := output[key]

		// Skip internal fields or handle special cases
		switch key {
		case "message":
			if msg, ok := value.(string); ok {
				fmt.Println(ui.SuccessText(msg))
				fmt.Println()
			}
			continue
		case "commands":
			if commands, ok := value.([]interface{}); ok {
				h.displayEnhancedCommandHelp(commands)
			}
			continue
		}

		pairs = append(pairs, ui.KeyValuePair{
			Key:   h.formatKeyName(key),
			Value: value,
		})
	}

	if len(pairs) > 0 {
		formatted := ui.RenderKeyValueList(pairs, ui.AssetCapTableStyle())
		fmt.Print(formatted)
	}
}

// displayEnhancedListOutput formats and displays list output with enhanced styling
func (h *EnhancedHandler) displayEnhancedListOutput(output []interface{}) {
	// Check if this is a list of objects that could be displayed as a table
	if len(output) > 0 {
		if firstItem, ok := output[0].(map[string]interface{}); ok {
			// Try to display as a table
			if h.shouldDisplayAsTable(firstItem) {
				h.displayListAsTable(output)
				return
			}
		}
	}

	// Display as enhanced list
	items := make([]string, 0, len(output))
	for _, item := range output {
		items = append(items, fmt.Sprintf("%v", item))
	}

	numberedList := ui.NumberedList(items)
	fmt.Print(numberedList)
}

// displayEnhancedCommandHelp displays command help information with enhanced styling
func (h *EnhancedHandler) displayEnhancedCommandHelp(commands []interface{}) {
	fmt.Println(ui.Subheader("Available commands:"))
	fmt.Println()

	for _, cmd := range commands {
		if cmdMap, ok := cmd.(map[string]interface{}); ok {
			if command, ok := cmdMap["command"].(string); ok {
				description, _ := cmdMap["description"].(string)

				// Format command name
				fmt.Printf("  %s", ui.Code(command))

				// Add description
				if description != "" {
					padding := 25 - len(command)
					if padding < 2 {
						padding = 2
					}
					fmt.Printf("%s%s", strings.Repeat(" ", padding), description)
				}
				fmt.Println()

				// Add examples if available
				if examples, ok := cmdMap["examples"].([]interface{}); ok && len(examples) > 0 {
					fmt.Printf("    %s ", ui.MutedText("Examples:"))
					for i, example := range examples {
						if i > 0 {
							fmt.Print(", ")
						}
						fmt.Print(ui.Code(fmt.Sprintf("%v", example)))
					}
					fmt.Println()
				}
			}
		}
	}
	fmt.Println()
}

// isTabularData determines if the output should be displayed as a table
func (h *EnhancedHandler) isTabularData(output map[string]interface{}) bool {
	// Look for array/slice values that could be table rows
	for _, value := range output {
		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			if _, ok := arr[0].(map[string]interface{}); ok {
				return true
			}
		}
	}
	return false
}

// shouldDisplayAsTable determines if a list should be displayed as a table
func (h *EnhancedHandler) shouldDisplayAsTable(firstItem map[string]interface{}) bool {
	// If the object has typical "table-like" fields, display as table
	tableFields := []string{"name", "id", "key", "status", "type", "created_at", "updated_at"}

	fieldCount := 0
	for field := range firstItem {
		for _, tableField := range tableFields {
			if strings.Contains(strings.ToLower(field), tableField) {
				fieldCount++
				break
			}
		}
	}

	// If it has 2 or more table-like fields, display as table
	return fieldCount >= 2
}

// displayAsTable displays map data as a table
func (h *EnhancedHandler) displayAsTable(output map[string]interface{}) {
	for key, value := range output {
		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			if _, ok := arr[0].(map[string]interface{}); ok {
				fmt.Println(ui.Subheader(h.formatKeyName(key) + ":"))
				h.displayListAsTable(arr)
				fmt.Println()
			}
		}
	}
}

// displayListAsTable displays a list of objects as a table
func (h *EnhancedHandler) displayListAsTable(items []interface{}) {
	if len(items) == 0 {
		return
	}

	// Convert to table format
	var table *ui.Table

	// Determine table type based on first item fields
	if firstItem, ok := items[0].(map[string]interface{}); ok {
		table = h.createTableForData(firstItem)

		// Add all rows
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				table.AddRow(itemMap)
			}
		}

		// Render the table
		fmt.Println(table.Render())
	}
}

// createTableForData creates an appropriate table based on the data structure
func (h *EnhancedHandler) createTableForData(sample map[string]interface{}) *ui.Table {
	factory := ui.NewAssetCapTableFactory()

	// Check if this looks like asset data
	if h.hasFields(sample, []string{"name", "status"}) {
		return factory.CreateAssetListTable()
	}

	// Check if this looks like task data
	if h.hasFields(sample, []string{"key", "summary"}) {
		return factory.CreateTaskListTable()
	}

	// Check if this looks like investment data
	if h.hasFields(sample, []string{"asset", "investment"}) || h.hasFields(sample, []string{"total_investment"}) {
		return factory.CreateInvestmentSummaryTable()
	}

	// Check if this looks like engineer data
	if h.hasFields(sample, []string{"name", "level", "hours", "cost"}) {
		return factory.CreateInvestmentDetailTable()
	}

	// Check if this looks like sprint data
	if h.hasFields(sample, []string{"name", "start_date", "end_date"}) || h.hasFields(sample, []string{"state", "goal"}) {
		return factory.CreateSprintTable()
	}

	// Generic table
	return h.createGenericTable(sample)
}

// hasFields checks if the map has any of the specified fields
func (h *EnhancedHandler) hasFields(data map[string]interface{}, fields []string) bool {
	for _, field := range fields {
		if _, exists := data[field]; exists {
			return true
		}
	}
	return false
}

// createGenericTable creates a generic table from the data
func (h *EnhancedHandler) createGenericTable(sample map[string]interface{}) *ui.Table {
	columns := make([]ui.TableColumn, 0, len(sample))

	// Sort keys for consistent display
	keys := ui.SortMapKeys(sample)

	for _, key := range keys {
		column := ui.TableColumn{
			Header: h.formatKeyName(key),
			Key:    key,
			Align:  "left",
		}

		// Set formatters based on field name
		switch strings.ToLower(key) {
		case "status", "state":
			column.Formatter = ui.StatusFormatter
		case "created_at", "updated_at", "date", "timestamp":
			column.Formatter = ui.DateFormatter
		case "cost", "investment", "price", "amount":
			if strings.Contains(strings.ToLower(key), "total") {
				column.Formatter = ui.MoneyFormatter
			}
		}

		columns = append(columns, column)
	}

	return ui.NewTable(columns)
}

// formatKeyName formats a key name for display
func (h *EnhancedHandler) formatKeyName(key string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(key, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

// formatResultForHistory formats a result for inclusion in history
func (h *EnhancedHandler) formatResultForHistory(result *application.ProcessResult) string {
	if result.Output == nil {
		return "Command executed successfully"
	}

	// Simple text representation for history
	switch output := result.Output.(type) {
	case map[string]interface{}:
		if msg, exists := output["message"]; exists {
			return fmt.Sprintf("%v", msg)
		}
		return fmt.Sprintf("Returned %d fields", len(output))
	case []interface{}:
		return fmt.Sprintf("Returned %d items", len(output))
	case string:
		return output
	default:
		return fmt.Sprintf("%v", output)
	}
}

// handleEnhancedContextCommand handles context-specific commands with enhanced UI
func (h *EnhancedHandler) handleEnhancedContextCommand(ctx context.Context, result *application.ProcessResult) error {
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
			h.displayEnhancedError(fmt.Sprintf("Failed to get context: %v", err))
			return nil
		}
		h.displayEnhancedContext(sessionContext)
	case CommandClear:
		h.displayEnhancedSuccess("Context cleared")
		// Refresh our local context reference
		sessionContext, err := h.consoleService.GetSessionContext(ctx, h.sessionContext.SessionID)
		if err == nil {
			h.sessionContext = sessionContext
		}
	}

	return nil
}

// displayEnhancedContext shows the current session context with enhanced styling
func (h *EnhancedHandler) displayEnhancedContext(context *domain.Context) {
	fmt.Println()
	fmt.Println(ui.Header("Current Context"))
	fmt.Println(ui.MutedText(strings.Repeat("─", 40)))

	// Create context info as key-value pairs
	pairs := []ui.KeyValuePair{
		{Key: "Session ID", Value: context.SessionID},
		{Key: "Duration", Value: context.GetSessionDuration().Round(time.Second).String()},
		{Key: "Commands executed", Value: len(context.Commands)},
	}

	if context.CurrentProject != "" {
		pairs = append(pairs, ui.KeyValuePair{Key: "Project", Value: context.CurrentProject})
	}
	if context.CurrentSprint != "" {
		pairs = append(pairs, ui.KeyValuePair{Key: "Sprint", Value: context.CurrentSprint})
	}
	if context.CurrentSpace != "" {
		pairs = append(pairs, ui.KeyValuePair{Key: "Space", Value: context.CurrentSpace})
	}
	if len(context.RecentAssets) > 0 {
		pairs = append(pairs, ui.KeyValuePair{Key: "Recent assets", Value: strings.Join(context.RecentAssets, ", ")})
	}
	if len(context.RecentTasks) > 0 {
		pairs = append(pairs, ui.KeyValuePair{Key: "Recent tasks", Value: strings.Join(context.RecentTasks, ", ")})
	}

	formatted := ui.RenderKeyValueList(pairs, ui.DefaultTableStyle())
	fmt.Print(formatted)
	fmt.Println()
}

// isSingleAssetDetail checks if the output is a single asset detail view
func (h *EnhancedHandler) isSingleAssetDetail(output map[string]interface{}) bool {
	// Check if it has typical asset fields and no array data
	hasAssetFields := h.hasFields(output, []string{"name", "id", "description", "status"})
	hasArrayData := h.isTabularData(output)

	return hasAssetFields && !hasArrayData
}

// displayAssetDetail displays a single asset in detail view
func (h *EnhancedHandler) displayAssetDetail(output map[string]interface{}) {
	factory := ui.NewAssetCapTableFactory()
	table := factory.CreateAssetDetailTable()

	// Convert map to detail rows
	rows := ui.ConvertMapToAssetDetailRows(output)

	// Add all rows to the table
	for _, row := range rows {
		table.AddRow(row)
	}

	fmt.Println(table.Render())
}

// addToCommandHistory adds a command to the history
func (h *EnhancedHandler) addToCommandHistory(input string) {
	// Don't add empty commands or duplicate consecutive commands
	if input == "" || (len(h.commandHistory) > 0 && h.commandHistory[len(h.commandHistory)-1] == input) {
		return
	}

	h.commandHistory = append(h.commandHistory, input)

	// Keep only last 100 commands
	if len(h.commandHistory) > 100 {
		h.commandHistory = h.commandHistory[1:]
	}

	// Reset history index
	h.historyIndex = -1
}

// handleSpecialCommands handles built-in console commands
func (h *EnhancedHandler) handleSpecialCommands(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))

	switch lower {
	case "history":
		h.displayCommandHistory()
		return true
	case CommandClear:
		h.clearScreen()
		return true
	case "help":
		h.displayEnhancedHelp()
		return true
	}

	return false
}

// displayCommandHistory shows the command history
func (h *EnhancedHandler) displayCommandHistory() {
	fmt.Println()
	fmt.Println(ui.Header("Command History"))
	fmt.Println(ui.MutedText(strings.Repeat("─", 40)))

	if len(h.commandHistory) == 0 {
		fmt.Println(ui.MutedText("No commands in history"))
		fmt.Println()
		return
	}

	// Show last 20 commands
	start := 0
	if len(h.commandHistory) > 20 {
		start = len(h.commandHistory) - 20
		fmt.Println(ui.MutedText(fmt.Sprintf("Showing last 20 of %d commands:", len(h.commandHistory))))
		fmt.Println()
	}

	for i := start; i < len(h.commandHistory); i++ {
		number := ui.MutedText(fmt.Sprintf("%3d.", i+1))
		command := h.commandHistory[i]
		fmt.Printf("%s %s\n", number, command)
	}
	fmt.Println()
}

// clearScreen clears the terminal screen
func (h *EnhancedHandler) clearScreen() {
	fmt.Print("\033[2J\033[H") // ANSI escape codes to clear screen and move cursor to top
	h.displayEnhancedWelcome()
}

// displayEnhancedHelp shows enhanced help information
func (h *EnhancedHandler) displayEnhancedHelp() {
	fmt.Println()
	fmt.Println(ui.Header("AssetCap AI Console Help"))
	fmt.Println(ui.MutedText(strings.Repeat("─", 60)))
	fmt.Println()

	// Natural language help
	fmt.Println(ui.Subheader("Natural Language Commands:"))
	fmt.Println("You can use natural language to interact with AssetCap:")
	fmt.Println()

	examples := []string{
		"\"Show all assets\"",
		"\"Create an asset called Payment Processing\"",
		"\"List tasks for project FN\"",
		"\"Calculate investment for User Authentication\"",
		"\"Show sprint allocation for Sprint 1\"",
		"\"Get details for asset API Gateway\"",
	}

	for _, example := range examples {
		fmt.Printf("  %s %s\n", ui.InfoText("•"), ui.Code(example))
	}

	fmt.Println()

	// Built-in commands
	fmt.Println(ui.Subheader("Built-in Commands:"))
	builtinCommands := []ui.KeyValuePair{
		{Key: "help", Value: "Show this help message"},
		{Key: "history", Value: "Show command history"},
		{Key: CommandClear, Value: "Clear the screen"},
		{Key: "context show", Value: "Show current session context"},
		{Key: "context clear", Value: "Clear session context"},
		{Key: "exit", Value: "Exit the console"},
	}

	for _, cmd := range builtinCommands {
		fmt.Printf("  %-15s %s\n", ui.Code(cmd.Key), cmd.Value)
	}

	fmt.Println()

	// Tips
	fmt.Println(ui.Subheader("Tips:"))
	tips := []string{
		"Use natural language - the AI will interpret your intent",
		"Commands are processed with context awareness",
		"Previous commands and results influence suggestions",
		"Type 'exit' or press Ctrl+C to quit",
	}

	for _, tip := range tips {
		fmt.Printf("  %s %s\n", ui.InfoText("TIP:"), tip)
	}

	fmt.Println()
}

// endSession terminates the console session
func (h *EnhancedHandler) endSession(ctx context.Context) error {
	if h.sessionContext != nil {
		return h.consoleService.EndSession(ctx, h.sessionContext.SessionID)
	}
	return nil
}
