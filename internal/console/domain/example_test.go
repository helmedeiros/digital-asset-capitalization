package domain_test

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// ExampleNewContext shows the empty starting state of a console
// session: no assets, no tasks, no project, no sprint.
func ExampleNewContext() {
	ctx := domain.NewContext("session-42")
	fmt.Println("session:", ctx.SessionID,
		"assets:", len(ctx.RecentAssets),
		"tasks:", len(ctx.RecentTasks))
	// Output: session: session-42 assets: 0 tasks: 0
}

// ExampleContext_UpdateTaskContext shows the recency list semantics:
// LRU-style head-insertion with deduplication and a cap of 5 entries.
func ExampleContext_UpdateTaskContext() {
	ctx := domain.NewContext("s")
	for _, key := range []string{"T-1", "T-2", "T-3", "T-1"} { // T-1 mentioned twice
		ctx.UpdateTaskContext(key, nil)
	}
	fmt.Println(ctx.RecentTasks)
	// Output: [T-1 T-3 T-2]
}

// ExampleContext_SetVariable demonstrates the typed-but-untyped scratch
// space the AI console uses for conversational follow-ups. Values
// returned from GetVariable carry their concrete type via the
// interface{} return.
func ExampleContext_SetVariable() {
	ctx := domain.NewContext("s")
	ctx.SetVariable("last_count", 42)
	if v, ok := ctx.GetVariable("last_count"); ok {
		fmt.Printf("%v (%T)\n", v, v)
	}
	// Output: 42 (int)
}
