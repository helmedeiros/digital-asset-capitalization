package httputil

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRetryableStatus(t *testing.T) {
	cases := []struct {
		code      int
		retryable bool
	}{
		{http.StatusOK, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusNotFound, false},
		{http.StatusRequestTimeout, true},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
	}
	for _, c := range cases {
		assert.Equal(t, c.retryable, IsRetryableStatus(c.code), "status %d", c.code)
	}
}

func TestRetryPolicy_ResolvedFallsBackToDefaults(t *testing.T) {
	p := RetryPolicy{}.resolved()
	assert.Equal(t, DefaultMaxAttempts, p.MaxAttempts)
	assert.Equal(t, DefaultBackoffBase, p.BackoffBase)

	p = RetryPolicy{MaxAttempts: 5, BackoffBase: 7 * time.Millisecond}.resolved()
	assert.Equal(t, 5, p.MaxAttempts)
	assert.Equal(t, 7*time.Millisecond, p.BackoffBase)
}

func TestDoWithRetry_RetriesTransientThenSucceeds(t *testing.T) {
	var hits int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt64(&hits, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`busy`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	}))
	defer server.Close()

	resp, err := DoWithRetry(http.DefaultClient,
		RetryPolicy{MaxAttempts: 3, BackoffBase: time.Microsecond},
		"upstream",
		func() (*http.Request, error) {
			return http.NewRequest("GET", server.URL+"/anything", nil)
		})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int64(3), atomic.LoadInt64(&hits))
}

func TestDoWithRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	var hits int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	resp, err := DoWithRetry(http.DefaultClient,
		RetryPolicy{MaxAttempts: 4, BackoffBase: time.Microsecond},
		"upstream",
		func() (*http.Request, error) {
			return http.NewRequest("GET", server.URL+"/anything", nil)
		})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "upstream failed after 4 attempts")
	assert.Equal(t, int64(4), atomic.LoadInt64(&hits))
}

func TestDoWithRetry_DoesNotRetryClientErrors(t *testing.T) {
	var hits int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	resp, err := DoWithRetry(http.DefaultClient,
		RetryPolicy{MaxAttempts: 5, BackoffBase: time.Microsecond},
		"upstream",
		func() (*http.Request, error) {
			return http.NewRequest("GET", server.URL+"/anything", nil)
		})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, int64(1), atomic.LoadInt64(&hits), "4xx must not retry")
}

func TestDoWithRetry_RetriesTransportError(t *testing.T) {
	// Spin a server to get an address, then close it so every attempt
	// fails at the transport layer.
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()

	resp, err := DoWithRetry(http.DefaultClient,
		RetryPolicy{MaxAttempts: 3, BackoffBase: time.Microsecond},
		"upstream",
		func() (*http.Request, error) {
			return http.NewRequest("GET", server.URL+"/anything", nil)
		})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "upstream failed after 3 attempts")
}

func TestDoWithRetry_EmptyErrLabelFallsBackToRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := DoWithRetry(http.DefaultClient,
		RetryPolicy{MaxAttempts: 2, BackoffBase: time.Microsecond},
		"",
		func() (*http.Request, error) {
			return http.NewRequest("GET", server.URL+"/anything", nil)
		})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed after 2 attempts")
}

func TestDoWithRetry_BuildRequestErrorPropagates(t *testing.T) {
	resp, err := DoWithRetry(http.DefaultClient,
		RetryPolicy{MaxAttempts: 3, BackoffBase: time.Microsecond},
		"upstream",
		func() (*http.Request, error) {
			return http.NewRequest("GET", "http://example.invalid\x00", nil)
		})
	require.Error(t, err)
	assert.Nil(t, resp)
	// We want the *request build* error, not a wrapped retry error -- if
	// the caller can't build a request we shouldn't lie about retries.
	assert.NotContains(t, err.Error(), "upstream failed after")
}
