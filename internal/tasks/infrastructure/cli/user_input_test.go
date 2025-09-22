package cli

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sprintPorts "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

func TestCLIUserInput_Confirm(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "accept with 'y'",
			input:          "y\n",
			expectedOutput: "Test message (y/n): ",
			expectedResult: true,
			expectError:    false,
		},
		{
			name:           "accept with 'yes'",
			input:          "yes\n",
			expectedOutput: "Test message (y/n): ",
			expectedResult: true,
			expectError:    false,
		},
		{
			name:           "reject with 'n'",
			input:          "n\n",
			expectedOutput: "Test message (y/n): ",
			expectedResult: false,
			expectError:    false,
		},
		{
			name:           "reject with 'no'",
			input:          "no\n",
			expectedOutput: "Test message (y/n): ",
			expectedResult: false,
			expectError:    false,
		},
		{
			name:           "invalid input",
			input:          "invalid\n",
			expectedOutput: "Test message (y/n): ",
			expectedResult: false,
			expectError:    true,
		},
		{
			name:           "empty input",
			input:          "\n",
			expectedOutput: "Test message (y/n): ",
			expectedResult: false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			var output bytes.Buffer

			// Create a reader with the test input
			reader := strings.NewReader(tt.input)

			// Create CLIUserInput with test reader and writer
			ui := &UserInput{
				reader: bufio.NewReader(reader),
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute the test
			result, err := ui.Confirm("Test message")

			// Restore stdout
			os.Stdout = oldStdout

			// Close the writer
			w.Close()

			// Read the output
			io.Copy(&output, r)

			// Verify the output
			assert.Equal(t, tt.expectedOutput, output.String())

			// Verify the result
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestNewUserInput(t *testing.T) {
	t.Run("should create UserInput with reader", func(t *testing.T) {
		userInput := NewUserInput()

		assert.NotNil(t, userInput, "UserInput should not be nil")
		assert.NotNil(t, userInput.reader, "Reader should not be nil")
	})
}

func TestUserInput_SelectSprint(t *testing.T) {
	createTestCandidates := func() []ports.SprintCandidate {
		return []ports.SprintCandidate{
			{
				Sprint: sprintPorts.Sprint{
					ID:        "123",
					Name:      "🇵🇦 Panama",
					State:     "closed",
					StartDate: "2025-08-20T00:00:00.000Z",
					EndDate:   "2025-09-03T00:00:00.000Z",
				},
				Reason: "exact match",
			},
			{
				Sprint: sprintPorts.Sprint{
					ID:        "456",
					Name:      "Panama City",
					State:     "active",
					StartDate: "2025-09-04T00:00:00.000Z",
					EndDate:   "2025-09-18T00:00:00.000Z",
				},
				Reason: "name contains 'Panama'",
			},
		}
	}

	tests := []struct {
		name           string
		candidates     []ports.SprintCandidate
		input          string
		expectedResult *sprintPorts.Sprint
		expectError    bool
		errorContains  string
	}{
		{
			name:          "empty candidates should return error",
			candidates:    []ports.SprintCandidate{},
			input:         "1\n",
			expectError:   true,
			errorContains: "no sprint candidates provided",
		},
		{
			name:       "valid selection should return selected sprint",
			candidates: createTestCandidates(),
			input:      "1\n",
			expectedResult: &sprintPorts.Sprint{
				ID:        "123",
				Name:      "🇵🇦 Panama",
				State:     "closed",
				StartDate: "2025-08-20T00:00:00.000Z",
				EndDate:   "2025-09-03T00:00:00.000Z",
			},
			expectError: false,
		},
		{
			name:       "second option selection",
			candidates: createTestCandidates(),
			input:      "2\n",
			expectedResult: &sprintPorts.Sprint{
				ID:        "456",
				Name:      "Panama City",
				State:     "active",
				StartDate: "2025-09-04T00:00:00.000Z",
				EndDate:   "2025-09-18T00:00:00.000Z",
			},
			expectError: false,
		},
		{
			name:           "empty input should cancel selection",
			candidates:     createTestCandidates(),
			input:          "\n",
			expectedResult: nil,
			expectError:    false,
		},
		{
			name:          "invalid number should return error",
			candidates:    createTestCandidates(),
			input:         "abc\n",
			expectError:   true,
			errorContains: "invalid selection",
		},
		{
			name:          "out of range selection should return error",
			candidates:    createTestCandidates(),
			input:         "5\n",
			expectError:   true,
			errorContains: "invalid selection: 5",
		},
		{
			name:          "zero selection should return error",
			candidates:    createTestCandidates(),
			input:         "0\n",
			expectError:   true,
			errorContains: "invalid selection: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			var output bytes.Buffer

			// Create a reader with the test input
			reader := strings.NewReader(tt.input)

			// Create UserInput with test reader
			ui := &UserInput{
				reader: bufio.NewReader(reader),
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute the test
			result, err := ui.SelectSprint(tt.candidates)

			// Restore stdout
			os.Stdout = oldStdout

			// Close the writer
			w.Close()

			// Read the output
			io.Copy(&output, r)

			// Verify the result
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				if tt.expectedResult == nil {
					assert.Nil(t, result, "result should be nil when user cancels")
				} else {
					require.NotNil(t, result, "result should not be nil")
					assert.Equal(t, tt.expectedResult.ID, result.ID)
					assert.Equal(t, tt.expectedResult.Name, result.Name)
					assert.Equal(t, tt.expectedResult.State, result.State)
				}
			}

			// Verify output contains expected elements
			if len(tt.candidates) > 0 && !tt.expectError {
				output := output.String()
				assert.Contains(t, output, "Sprint not found exactly", "should show header message")
				assert.Contains(t, output, "Found similar sprints", "should show search results header")

				// Should show all candidates
				for _, candidate := range tt.candidates {
					assert.Contains(t, output, candidate.Sprint.Name, "should show sprint name")
					assert.Contains(t, output, candidate.Sprint.ID, "should show sprint ID")
					if candidate.Sprint.State != "" {
						assert.Contains(t, output, candidate.Sprint.State, "should show sprint state")
					}
					assert.Contains(t, output, candidate.Reason, "should show match reason")
				}
			}
		})
	}
}

func TestUserInput_FormatDate(t *testing.T) {
	ui := &UserInput{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string should return empty",
			input:    "",
			expected: "",
		},
		{
			name:     "ISO date with milliseconds should be formatted",
			input:    "2025-08-20T00:00:00.000Z",
			expected: "Aug 20, 2025",
		},
		{
			name:     "ISO date without milliseconds should be formatted",
			input:    "2025-08-20T00:00:00Z",
			expected: "Aug 20, 2025",
		},
		{
			name:     "RFC3339 date should be formatted",
			input:    "2025-08-20T00:00:00Z",
			expected: "Aug 20, 2025",
		},
		{
			name:     "simple date should be formatted",
			input:    "2025-08-20",
			expected: "Aug 20, 2025",
		},
		{
			name:     "invalid date should return as-is",
			input:    "invalid-date",
			expected: "invalid-date",
		},
		{
			name:     "partial date should return as-is",
			input:    "2025-08",
			expected: "2025-08",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ui.formatDate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
