package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// CLIUserInput implements UserInput interface for command-line interaction
type CLIUserInput struct {
	reader *bufio.Reader
}

// NewCLIUserInput creates a new CLIUserInput instance
func NewCLIUserInput() *CLIUserInput {
	return &CLIUserInput{
		reader: bufio.NewReader(os.Stdin),
	}
}

// Confirm asks the user for a yes/no confirmation via command line
func (ui *CLIUserInput) Confirm(format string, args ...interface{}) (bool, error) {
	// Print the formatted message
	fmt.Printf(format+" (y/n): ", args...)

	// Read user input
	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	// Clean and normalize the input
	input = strings.TrimSpace(strings.ToLower(input))

	// Check for valid responses
	switch input {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid input: %s. Please answer with 'y' or 'n'", input)
	}
}
