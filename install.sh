#!/bin/bash

# TraceVibe Installation Script
# Automatically detects platform and downloads appropriate binary

set -e

REPO="peshwar9/tracevibe"
VERSION="latest"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="tracevibe"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and Architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin)
            PLATFORM="darwin"
            ;;
        mingw*|cygwin*|msys*)
            PLATFORM="windows"
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            echo "TraceVibe binaries are available for macOS and Windows only."
            echo "For other platforms, please build from source: https://github.com/$REPO"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac

    echo "$PLATFORM-$ARCH"
}

# Download binary from GitHub releases
download_binary() {
    local platform=$1
    local temp_file="/tmp/tracevibe-$$"

    echo -e "${YELLOW}Downloading TraceVibe for $platform...${NC}"

    # Get latest release URL
    if [ "$VERSION" = "latest" ]; then
        DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/tracevibe-$platform"
    else
        DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/tracevibe-$platform"
    fi

    # Add .exe extension for Windows
    if [[ "$platform" == *"windows"* ]]; then
        DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
        temp_file="${temp_file}.exe"
    fi

    # Download the binary
    if command -v curl &> /dev/null; then
        curl -fsSL "$DOWNLOAD_URL" -o "$temp_file" || {
            echo -e "${RED}Failed to download TraceVibe${NC}"
            echo "URL attempted: $DOWNLOAD_URL"
            exit 1
        }
    elif command -v wget &> /dev/null; then
        wget -q "$DOWNLOAD_URL" -O "$temp_file" || {
            echo -e "${RED}Failed to download TraceVibe${NC}"
            echo "URL attempted: $DOWNLOAD_URL"
            exit 1
        }
    else
        echo -e "${RED}Neither curl nor wget found. Please install one of them.${NC}"
        exit 1
    fi

    echo "$temp_file"
}

# Install binary to system
install_binary() {
    local temp_file=$1
    local install_path="$INSTALL_DIR/$BINARY_NAME"

    # Make binary executable
    chmod +x "$temp_file"

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$temp_file" "$install_path"
    else
        echo -e "${YELLOW}Installing to $INSTALL_DIR requires sudo access${NC}"
        sudo mv "$temp_file" "$install_path"
    fi

    echo -e "${GREEN}TraceVibe installed successfully to $install_path${NC}"
}

# Verify installation
verify_installation() {
    if command -v tracevibe &> /dev/null; then
        echo -e "${GREEN}✓ TraceVibe is in PATH${NC}"
        tracevibe --version
        return 0
    else
        echo -e "${YELLOW}⚠ TraceVibe installed but not in PATH${NC}"
        echo "Add $INSTALL_DIR to your PATH:"
        echo "  export PATH=\$PATH:$INSTALL_DIR"
        return 1
    fi
}

# Main installation flow
main() {
    echo "╔══════════════════════════════════════╗"
    echo "║     TraceVibe Installation Script    ║"
    echo "╚══════════════════════════════════════╝"
    echo

    # Check for custom install directory
    if [ -n "$1" ]; then
        INSTALL_DIR="$1"
        echo -e "${YELLOW}Using custom install directory: $INSTALL_DIR${NC}"
    fi

    # Detect platform
    PLATFORM=$(detect_platform)
    echo -e "${GREEN}Detected platform: $PLATFORM${NC}"

    # Download binary
    TEMP_FILE=$(download_binary "$PLATFORM")

    # Install binary
    install_binary "$TEMP_FILE"

    # Verify installation
    verify_installation

    echo
    echo "🚀 Get started with TraceVibe:"
    echo "  1. Generate guidelines: tracevibe guidelines -o rtm-guidelines.md"
    echo "  2. Import RTM data: tracevibe import <file> --project <name>"
    echo "  3. Start web UI: tracevibe serve"
    echo
    echo "📚 Documentation: https://github.com/$REPO"
}

# Run main function
main "$@"