package ports

// StatusPort defines the interface for status normalization in the domain layer
// This follows hexagonal architecture by keeping external dependencies at the boundary
type StatusPort interface {
	// IsInProgress checks if a status represents work in progress for a specific team and board
	IsInProgress(status string, teamKey string, boardID string) bool

	// IsDone checks if a status represents completed work for a specific team and board
	IsDone(status string, teamKey string, boardID string) bool

	// IsWontDo checks if a status represents work that won't be done for a specific team and board
	IsWontDo(status string, teamKey string, boardID string) bool

	// GetBoardIDForTeam gets the board ID for a team (returns first board if multiple, empty if none)
	GetBoardIDForTeam(teamKey string) string
}
