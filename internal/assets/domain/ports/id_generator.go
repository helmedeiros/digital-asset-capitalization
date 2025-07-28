package ports

// IDGenerator defines the interface for generating unique identifiers
type IDGenerator interface {
	// GenerateID creates a unique ID based on the given name
	GenerateID(name string) string
}
