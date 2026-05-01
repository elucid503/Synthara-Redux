#!/bin/bash

# Build script for Synthara-Redux

echo "Building Synthara-Redux with optimizations..."
echo ""

# Set CGO flags for optimization and warning suppression

export CGO_CFLAGS="-O3 -Wno-unused-result -Wno-stringop-overread"
export CGO_CFLAGS="$CGO_CFLAGS -I$HOME/.local/include"
export CGO_LDFLAGS="-L$HOME/.local/lib -Wl,-rpath,$HOME/.local/lib"
export PKG_CONFIG_PATH="$HOME/.local/lib/pkgconfig:$PKG_CONFIG_PATH"

# CGO for explicit confirmation

export CGO_ENABLED=1

# Build the project

go build -v -o synthara-redux

# Check if build was successful

if [ $? -eq 0 ]; then
    echo ""
    echo "Build successful!"
    echo "Executable: synthara-redux"
else
    echo ""
    echo "Build failed"
    exit 1
fi
