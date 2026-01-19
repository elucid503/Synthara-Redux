#!/bin/bash

# Build script for Synthara-Redux
# This script enables C compiler optimizations and suppresses false-positive warnings

echo "Building Synthara-Redux with optimizations..."
echo ""

# Set CGO flags for optimization and warning suppression

export CGO_CFLAGS="-O3 -Wno-stringop-overread -Wno-unused-result"
export CGO_LDFLAGS=""

# CGO for explicit confirmation

export CGO_ENABLED=1

# Ensure disgo is patched for segfault fix

if [ ! -d "../disgo" ]; then
    echo "Cloning disgo repository..."
    git clone https://github.com/disgoorg/disgo.git ../disgo
fi

cd ../disgo
git checkout v0.19.0-rc.15
sed -i '/func (u \*udpConnImpl) Close() error {/,/}/c\
func (u *udpConnImpl) Close() error {\
	u.connMu.Lock()\
	defer u.connMu.Unlock()\
	if u.conn == nil {\
		return nil\
	}\
	return u.conn.Close()\
}' voice/udp_conn.go
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