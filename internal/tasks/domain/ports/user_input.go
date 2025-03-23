package ports

// UserInput defines the interface for handling user interactions
type UserInput interface {
	// Confirm asks the user for a yes/no confirmation
	Confirm(format string, args ...interface{}) (bool, error)
}
