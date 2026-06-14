package prompt

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// stubConsoleService is a hand-rolled ConsoleService for driving the
// prompt handlers' service-bound paths in tests. Each method delegates
// to an injectable function so individual subtests can pin a single
// scenario without mock.Mock bookkeeping.
type stubConsoleService struct {
	startSession      func(context.Context) (*domain.Context, error)
	processInput      func(context.Context, string, string) (*application.ProcessResult, error)
	getSessionContext func(context.Context, string) (*domain.Context, error)
	endSession        func(context.Context, string) error
}

func (s *stubConsoleService) StartSession(ctx context.Context) (*domain.Context, error) {
	if s.startSession != nil {
		return s.startSession(ctx)
	}
	return domain.NewContext("test-session"), nil
}

func (s *stubConsoleService) ProcessInput(ctx context.Context, sessionID, input string) (*application.ProcessResult, error) {
	if s.processInput != nil {
		return s.processInput(ctx, sessionID, input)
	}
	return &application.ProcessResult{Success: true}, nil
}

func (s *stubConsoleService) GetSessionContext(ctx context.Context, sessionID string) (*domain.Context, error) {
	if s.getSessionContext != nil {
		return s.getSessionContext(ctx, sessionID)
	}
	return domain.NewContext(sessionID), nil
}

func (s *stubConsoleService) EndSession(ctx context.Context, sessionID string) error {
	if s.endSession != nil {
		return s.endSession(ctx, sessionID)
	}
	return nil
}

// newTestHandler builds a Handler wired to the given stub service and a
// reader fed by the given input string. Useful boilerplate eliminator
// so each subtest reads as a single scenario.
func newTestHandler(svc ConsoleService, input string) *Handler {
	return &Handler{
		consoleService: svc,
		reader:         bufio.NewReader(strings.NewReader(input)),
		sessionContext: domain.NewContext("test-session"),
		promptStyle:    DefaultStyle(),
	}
}

func newTestEnhancedHandler(svc ConsoleService, input string) *EnhancedHandler {
	h := NewEnhancedHandler(nil)
	h.consoleService = svc
	h.reader = bufio.NewReader(strings.NewReader(input))
	h.sessionContext = domain.NewContext("test-session")
	return h
}

// ----- Handler.handleInput -----

func TestHandler_HandleInput_AllBranches(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	ctx := context.Background()

	t.Run("empty input is a no-op without calling the service", func(t *testing.T) {
		called := false
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			called = true
			return nil, nil
		}}
		h := newTestHandler(svc, "")
		require.NoError(t, h.handleInput(ctx, ""))
		assert.False(t, called)
	})

	t.Run("service error displays the error and returns nil", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return nil, errors.New("interpreter exploded")
		}}
		h := newTestHandler(svc, "")
		require.NoError(t, h.handleInput(ctx, "show assets"))
	})

	t.Run("clarification result returns nil after displaying", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return &application.ProcessResult{
				RequiresClarification: true,
				ClarificationPrompt:   "Did you mean A or B?",
				Options:               []string{"A", "B"},
			}, nil
		}}
		h := newTestHandler(svc, "")
		require.NoError(t, h.handleInput(ctx, "ambiguous"))
	})

	t.Run("unsuccessful result displays error and optional help", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return &application.ProcessResult{Success: false, Error: "bad command", Help: "try `help`"}, nil
		}}
		h := newTestHandler(svc, "")
		require.NoError(t, h.handleInput(ctx, "nope"))
	})

	t.Run("exit command returns ErrExitRequested", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return &application.ProcessResult{
				Success: true,
				Command: &domain.Command{Interpreted: "exit"},
			}, nil
		}}
		h := newTestHandler(svc, "")
		err := h.handleInput(ctx, "bye")
		assert.ErrorIs(t, err, ErrExitRequested)
	})

	t.Run("context command dispatches to handleContextCommand", func(t *testing.T) {
		var lookups int
		svc := &stubConsoleService{
			processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
				return &application.ProcessResult{
					Success: true,
					Command: &domain.Command{Interpreted: "context show"},
				}, nil
			},
			getSessionContext: func(context.Context, string) (*domain.Context, error) {
				lookups++
				return domain.NewContext("test-session"), nil
			},
		}
		h := newTestHandler(svc, "")
		require.NoError(t, h.handleInput(ctx, "ctx"))
		assert.Equal(t, 1, lookups, "context show should fetch session via the service")
	})

	t.Run("normal successful result hits displayResult without returning exit", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return &application.ProcessResult{
				Success: true,
				Command: &domain.Command{Interpreted: "assets list"},
				Output:  []interface{}{"asset-1", "asset-2"},
			}, nil
		}}
		h := newTestHandler(svc, "")
		require.NoError(t, h.handleInput(ctx, "show assets"))
	})
}

