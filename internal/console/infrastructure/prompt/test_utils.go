package prompt

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// setupTestEnvironment isolates terminal state for tests
func setupTestEnvironment(_ *testing.T) func() {
	// Save original state
	originalStdin := os.Stdin
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Create test buffers
	testStdin := &bytes.Buffer{}
	testStdout := &bytes.Buffer{}
	testStderr := &bytes.Buffer{}

	// Override standard streams
	r, w, _ := os.Pipe()
	os.Stdin = r
	os.Stdout = w
	os.Stderr = w

	// Return cleanup function
	return func() {
		w.Close()
		r.Close()

		// Restore original state
		os.Stdin = originalStdin
		os.Stdout = originalStdout
		os.Stderr = originalStderr

		// Drain buffers
		_, _ = io.Copy(io.Discard, testStdin)
		_, _ = io.Copy(io.Discard, testStdout)
		_, _ = io.Copy(io.Discard, testStderr)
	}
}
