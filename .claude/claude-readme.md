# AssetCap - Claude Assistant Context

## 🚀 Quick Start (for Claude)

```bash
# ALWAYS use these patterns:
make build && ./assetcap [commands]           # Recommended
./bin/run-assetcap.sh [commands]              # Auto-build wrapper
make build-run ARGS="assets teams list"      # Single command
```

## 🚨 NEVER Use

```bash
./main [commands]  # ❌ BLOCKED - outdated binary
```

## 💡 Why This Setup?

1. **Team management features** exist only in latest source code
2. **Building ensures** latest functionality
3. **Permissions block** old binary usage
4. **Wrapper script** automates build+run

## 🔧 Key Configurations Applied

- **`.claude/settings.local.json`**: Permissions updated to block `./main` and allow build commands
- **`Makefile`**: Added `build` and `build-run` targets
- **`bin/run-assetcap.sh`**: Auto-build wrapper script
- **`.claude/INSTRUCTIONS.md`**: Detailed usage guide

This ensures Claude always uses the latest binary with full team management capabilities! 🎯
