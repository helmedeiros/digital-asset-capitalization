package cli

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
			ui := &CLIUserInput{
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
