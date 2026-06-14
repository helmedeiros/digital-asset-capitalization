// Package httputil provides shared HTTP helpers used by infrastructure
// clients (currently the Ollama and JIRA clients). Keeping them here
// avoids two copies of the same retry logic drifting in different
// directions.
package httputil

import (
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"time"
)

// Retry defaults. DefaultMaxAttempts means "one try plus two retries",
// which covers brief transport blips and short server hiccups without
// making a genuinely-broken upstream wait around for ages.
const (
	DefaultMaxAttempts = 3
	DefaultBackoffBase = 500 * time.Millisecond
)

// HTTPClient is the minimum interface satisfied by *http.Client and by
// the test fakes used by the JIRA client. Taking the interface here
// lets the helper work with either without dragging in concrete types.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// RetryPolicy bounds DoWithRetry's behaviour. Zero values fall back to
// the package defaults so callers can pass RetryPolicy{} when they
// don't want to tune.
type RetryPolicy struct {
	MaxAttempts int           // 0 -> DefaultMaxAttempts; 1 disables retries
	BackoffBase time.Duration // 0 -> DefaultBackoffBase
}

// resolved returns a copy with zero fields filled in from the defaults.
func (p RetryPolicy) resolved() RetryPolicy {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = DefaultMaxAttempts
	}
	if p.BackoffBase <= 0 {
		p.BackoffBase = DefaultBackoffBase
	}
	return p
}

// IsRetryableStatus reports whether to retry on the given HTTP status
// code. We retry on the canonical transient-server family (5xx) plus
// 408 (Request Timeout) and 429 (Too Many Requests). 4xx other than
// those is a client error -- repeating the same request will not
// improve matters.
func IsRetryableStatus(code int) bool {
	if code >= 500 && code < 600 {
		return true
	}
	return code == http.StatusRequestTimeout || code == http.StatusTooManyRequests
}

// DoWithRetry sends an HTTP request and retries on transport errors
// and retryable status codes with exponential backoff and full jitter.
// The caller passes a request *builder* rather than a built request so
// the body is fresh on every attempt -- a single bytes.Buffer or
// strings.Reader would be drained after the first try and silently
// send empty payloads on retries.
//
// errLabel prefixes the final wrapped error so callers can identify
// which upstream gave up; e.g. "Ollama request" or "JIRA request".
func DoWithRetry(client HTTPClient, policy RetryPolicy, errLabel string, buildReq func() (*http.Request, error)) (*http.Response, error) {
	p := policy.resolved()

	var lastErr error
	for attempt := 0; attempt < p.MaxAttempts; attempt++ {
		if attempt > 0 {
			sleep := p.BackoffBase << uint(attempt-1)
			// Full jitter halves the worst case and stops fleets of
			// concurrent clients from synchronising into a thundering herd.
			if sleep > 0 {
				sleep = sleep/2 + time.Duration(rand.Int64N(int64(sleep)/2+1))
			}
			time.Sleep(sleep)
		}

		req, err := buildReq()
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if !IsRetryableStatus(resp.StatusCode) {
			return resp, nil
		}

		// Drain and close the retryable response so the connection can
		// return to the pool before we sleep.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
	}

	if errLabel == "" {
		errLabel = "request"
	}
	return nil, fmt.Errorf("%s failed after %d attempts: %w", errLabel, p.MaxAttempts, lastErr)
}
