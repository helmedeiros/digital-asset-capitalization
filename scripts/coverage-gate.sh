#!/bin/bash
set -e

echo "Running coverage quality gate check..."
gotestsum -- -coverprofile=coverage.out ./... > /dev/null 2>&1
grep -v "testutil" coverage.out > coverage.filtered.out
COVERAGE=$(go tool cover -func=coverage.filtered.out | grep "total:" | awk '{print $3}' | sed 's/%//')
echo "Coverage: $COVERAGE%"
if [ "$(echo "$COVERAGE < 70" | bc -l)" -eq 1 ]; then
	echo "❌ Coverage $COVERAGE% is below 70% threshold"
	rm -f coverage.out coverage.filtered.out
	exit 1
fi
echo "✅ Coverage $COVERAGE% meets 70% threshold"
rm -f coverage.out coverage.filtered.out