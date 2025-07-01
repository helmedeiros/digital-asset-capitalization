package ports

// UserInteraction defines the contract for user interaction
type UserInteraction interface {
	// PromptString prompts the user for a string input
	PromptString(message string) (string, error)

	// PromptStringWithDefault prompts the user for a string input with a default value
	PromptStringWithDefault(message, defaultValue string) (string, error)

	// PromptPassword prompts the user for a password (hidden input)
	PromptPassword(message string) (string, error)

	// PromptConfirm prompts the user for a yes/no confirmation
	PromptConfirm(message string) (bool, error)

	// PromptSelect prompts the user to select from a list of options
	PromptSelect(message string, options []string) (string, error)

	// PromptMultiSelect prompts the user to select multiple options
	PromptMultiSelect(message string, options []string) ([]string, error)

	// DisplayMessage displays a message to the user
	DisplayMessage(message string)

	// DisplayError displays an error message to the user
	DisplayError(message string)

	// DisplaySuccess displays a success message to the user
	DisplaySuccess(message string)

	// DisplayWarning displays a warning message to the user
	DisplayWarning(message string)
}
