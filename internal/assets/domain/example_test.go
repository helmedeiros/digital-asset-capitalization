package domain_test

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// ExampleNewAsset shows the canonical constructor. A fresh asset
// starts at version 1 with zero associated tasks.
func ExampleNewAsset() {
	asset, err := domain.NewAsset("Search", "Search subsystem ranking and indexing")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(asset.Name, "v"+fmt.Sprint(asset.GetVersion()), "tasks:", asset.GetTaskCount())
	// Output: Search v1 tasks: 0
}

// ExampleAsset_IncrementTaskCount demonstrates the version bump that
// rides along with every state-changing operation, keeping consumers
// honest about optimistic concurrency.
func ExampleAsset_IncrementTaskCount() {
	asset, _ := domain.NewAsset("Search", "x")
	_ = asset.IncrementTaskCount()
	_ = asset.IncrementTaskCount()
	fmt.Println("tasks:", asset.GetTaskCount(), "version:", asset.GetVersion())
	// Output: tasks: 2 version: 3
}

// ExampleAsset_AddContributingTeam shows the team set semantics: each
// team only appears once even if added repeatedly, mirroring the
// owning-team / contributing-teams shape rendered on the asset page.
func ExampleAsset_AddContributingTeam() {
	asset, _ := domain.NewAsset("Search", "x")
	_ = asset.SetOwningTeam("Voyager")
	_ = asset.AddContributingTeam("Catalog")
	_ = asset.AddContributingTeam("Catalog") // duplicate -- no-op
	_ = asset.AddContributingTeam("Indexer")
	fmt.Println("owner:", asset.GetOwningTeam(), "contributors:", asset.ContributingTeams)
	// Output: owner: Voyager contributors: [Catalog Indexer]
}