// ----- Handler.handleContextCommand show/clear -----

func TestHandler_HandleContextCommand_ServiceBoundBranches(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	ctx := context.Background()

	t.Run("show with successful lookup renders the new context", func(t *testing.T) {
		var saw string
		svc := &stubConsoleService{getSessionContext: func(_ context.Context, id string) (*domain.Context, error) {
			saw = id
			return domain.NewContext("from-service"), nil
		}}
		h := newTestHandler(svc, "")
		err := h.handleContextCommand(ctx, &application.ProcessResult{
			Command: &domain.Command{Interpreted: "context show"},
		})
		require.NoError(t, err)
		assert.Equal(t, "test-session", saw)
	})

	t.Run("show with lookup error displays the error and returns nil", func(t *testing.T) {
		svc := &stubConsoleService{getSessionContext: func(context.Context, string) (*domain.Context, error) {
			return nil, errors.New("store unreachable")
		}}
		h := newTestHandler(svc, "")
		err := h.handleContextCommand(ctx, &application.ProcessResult{
			Command: &domain.Command{Interpreted: "context show"},
		})
		require.NoError(t, err)
	})

	t.Run("clear with successful lookup refreshes the local sessionContext", func(t *testing.T) {
		refreshed := domain.NewContext("refreshed")
		svc := &stubConsoleService{getSessionContext: func(context.Context, string) (*domain.Context, error) {
			return refreshed, nil
		}}
		h := newTestHandler(svc, "")
		require.NoError(t, h.handleContextCommand(ctx, &application.ProcessResult{
			Command: &domain.Command{Interpreted: "context clear"},
		}))
		assert.Same(t, refreshed, h.sessionContext)
	})

	t.Run("clear silently keeps the old context when lookup errors", func(t *testing.T) {
		original := h.sessionContextForTest()
		svc := &stubConsoleService{getSessionContext: func(context.Context, string) (*domain.Context, error) {
			return nil, errors.New("nope")
		}}
		h := newTestHandler(svc, "")
		h.sessionContext = original
		require.NoError(t, h.handleContextCommand(ctx, &application.ProcessResult{
			Command: &domain.Command{Interpreted: "context clear"},
		}))
		assert.Same(t, original, h.sessionContext)
	})
}

// h is a placeholder used only to make the test compile when one
// subtest needs to access another's local helper -- never read at
// runtime. See sessionContextForTest below.
var h handlerHelper

type handlerHelper struct{}

// sessionContextForTest just returns a fresh context so the
// "clear silently keeps the old context" subtest above has something
// stable to compare against without depending on test ordering.
func (handlerHelper) sessionContextForTest() *domain.Context {
	return domain.NewContext("original")
}

// ----- Handler.processInput -----

