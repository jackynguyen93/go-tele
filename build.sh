#!/bin/bash
set -e

# Build script for tdlib-go with proper library paths

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

echo "Building for $OS-$ARCH..."

# Set up library paths based on OS
if [[ "$OS" == "darwin" ]]; then
    # macOS - detect Homebrew installation
    if [[ "$ARCH" == "arm64" ]]; then
        # Apple Silicon
        BREW_PREFIX="/opt/homebrew"
    else
        # Intel Mac
        BREW_PREFIX="/usr/local"
    fi

    # Find OpenSSL
    if [ -d "$BREW_PREFIX/opt/openssl@3" ]; then
        OPENSSL_PATH="$BREW_PREFIX/opt/openssl@3"
    elif [ -d "$BREW_PREFIX/opt/openssl@1.1" ]; then
        OPENSSL_PATH="$BREW_PREFIX/opt/openssl@1.1"
    elif [ -d "$BREW_PREFIX/opt/openssl" ]; then
        OPENSSL_PATH="$BREW_PREFIX/opt/openssl"
    else
        echo "Error: OpenSSL not found. Install with: brew install openssl"
        exit 1
    fi

    echo "Using OpenSSL from: $OPENSSL_PATH"

    # Check for TDLib installation
    TDLIB_PATH=""
    if [ -d "/usr/local/tdlib-1.8.19" ]; then
        TDLIB_PATH="/usr/local/tdlib-1.8.19"
    elif pkg-config --exists tdjson; then
        TDLIB_PATH=$(pkg-config --variable=libdir tdjson | sed 's|/lib$||')
    fi

    if [ -n "$TDLIB_PATH" ]; then
        echo "Found TDLib at: $TDLIB_PATH"
    else
        echo "Error: TDLib not found. Install with: brew install tdlib"
        exit 1
    fi

    # Set CGO flags for macOS
    export CGO_CFLAGS="-I$BREW_PREFIX/include -I$OPENSSL_PATH/include -I$TDLIB_PATH/include"
    export CGO_LDFLAGS="-L$BREW_PREFIX/lib -L$OPENSSL_PATH/lib -L$TDLIB_PATH/lib"
    export PKG_CONFIG_PATH="$TDLIB_PATH/lib/pkgconfig:$OPENSSL_PATH/lib/pkgconfig:$PKG_CONFIG_PATH"
    export DYLD_LIBRARY_PATH="$TDLIB_PATH/lib:$DYLD_LIBRARY_PATH"

elif [[ "$OS" == "linux" ]]; then
    # Linux
    export CGO_CFLAGS="-I/usr/local/include -I/usr/include"
    export CGO_LDFLAGS="-L/usr/local/lib -L/usr/lib"
fi

# Enable CGO
export CGO_ENABLED=1

# Build
echo "Building tdclient..."
go build -v -tags libtdjson -o tdclient ./cmd/tdclient

echo "✓ Build successful!"
echo "Run with: ./tdclient -config config.yaml"
