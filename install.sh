#!/bin/bash

# AssetCap Installation Script
# This script installs the latest version of assetcap for your platform

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="helmedeiros/digital-asset-capitalization"
BINARY_NAME="assetcap"
INSTALL_DIR="/usr/local/bin"

# Print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect platform
detect_platform() {
    local os=$(uname -s)
    local arch=$(uname -m)

    case $os in
        Linux*)
            case $arch in
                x86_64) echo "Linux_x86_64" ;;
                aarch64|arm64) echo "Linux_arm64" ;;
                *) print_error "Unsupported architecture: $arch"; exit 1 ;;
            esac
            ;;
        Darwin*)
            case $arch in
                x86_64) echo "Darwin_x86_64" ;;
                arm64) echo "Darwin_arm64" ;;
                *) print_error "Unsupported architecture: $arch"; exit 1 ;;
            esac
            ;;
        CYGWIN*|MINGW32*|MSYS*|MINGW*)
            case $arch in
                x86_64) echo "Windows_x86_64" ;;
                *) print_error "Unsupported architecture: $arch"; exit 1 ;;
            esac
            ;;
        *)
            print_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
}

# Get latest release version
get_latest_version() {
    curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install binary
install_binary() {
    local platform=$1
    local version=$2
    local temp_dir=$(mktemp -d)

    print_status "Downloading assetcap $version for $platform..."

    # Determine file extension
    local file_ext="tar.gz"
    if [[ $platform == *"Windows"* ]]; then
        file_ext="zip"
    fi

    local download_url="https://github.com/$REPO/releases/download/$version/${BINARY_NAME}_${platform}.${file_ext}"
    local download_file="$temp_dir/${BINARY_NAME}_${platform}.${file_ext}"

    if ! curl -sL "$download_url" -o "$download_file"; then
        print_error "Failed to download $download_url"
        exit 1
    fi

    print_status "Extracting binary..."
    cd "$temp_dir"

    if [[ $file_ext == "zip" ]]; then
        unzip -q "$download_file"
    else
        tar -xzf "$download_file"
    fi

    # Find the binary (it might be in a subdirectory)
    local binary_path=$(find . -name "$BINARY_NAME" -type f | head -n 1)
    if [[ -z "$binary_path" ]]; then
        print_error "Binary not found in archive"
        exit 1
    fi

    print_status "Installing binary to $INSTALL_DIR..."

    # Check if we need sudo
    if [[ ! -w "$INSTALL_DIR" ]]; then
        print_warning "Installing to $INSTALL_DIR requires sudo privileges"
        sudo mv "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
        sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        mv "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi

    # Cleanup
    rm -rf "$temp_dir"

    print_status "assetcap installed successfully!"
}

# Install dependencies
install_dependencies() {
    print_status "Installing dependencies..."

    # Check if Ollama is already installed
    if command -v ollama &> /dev/null; then
        print_status "Ollama is already installed"
        return
    fi

    print_status "Installing Ollama for AI features..."

    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if command -v brew &> /dev/null; then
            brew install ollama
            brew services start ollama
        else
            print_warning "Homebrew not found. Please install Ollama manually:"
            print_warning "https://ollama.com/download"
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        curl -fsSL https://ollama.com/install.sh | sh
        # Try to start the service
        if command -v systemctl &> /dev/null; then
            sudo systemctl start ollama || true
        fi
    else
        print_warning "Please install Ollama manually for your platform:"
        print_warning "https://ollama.com/download"
        return
    fi

    # Pull the required model
    print_status "Pulling LLaMA model..."
    if command -v ollama &> /dev/null; then
        ollama pull llama3 || print_warning "Failed to pull LLaMA model. You can run 'ollama pull llama3' later."
    fi
}

# Verify installation
verify_installation() {
    print_status "Verifying installation..."

    if command -v "$BINARY_NAME" &> /dev/null; then
        local version=$($BINARY_NAME --version 2>/dev/null || echo "unknown")
        print_status "assetcap installed successfully! Version: $version"
        print_status "Run 'assetcap --help' to get started"
    else
        print_error "Installation failed. Binary not found in PATH"
        exit 1
    fi
}

# Main installation process
main() {
    print_status "Starting AssetCap installation..."

    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        print_error "curl is required but not installed. Please install curl and try again."
        exit 1
    fi

    # Detect platform
    local platform=$(detect_platform)
    print_status "Detected platform: $platform"

    # Get latest version
    local version=$(get_latest_version)
    if [[ -z "$version" ]]; then
        print_error "Failed to get latest version from GitHub"
        exit 1
    fi
    print_status "Latest version: $version"

    # Install binary
    install_binary "$platform" "$version"

    # Install dependencies
    if [[ "${SKIP_DEPS:-}" != "true" ]]; then
        install_dependencies
    fi

    # Verify installation
    verify_installation

    print_status "Installation completed successfully!"
    echo
    print_status "Next steps:"
    echo "  1. Run 'assetcap config init' to set up your configuration"
    echo "  2. Run 'assetcap --help' to see available commands"
    echo "  3. Visit https://github.com/helmedeiros/digital-asset-capitalization for documentation"
}

# Handle command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-deps)
            export SKIP_DEPS=true
            shift
            ;;
        --help)
            echo "AssetCap Installation Script"
            echo
            echo "Usage: $0 [OPTIONS]"
            echo
            echo "Options:"
            echo "  --skip-deps    Skip dependency installation (Ollama)"
            echo "  --help         Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run main function
main
