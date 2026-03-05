#!/bin/bash

# Build script for Synthara-Redux
# This script enables C compiler optimizations and suppresses false-positive warnings

echo "Building Synthara-Redux with optimizations..."
echo ""

# Set CGO flags for optimization and warning suppression

export CGO_CFLAGS="-O3 -Wno-unused-result"
export CGO_CFLAGS="$CGO_CFLAGS -I$HOME/.local/include"
export CGO_LDFLAGS="-L$HOME/.local/lib -Wl,-rpath,$HOME/.local/lib"
export PKG_CONFIG_PATH="$HOME/.local/lib/pkgconfig:$PKG_CONFIG_PATH"

# CGO for explicit confirmation

export CGO_ENABLED=1

# Ensure disgo is patched for segfault fix

if [ ! -d "../disgo" ]; then
    echo "Cloning disgo repository..."
    git clone https://github.com/disgoorg/disgo.git ../disgo
fi

cd ../disgo
git checkout v0.19.0-rc.15
git apply ../Synthara-Redux/disgo.patch
cd ../Synthara-Redux

# Build the project

go build -v -o synthara-redux

# Check if build was successful

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Build successful!"
    echo "Executable: synthara-redux"
else
    echo ""
    echo "✗ Build failed"
    exit 1
fi