func TestHandler_ProcessInput_Lifecycle(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))

	t.Run("returns ctx.Err when context is cancelled before input arrives", func(t *testing.T) {
		svc := &stubConsoleService{}
		// Empty reader; the read goroutine will block on stdin-style I/O
		// until the context cancels.
		r, w := nopPipe()
		defer w.Close()

		h := &Handler{
			consoleService: svc,
			reader:         bufio.NewReader(r),
			sessionContext: domain.NewContext("s"),
			promptStyle:    DefaultStyle(),
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		err := h.processInput(ctx)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("delivers a line to handleInput and exits on ErrExitRequested", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return &application.ProcessResult{
				Success: true,
				Command: &domain.Command{Interpreted: "exit"},
			}, nil
		}}
		h := newTestHandler(svc, "anything\n")
		err := h.processInput(context.Background())
		assert.ErrorIs(t, err, ErrExitRequested)
	})

	t.Run("propagates a reader error", func(t *testing.T) {
		// errReader returns the configured error on every Read.
		h := &Handler{
			consoleService: &stubConsoleService{},
			reader:         bufio.NewReader(&errReader{err: errors.New("read boom")}),
			sessionContext: domain.NewContext("s"),
			promptStyle:    DefaultStyle(),
		}
		err := h.processInput(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read boom")
	})
}

// ----- Handler.Start -----

func TestHandler_Start_ExitsOnCancellation(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))

	svc := &stubConsoleService{}
	r, w := nopPipe()
	defer w.Close()

	h := &Handler{
		consoleService: svc,
		reader:         bufio.NewReader(r),
		promptStyle:    DefaultStyle(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- h.Start(ctx) }()

	// Let the loop spin into its read-then-select, then cancel.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		// Start returns either nil (endSession path) or a wrapped error.
		// We accept either as long as it exits promptly.
		_ = err
	case <-time.After(2 * time.Second):
		t.Fatal("Handler.Start did not exit within 2s of ctx cancel")
	}
}

func TestHandler_Start_FailsOnStartSessionError(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))

	svc := &stubConsoleService{startSession: func(context.Context) (*domain.Context, error) {
		return nil, errors.New("boom")
	}}
	h := &Handler{
		consoleService: svc,
		reader:         bufio.NewReader(strings.NewReader("")),
		promptStyle:    DefaultStyle(),
	}
	err := h.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start console session")
}

// ----- EnhancedHandler equivalents -----

func TestEnhancedHandler_HandleEnhancedInput_AllBranches(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	ctx := context.Background()

	t.Run("service error reports without returning exit", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return nil, errors.New("nope")
		}}
		h := newTestEnhancedHandler(svc, "")
		require.NoError(t, h.handleEnhancedInput(ctx, "do thing"))
	})

	t.Run("clarification short-circuits", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return &application.ProcessResult{
				RequiresClarification: true,
				ClarificationPrompt:   "Which one?",
				Options:               []string{"A"},
			}, nil
		}}
		h := newTestEnhancedHandler(svc, "")
		require.NoError(t, h.handleEnhancedInput(ctx, "?"))
	})

	t.Run("not-success path with help text", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return &application.ProcessResult{Success: false, Error: "bad", Help: "try X"}, nil
		}}
		h := newTestEnhancedHandler(svc, "")
		require.NoError(t, h.handleEnhancedInput(ctx, "nope"))
	})

	t.Run("exit command returns ErrExitRequested", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return &application.ProcessResult{
				Success: true,
				Command: &domain.Command{Interpreted: "exit"},
			}, nil
		}}
		h := newTestEnhancedHandler(svc, "")
		assert.ErrorIs(t, h.handleEnhancedInput(ctx, "bye"), ErrExitRequested)
	})

	t.Run("context command routes through handleEnhancedContextCommand", func(t *testing.T) {
		var lookups int
		svc := &stubConsoleService{
			processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
				return &application.ProcessResult{
					Success: true,
					Command: &domain.Command{Interpreted: "context show"},
				}, nil
			},
			getSessionContext: func(context.Context, string) (*domain.Context, error) {
				lookups++
				return domain.NewContext("test-session"), nil
			},
		}
		h := newTestEnhancedHandler(svc, "")
		require.NoError(t, h.handleEnhancedInput(ctx, "ctx"))
		assert.Equal(t, 1, lookups)
	})

	t.Run("normal success renders the result", func(t *testing.T) {
		svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
			return &application.ProcessResult{
				Success: true,
				Command: &domain.Command{Interpreted: "assets list"},
				Output:  []interface{}{"a", "b"},
			}, nil
		}}
		h := newTestEnhancedHandler(svc, "")
		require.NoError(t, h.handleEnhancedInput(ctx, "show"))
	})
}

