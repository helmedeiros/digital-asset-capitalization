#!/bin/bash
set -e

# Minimum coverage the gate enforces. We sit at ~80.4% as of this
# writing; the threshold is set a few points below that so contributors
# have headroom for genuinely-untestable wiring without it slipping
# silently below 80%.
THRESHOLD=78

echo "Running coverage quality gate check..."
# Plain `go test` instead of gotestsum: CI uses go test here too, and
# the gate only needs the coverage profile — gotestsum's formatted
# output was being discarded with `> /dev/null 2>&1` either way. The
# -race flag turns this same run into a race-detector sweep so a
# parallel-test data race fails pre-push instead of fails CI.
go test -race -coverprofile=coverage.out ./... > /dev/null 2>&1
grep -v "testutil" coverage.out > coverage.filtered.out
COVERAGE=$(go tool cover -func=coverage.filtered.out | grep "total:" | awk '{print $3}' | sed 's/%//')
echo "Coverage: $COVERAGE%"
if [ "$(echo "$COVERAGE < $THRESHOLD" | bc -l)" -eq 1 ]; then
	echo "❌ Coverage $COVERAGE% is below $THRESHOLD% threshold"
	rm -f coverage.out coverage.filtered.out
	exit 1
fi
echo "✅ Coverage $COVERAGE% meets $THRESHOLD% threshold"
rm -f coverage.out coverage.filtered.out