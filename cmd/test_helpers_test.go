package main

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
)

// newContextWithFlags builds a *cli.Context whose String/Bool lookups
// return the supplied values. Used to drive Action methods directly
// without the full cli.App boot. Shared across the command tests.
func newContextWithFlags(t *testing.T, strFlags map[string]string, boolFlags map[string]bool) *cli.Context {
	t.Helper()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for k, v := range strFlags {
		set.String(k, v, "")
	}
	for k, v := range boolFlags {
		set.Bool(k, v, "")
	}
	return cli.NewContext(nil, set, nil)
}
