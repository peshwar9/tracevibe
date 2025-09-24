#!/bin/bash

# Test script for TraceVibe release
set -e

echo "TraceVibe Release Testing"
echo "========================="
echo ""

# 1. Download the artifact (you'll need to do this manually from GitHub Actions)
echo "Step 1: Download artifact from GitHub Actions"
echo "  - Go to: https://github.com/peshwar9/tracevibe/actions"
echo "  - Find your workflow run"
echo "  - Download tracevibe-darwin-amd64.tar.gz"
echo ""

# 2. Extract and test
if [ -f "tracevibe-darwin-amd64.tar.gz" ]; then
    echo "Step 2: Extracting binary..."
    tar -xzf tracevibe-darwin-amd64.tar.gz

    echo "Step 3: Testing binary..."
    ./tracevibe-darwin-amd64 --version
    ./tracevibe-darwin-amd64 --help

    echo "✅ Binary works!"
else
    echo "Please download tracevibe-darwin-amd64.tar.gz first"
    exit 1
fi

# 3. Create GitHub release
echo ""
echo "Step 4: Create GitHub Release"
echo "==============================="
echo ""
echo "Option A: Using GitHub CLI (recommended):"
echo "gh release create v1.0.0 \\"
echo "  --title 'TraceVibe v1.0.0' \\"
echo "  --notes 'Initial release with macOS support' \\"
echo "  tracevibe-darwin-amd64.tar.gz \\"
echo "  tracevibe-darwin-amd64.tar.gz.sha256"
echo ""
echo "Option B: Manual via GitHub UI:"
echo "  1. Go to: https://github.com/peshwar9/tracevibe/releases/new"
echo "  2. Tag: v1.0.0"
echo "  3. Title: TraceVibe v1.0.0"
echo "  4. Upload the .tar.gz and .sha256 files"
echo "  5. Publish release"