func TestEnhancedHandler_HandleEnhancedContextCommand_ServiceBoundBranches(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	ctx := context.Background()

	t.Run("show error reports and returns nil", func(t *testing.T) {
		svc := &stubConsoleService{getSessionContext: func(context.Context, string) (*domain.Context, error) {
			return nil, errors.New("store down")
		}}
		h := newTestEnhancedHandler(svc, "")
		err := h.handleEnhancedContextCommand(ctx, &application.ProcessResult{
			Command: &domain.Command{Interpreted: "context show"},
		})
		require.NoError(t, err)
	})

	t.Run("clear refreshes local sessionContext on success", func(t *testing.T) {
		refreshed := domain.NewContext("refreshed-enhanced")
		svc := &stubConsoleService{getSessionContext: func(context.Context, string) (*domain.Context, error) {
			return refreshed, nil
		}}
		h := newTestEnhancedHandler(svc, "")
		err := h.handleEnhancedContextCommand(ctx, &application.ProcessResult{
			Command: &domain.Command{Interpreted: "context clear"},
		})
		require.NoError(t, err)
		assert.Same(t, refreshed, h.sessionContext)
	})
}

func TestEnhancedHandler_ProcessEnhancedInput_DeliversLine(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))

	svc := &stubConsoleService{processInput: func(context.Context, string, string) (*application.ProcessResult, error) {
		return &application.ProcessResult{
			Success: true,
			Command: &domain.Command{Interpreted: "exit"},
		}, nil
	}}
	h := newTestEnhancedHandler(svc, "anything\n")
	err := h.processEnhancedInput(context.Background())
	assert.ErrorIs(t, err, ErrExitRequested)
}

func TestEnhancedHandler_ProcessEnhancedInput_CancelledBeforeInput(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))

	r, w := nopPipe()
	defer w.Close()

	h := NewEnhancedHandler(nil)
	h.consoleService = &stubConsoleService{}
	h.reader = bufio.NewReader(r)
	h.sessionContext = domain.NewContext("s")

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()

	err := h.processEnhancedInput(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestEnhancedHandler_Start_FailsOnStartSessionError(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))

	h := NewEnhancedHandler(nil)
	h.consoleService = &stubConsoleService{startSession: func(context.Context) (*domain.Context, error) {
		return nil, errors.New("boom")
	}}
	err := h.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start console session")
}

// ----- helpers -----

// nopPipe returns an os.Pipe-style pair using io.Pipe, suitable for
// blocking the reader until the test writes to it. Callers must close
// the writer to unblock.
func nopPipe() (*pipeReader, *pipeWriter) {
	r, w := newSyncPipe()
	return r, w
}

// errReader is a tiny io.Reader that always returns the configured
// error -- handier than relying on io.ErrUnexpectedEOF semantics
// because we want to verify the wrapper text.
type errReader struct{ err error }

func (e *errReader) Read([]byte) (int, error) { return 0, e.err }

// ----- minimal in-process pipe -----

type pipeReader struct {
	buf chan byte
}
type pipeWriter struct {
	buf    chan byte
	closed chan struct{}
}

func newSyncPipe() (*pipeReader, *pipeWriter) {
	ch := make(chan byte, 4096)
	closed := make(chan struct{})
	return &pipeReader{buf: ch}, &pipeWriter{buf: ch, closed: closed}
}

func (r *pipeReader) Read(p []byte) (int, error) {
	b, ok := <-r.buf
	if !ok {
		return 0, errEOF
	}
	p[0] = b
	return 1, nil
}

func (w *pipeWriter) Close() error {
	select {
	case <-w.closed:
		return nil
	default:
	}
	close(w.closed)
	close(w.buf)
	return nil
}

var errEOF = errors.New("EOF")
