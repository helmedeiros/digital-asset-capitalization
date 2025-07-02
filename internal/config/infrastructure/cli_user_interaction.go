package infrastructure

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain/ports"
)

// CLIUserInteraction implements the ports.UserInteraction interface for CLI
type CLIUserInteraction struct {
	input  io.Reader
	output io.Writer
}

// NewCLIUserInteraction creates a new CLIUserInteraction instance using stdin/stdout
func NewCLIUserInteraction() ports.UserInteraction {
	return &CLIUserInteraction{
		input:  os.Stdin,
		output: os.Stdout,
	}
}

// NewCLIUserInteractionWithIO creates a new CLIUserInteraction with custom I/O (for testing)
func NewCLIUserInteractionWithIO(input io.Reader, output io.Writer) ports.UserInteraction {
	return &CLIUserInteraction{
		input:  input,
		output: output,
	}
}

// PromptString prompts the user for a string input
func (c *CLIUserInteraction) PromptString(message string) (string, error) {
	fmt.Fprintf(c.output, "%s ", message)
	reader := bufio.NewReader(c.input)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response), nil
}

// PromptStringWithDefault prompts the user for a string input with a default value
func (c *CLIUserInteraction) PromptStringWithDefault(message, defaultValue string) (string, error) {
	fmt.Fprintf(c.output, "%s [%s]: ", message, defaultValue)
	reader := bufio.NewReader(c.input)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	response = strings.TrimSpace(response)
	if response == "" {
		return defaultValue, nil
	}
	return response, nil
}

// PromptPassword prompts the user for a password (hidden input)
func (c *CLIUserInteraction) PromptPassword(message string) (string, error) {
	// For now, just use regular input - in a real implementation, we'd use something like
	// golang.org/x/term to hide the input
	fmt.Fprintf(c.output, "%s (hidden): ", message)
	reader := bufio.NewReader(c.input)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response), nil
}

// PromptConfirm prompts the user for a yes/no confirmation
func (c *CLIUserInteraction) PromptConfirm(message string) (bool, error) {
	fmt.Fprintf(c.output, "%s [y/N]: ", message)
	reader := bufio.NewReader(c.input)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// PromptSelect prompts the user to select from a list of options
func (c *CLIUserInteraction) PromptSelect(message string, options []string) (string, error) {
	reader := bufio.NewReader(c.input)

	for {
		fmt.Fprintf(c.output, "%s\n", message)
		for i, option := range options {
			fmt.Fprintf(c.output, "%d) %s\n", i+1, option)
		}
		fmt.Fprintf(c.output, "Enter your choice (number or name): ")

		response, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		response = strings.TrimSpace(response)

		// Try to parse as number
		if num, err := strconv.Atoi(response); err == nil {
			if num >= 1 && num <= len(options) {
				return options[num-1], nil
			}
		}

		// Try to match by name
		for _, option := range options {
			if strings.EqualFold(response, option) {
				return option, nil
			}
		}

		fmt.Fprintf(c.output, "Invalid selection. Please try again.\n")
	}
}

// PromptMultiSelect prompts the user to select multiple options
func (c *CLIUserInteraction) PromptMultiSelect(message string, options []string) ([]string, error) {
	fmt.Fprintf(c.output, "%s\n", message)
	for i, option := range options {
		fmt.Fprintf(c.output, "%d) %s\n", i+1, option)
	}
	fmt.Fprintf(c.output, "Enter your choices (comma-separated numbers or names, or press enter for none): ")

	reader := bufio.NewReader(c.input)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	response = strings.TrimSpace(response)

	if response == "" {
		return []string{}, nil
	}

	var selected []string
	choices := strings.Split(response, ",")

	for _, choice := range choices {
		choice = strings.TrimSpace(choice)

		// Try to parse as number
		if num, err := strconv.Atoi(choice); err == nil {
			if num >= 1 && num <= len(options) {
				selected = append(selected, options[num-1])
				continue
			}
		}

		// Try to match by name
		for _, option := range options {
			if strings.EqualFold(choice, option) {
				selected = append(selected, option)
				break
			}
		}
	}

	return selected, nil
}

// DisplayMessage displays a message to the user
func (c *CLIUserInteraction) DisplayMessage(message string) {
	fmt.Fprintf(c.output, "%s\n", message)
}

// DisplayError displays an error message to the user
func (c *CLIUserInteraction) DisplayError(message string) {
	fmt.Fprintf(c.output, "❌ %s\n", message)
}

// DisplaySuccess displays a success message to the user
func (c *CLIUserInteraction) DisplaySuccess(message string) {
	fmt.Fprintf(c.output, "✅ %s\n", message)
}

// DisplayWarning displays a warning message to the user
func (c *CLIUserInteraction) DisplayWarning(message string) {
	fmt.Fprintf(c.output, "⚠️ %s\n", message)
}
