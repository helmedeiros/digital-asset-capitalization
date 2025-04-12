#!/bin/bash

# Exit on error
set -e

echo "Installing dependencies for Digital Asset Capitalization Tool..."

# Check OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    echo "Detected macOS..."

    # Check if Homebrew is installed
    if ! command -v brew &> /dev/null; then
        echo "Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    fi

    # Install Ollama
    echo "Installing Ollama..."
    brew install ollama

    # Start Ollama service
    echo "Starting Ollama service..."
    brew services start ollama

    # Pull LLaMA model
    echo "Pulling LLaMA model..."
    ollama pull llama3

elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    echo "Detected Linux..."

    # Check if curl is installed
    if ! command -v curl &> /dev/null; then
        echo "Installing curl..."
        sudo apt-get update && sudo apt-get install -y curl
    fi

    # Install Ollama
    echo "Installing Ollama..."
    curl -fsSL https://ollama.com/install.sh | sh

    # Start Ollama service
    echo "Starting Ollama service..."
    sudo systemctl start ollama

    # Pull LLaMA model
    echo "Pulling LLaMA model..."
    ollama pull llama3
else
    echo "Unsupported operating system: $OSTYPE"
    exit 1
fi

# Verify Ollama installation
if ! command -v ollama &> /dev/null; then
    echo "Error: Failed to install Ollama"
    exit 1
fi

echo "Dependencies installed successfully!"
echo "To use the asset enrichment feature:"
echo "1. Make sure Ollama is running (default: http://localhost:11434)"
echo "2. The LLaMA model will be available for use"
echo "3. You can optionally set OLLAMA_API_URL environment variable"
