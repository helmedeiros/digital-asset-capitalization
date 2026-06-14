package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fieldResolverServer returns an httptest server that serves the JIRA
// /rest/api/3/field endpoint with the given payload bytes and status
// code. Other paths return 404 so a missing-route bug surfaces early.
func fieldResolverServer(t *testing.T, body []byte, status int, hits *int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/field" {
			http.NotFound(w, r)
			return
		}
		if hits != nil {
			atomic.AddInt64(hits, 1)
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
}

func TestNewFieldResolver(t *testing.T) {
	r := NewFieldResolver("http://example.invalid", "Bearer xyz")
	require.NotNil(t, r)
	assert.Equal(t, "http://example.invalid", r.baseURL)
	assert.Equal(t, "Bearer xyz", r.authHeader)
	require.NotNil(t, r.httpClient)
	assert.NotZero(t, r.httpClient.Timeout, "client should be built with a timeout")
}

func TestResolveCustomFieldIDs_HappyPath(t *testing.T) {
	body, err := json.Marshal([]Field{
		{ID: "customfield_100", Name: "TPD Business Unit"},
		{ID: "customfield_200", Name: "Engineering time spent (hours)"},
		{ID: "customfield_300", Name: "Work Stream"},
		{ID: "customfield_999", Name: "Some Other Field"},
	})
	require.NoError(t, err)

	server := fieldResolverServer(t, body, http.StatusOK, nil)
	defer server.Close()

	r := NewFieldResolver(server.URL, "Bearer t")
	ids, err := r.ResolveCustomFieldIDs()
	require.NoError(t, err)
	require.NotNil(t, ids)
	assert.Equal(t, "customfield_100", ids.TPDBusinessUnit)
	assert.Equal(t, "customfield_200", ids.EngineeringHours)
	assert.Equal(t, "customfield_300", ids.WorkStream)
}

func TestResolveCustomFieldIDs_OnlyPartialFieldsPresent(t *testing.T) {
	// JIRA may return only some of the expected fields; missing ones
	// should leave the corresponding CustomFieldIDs entry as "".
	body, _ := json.Marshal([]Field{
		{ID: "customfield_100", Name: "TPD Business Unit"},
	})
	server := fieldResolverServer(t, body, http.StatusOK, nil)
	defer server.Close()

	ids, err := NewFieldResolver(server.URL, "x").ResolveCustomFieldIDs()
	require.NoError(t, err)
	require.NotNil(t, ids)
	assert.Equal(t, "customfield_100", ids.TPDBusinessUnit)
	assert.Empty(t, ids.EngineeringHours)
	assert.Empty(t, ids.WorkStream)
}

func TestResolveCustomFieldIDs_CachesAfterFirstCall(t *testing.T) {
	body, _ := json.Marshal([]Field{{ID: "customfield_1", Name: "Work Stream"}})
	var hits int64
	server := fieldResolverServer(t, body, http.StatusOK, &hits)
	defer server.Close()

	r := NewFieldResolver(server.URL, "x")

	// Three calls in a row -- only the first should reach the server.
	first, err := r.ResolveCustomFieldIDs()
	require.NoError(t, err)

	second, err := r.ResolveCustomFieldIDs()
	require.NoError(t, err)

	third, err := r.ResolveCustomFieldIDs()
	require.NoError(t, err)

	assert.Same(t, first, second, "cached pointer should be returned on subsequent calls")
	assert.Same(t, first, third)
	assert.Equal(t, int64(1), atomic.LoadInt64(&hits), "field endpoint should be hit exactly once")
}

func TestResolveCustomFieldIDs_NonOKStatusReturnsError(t *testing.T) {
	server := fieldResolverServer(t, []byte(`forbidden`), http.StatusForbidden, nil)
	defer server.Close()

	ids, err := NewFieldResolver(server.URL, "x").ResolveCustomFieldIDs()
	require.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "unexpected status")
	assert.Contains(t, err.Error(), "403")
}

func TestResolveCustomFieldIDs_MalformedJSONReturnsError(t *testing.T) {
	server := fieldResolverServer(t, []byte(`{not json}`), http.StatusOK, nil)
	defer server.Close()

	ids, err := NewFieldResolver(server.URL, "x").ResolveCustomFieldIDs()
	require.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "decoding fields")
}

func TestResolveCustomFieldIDs_TransportErrorReturnsError(t *testing.T) {
	// Spin a server up just so we have a URL, then close it before
	// calling the resolver so the request fails at the transport layer.
	server := fieldResolverServer(t, []byte(`[]`), http.StatusOK, nil)
	server.Close()

	ids, err := NewFieldResolver(server.URL, "x").ResolveCustomFieldIDs()
	require.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "fetching fields")
}

func TestResolveCustomFieldIDs_InvalidURLFailsAtRequestBuild(t *testing.T) {
	// A NUL byte in the URL breaks http.NewRequest before any I/O.
	r := NewFieldResolver("http://example.invalid\x00", "x")
	ids, err := r.ResolveCustomFieldIDs()
	require.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "creating field request")
}

// TestResolveCustomFieldIDs_ConcurrentCallsHitServerOnce asserts the
// mutex-protected cache coalesces concurrent callers so the JIRA
// endpoint is hit exactly once even under fan-out. Designed to run
// under -race.
func TestResolveCustomFieldIDs_ConcurrentCallsHitServerOnce(t *testing.T) {
	body, _ := json.Marshal([]Field{{ID: "customfield_1", Name: "Work Stream"}})
	var hits int64
	server := fieldResolverServer(t, body, http.StatusOK, &hits)
	defer server.Close()

	r := NewFieldResolver(server.URL, "x")

	const callers = 32
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			_, err := r.ResolveCustomFieldIDs()
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), atomic.LoadInt64(&hits))
}

func TestResolveCustomFieldIDs_SetsAuthAndContentTypeHeaders(t *testing.T) {
	var seenAuth, seenContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		seenContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	_, err := NewFieldResolver(server.URL, "Bearer secret").ResolveCustomFieldIDs()
	require.NoError(t, err)
	assert.Equal(t, "Bearer secret", seenAuth)
	assert.Equal(t, "application/json", seenContentType)
}
