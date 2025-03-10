#!/bin/sh

set -e  # Exit if any command fails

OUT_BIN="ani" 
INSTALL_DIR="/usr/local/bin"

# Build the Go project
echo "Building Go project..."
go build -o "$OUT_BIN" cmd/aniview/main.go

# Move the binary to the install directory (requires sudo)
echo "Moving executable to $INSTALL_DIR..."
sudo mv "$OUT_BIN" "$INSTALL_DIR/"

echo "Build complete. You can now run '$OUT_BIN' from anywhere."
