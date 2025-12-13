#!/bin/bash

# Build script for Synthara-Redux with optimized opus compilation
# This script enables C compiler optimizations and suppresses false-positive warnings

echo "Building Synthara-Redux with optimized opus..."
echo ""

# Set CGO flags for optimization and warning suppression

export CGO_CFLAGS="-O3 -Wno-stringop-overread -Wno-unused-result"
export CGO_LDFLAGS=""

# Optional: Enable CGO for explicit confirmation

export CGO_ENABLED=1

# Build the project

echo "CGO_CFLAGS: $CGO_CFLAGS"
echo ""

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
