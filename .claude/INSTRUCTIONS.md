# Claude Assistant Instructions for AssetCap Project

## 🚨 CRITICAL: Always Use Latest Binary

**NEVER use `./main` - it's an outdated binary!**

## ✅ Approved Command Patterns

### Option 1: Build-then-run (Recommended)

```bash
# Always build first, then run
make build && ./assetcap [commands]

# Or use the convenient wrapper
./bin/run-assetcap.sh [commands]
```

### Option 2: Direct build-run

```bash
# Build and run in one command
make build-run ARGS="assets teams --help"
make build-run ARGS="assets show --name 'Asset Name'"
```

### Option 3: Install and use globally

```bash
# Install to GOPATH (less preferred for development)
make install
assetcap [commands]
```

## 🛠️ Development Workflow

1. **Always build first**: `make build`
2. **Run latest binary**: `./assetcap [commands]`
3. **For testing**: Use `./bin/run-assetcap.sh` wrapper

## 📋 Key Asset Commands

```bash
# Team Management
./assetcap assets teams assign --asset "Asset Name" --owner "TeamName"
./assetcap assets teams add-contributor --asset "Asset Name" --team "TeamName"
./assetcap assets teams show --asset "Asset Name"
./assetcap assets teams list

# Keywords & Content
./assetcap assets keywords --name "Asset Name"
./assetcap assets sync --space "MZN" --label "cap-asset"

# Basic Asset Operations
./assetcap assets create --name "Name" --description "Description"
./assetcap assets show --name "Asset Name"
./assetcap assets list
```

## 🔒 Why This Matters

- The source code has the latest team management features
- Old binaries lack critical functionality
- Building ensures you get all recent changes
- Prevents "command not found" errors

## ⚠️ Blocked Commands

- `./main` (explicitly denied in permissions)
- Any reference to old binary paths
