package main

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// stubAssetServiceForSyncAndEnrich is a concurrent-safe stub for the
// sync-and-enrich Action. The Action fans out GenerateKeywords and
// EnrichAsset calls via errgroup with a configurable concurrency
// limit, so the recording fields are guarded by a mutex.
type stubAssetServiceForSyncAndEnrich struct {
	unimplementedAssetService

	syncResult *assetdomain.SyncResult
	syncErr    error

	mu              sync.Mutex
	syncCalls       int
	syncSpace       string
	syncLabel       string
	syncDebug       bool
	keywordsCalled  []string
	keywordsErrFor  map[string]error
	enrichCalled    []enrichCall
	enrichErrForKey map[string]error
}

type enrichCall struct {
	name, field string
}

func (s *stubAssetServiceForSyncAndEnrich) SyncFromConfluence(space, label string, debug bool) (*assetdomain.SyncResult, error) {
	s.mu.Lock()
	s.syncCalls++
	s.syncSpace = space
	s.syncLabel = label
	s.syncDebug = debug
	s.mu.Unlock()
	return s.syncResult, s.syncErr
}

func (s *stubAssetServiceForSyncAndEnrich) GenerateKeywords(name string) error {
	s.mu.Lock()
	s.keywordsCalled = append(s.keywordsCalled, name)
	err := s.keywordsErrFor[name]
	s.mu.Unlock()
	return err
}

func (s *stubAssetServiceForSyncAndEnrich) EnrichAsset(name, field string) error {
	s.mu.Lock()
	s.enrichCalled = append(s.enrichCalled, enrichCall{name, field})
	err := s.enrichErrForKey[name+":"+field]
	s.mu.Unlock()
	return err
}

func TestApp_assetsSyncAndEnrichAction_LabelRequired(t *testing.T) {
	t.Parallel()
	a := &App{assetService: &stubAssetServiceForSyncAndEnrich{}}
	err := a.assetsSyncAndEnrichAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "label is required")
}

func TestApp_assetsSyncAndEnrichAction_DryRunSkipsSync(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &stubAssetServiceForSyncAndEnrich{}
	a := &App{assetService: stub}
	ctx := newContextWithFlags(t,
		map[string]string{"label": "cap-asset"},
		map[string]bool{"dry-run": true, "keywords": true})
	out, err := captureStdout(t, func() error { return a.assetsSyncAndEnrichAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Starting sync-and-enrich workflow")
	assert.Contains(t, out, "DRY RUN: Would sync assets and enrich with keywords=true")
	assert.Equal(t, 0, stub.syncCalls, "dry-run should not call SyncFromConfluence")
}

func TestApp_assetsSyncAndEnrichAction_SyncErrorWraps(t *testing.T) {
	t.Parallel()
	stub := &stubAssetServiceForSyncAndEnrich{syncErr: errors.New("confluence down")}
	a := &App{assetService: stub}
	ctx := newContextWithFlags(t, map[string]string{"label": "cap-asset"}, nil)
	err := a.assetsSyncAndEnrichAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync assets")
}

func TestApp_assetsSyncAndEnrichAction_MaxConcurrentClampedToOne(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &stubAssetServiceForSyncAndEnrich{syncResult: &assetdomain.SyncResult{}}
	a := &App{assetService: stub}
	// max-concurrent=0 should clamp to 1.
	ctx := newContextWithFlags(t,
		map[string]string{"label": "cap-asset"},
		nil)
	out, err := captureStdout(t, func() error { return a.assetsSyncAndEnrichAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "MaxConcurrent: 1")
	assert.Equal(t, 1, stub.syncCalls)
	assert.Equal(t, "cap-asset", stub.syncLabel)
}

func TestApp_assetsSyncAndEnrichAction_KeywordsBranchHitsBothSuccessAndWarning(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &stubAssetServiceForSyncAndEnrich{
		syncResult: &assetdomain.SyncResult{SyncedAssets: []*assetdomain.Asset{
			{Name: "Alpha"}, {Name: "Beta"},
		}},
		keywordsErrFor: map[string]error{"Beta": errors.New("llama down")},
	}
	a := &App{assetService: stub}
	ctx := newContextWithFlags(t,
		map[string]string{"label": "cap-asset"},
		map[string]bool{"keywords": true})
	out, err := captureStdout(t, func() error { return a.assetsSyncAndEnrichAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Step 2: Generating keywords")
	assert.Contains(t, out, "Generated keywords for: Alpha")
	assert.Contains(t, out, "Warning: Failed to generate keywords for Beta: llama down")
	sort.Strings(stub.keywordsCalled)
	assert.Equal(t, []string{"Alpha", "Beta"}, stub.keywordsCalled)
}

func TestApp_assetsSyncAndEnrichAction_FieldsBranchHitsBothSuccessAndWarning(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &stubAssetServiceForSyncAndEnrich{
		syncResult: &assetdomain.SyncResult{SyncedAssets: []*assetdomain.Asset{
			{Name: "Alpha"},
		}},
		enrichErrForKey: map[string]error{"Alpha:why": errors.New("llama timeout")},
	}
	a := &App{assetService: stub}
	// Two fields → enrich called twice for Alpha.
	ctx := newContextWithStringSlices(t,
		map[string]string{"label": "cap-asset"},
		nil,
		map[string][]string{"fields": {"description", "why"}})
	out, err := captureStdout(t, func() error { return a.assetsSyncAndEnrichAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Step 3: Enriching fields")
	assert.Contains(t, out, "Enriched description field for: Alpha")
	assert.Contains(t, out, "Warning: Failed to enrich why field for Alpha: llama timeout")
	require.Len(t, stub.enrichCalled, 2)
}

func TestApp_assetsSyncAndEnrichAction_NoSyncedAssetsSkipsBothEnrichmentSteps(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &stubAssetServiceForSyncAndEnrich{
		syncResult: &assetdomain.SyncResult{}, // empty SyncedAssets
	}
	a := &App{assetService: stub}
	ctx := newContextWithStringSlices(t,
		map[string]string{"label": "cap-asset"},
		map[string]bool{"keywords": true},
		map[string][]string{"fields": {"description"}})
	out, err := captureStdout(t, func() error { return a.assetsSyncAndEnrichAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Synced 0 assets")
	assert.NotContains(t, out, "Step 2: Generating keywords")
	assert.NotContains(t, out, "Step 3: Enriching fields")
	assert.True(t, strings.Contains(out, "Sync-and-enrich workflow completed successfully!"))
}
