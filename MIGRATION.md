# Migrating TraceVibe to Its Own Repository

## Step 1: Create New Repository

1. Create new GitHub repository: `tracevibe` (or `tracevibe-cli`)
2. Initialize with README

## Step 2: Copy TraceVibe Code

```bash
# Clone the new repo
git clone https://github.com/yourusername/tracevibe.git
cd tracevibe

# Copy all TraceVibe files
cp -r /path/to/statsly/rtm-system/tracevibe/* .

# Update module path in go.mod
# Change: module github.com/peshwar9/statsly/tracevibe
# To: module github.com/yourusername/tracevibe
```

## Step 3: Update Import Paths

Update all Go import statements:

```go
// Old
import "github.com/peshwar9/statsly/tracevibe/internal/database"

// New
import "github.com/yourusername/tracevibe/internal/database"
```

Files to update:
- main.go
- cmd/*.go
- internal/importer/importer.go

## Step 4: Update Documentation

1. Update README.md with standalone focus
2. Update installation instructions
3. Remove references to Statsly-specific paths

## Step 5: Test the Standalone Tool

```bash
# Build and test
go mod tidy
go build -o tracevibe
./tracevibe --help

# Test with example data
./tracevibe guidelines --with-prompt
./tracevibe import ../statsly/rtm-system/rtm-data-improved.json --project test
./tracevibe serve
```

## Step 6: Set Up GitHub Releases

1. Create first release tag:
```bash
git tag v1.0.0
git push origin v1.0.0
```

2. Build binaries for different platforms:
```bash
# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o tracevibe-darwin-amd64

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o tracevibe-darwin-arm64

# Linux
GOOS=linux GOARCH=amd64 go build -o tracevibe-linux-amd64

# Windows
GOOS=windows GOARCH=amd64 go build -o tracevibe-windows-amd64.exe
```

3. Upload binaries to GitHub release

## Step 7: Update Installation Instructions

Update install.sh and INSTALL.md with new repository URL:

```bash
# Old
go install github.com/peshwar9/statsly/rtm-system/tracevibe@latest

# New
go install github.com/peshwar9/tracevibe@latest
```

## Step 8: Clean Up Statsly Repository

### Option A: Complete Removal
```bash
cd statsly
git rm -r rtm-system
git commit -m "Moved RTM system to standalone TraceVibe repository"
```

### Option B: Keep RTM Data Only
```bash
cd statsly
# Keep the RTM data file
mv rtm-system/rtm-data-improved.json .

# Create pointer README
echo "# RTM System" > RTM_README.md
echo "The RTM system has been moved to: https://github.com/yourusername/tracevibe" >> RTM_README.md
echo "" >> RTM_README.md
echo "Statsly's RTM data is in: rtm-data-improved.json" >> RTM_README.md

# Remove the rest
git rm -r rtm-system
git add rtm-data-improved.json RTM_README.md
git commit -m "Moved RTM system to TraceVibe, kept Statsly RTM data"
```

## What to Keep in Statsly

Consider keeping:
1. `rtm-data-improved.json` - Statsly's actual RTM data
2. A README pointing to TraceVibe
3. Example of how Statsly uses TraceVibe

## Benefits After Migration

1. **Independent versioning** - TraceVibe can have its own release cycle
2. **Cleaner installation** - `go install github.com/yourusername/tracevibe@latest`
3. **Focused documentation** - TraceVibe docs not mixed with Statsly
4. **Better discoverability** - Standalone tool is easier to find
5. **Community contributions** - Easier for others to contribute

## Post-Migration Checklist

- [ ] New repository created and working
- [ ] All tests passing in new location
- [ ] Documentation updated
- [ ] Installation instructions updated
- [ ] First release created
- [ ] Statsly repository cleaned up
- [ ] Any references in Statsly updated to point to new repo