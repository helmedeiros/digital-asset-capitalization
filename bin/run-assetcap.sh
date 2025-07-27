#!/bin/bash

# AssetCap Runner - Always builds latest binary before execution
# Usage: ./bin/run-assetcap.sh [assetcap arguments...]

set -e

echo "🔧 Building latest assetcap binary..."
go build -o assetcap cmd/main.go

echo "🚀 Running assetcap $@"
./assetcap "$@"
