#!/bin/bash

# Exit on error
set -e

# Navigate to the project root (assuming script is in scripts/ or similar)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GO_LIB_DIR="$PROJECT_ROOT/go_lib"
PLUGINS_DIR="$PROJECT_ROOT/plugins"
OUTPUT_NAME="botsec"

echo "📍 Project Root: $PROJECT_ROOT"
echo "📂 Go Lib Dir: $GO_LIB_DIR"

# Detect Operating System
OS="$(uname -s)"
ARCH="$(uname -m)"

echo "🖥️  Detected System: $OS ($ARCH)"

EXT=""
case "${OS}" in
    Linux*)     EXT=".so";;
    Darwin*)    EXT=".dylib";;
    CYGWIN*|MINGW*|MSYS*) EXT=".dll";;
    *)          EXT=".so";;
esac

TARGET_FILE="$GO_LIB_DIR/${OUTPUT_NAME}${EXT}"

echo "🔨 Building Go Shared Library..."
cd "$GO_LIB_DIR"

# Clean previous build (both old openclaw and new botsec)
rm -f "openclaw.so" "openclaw.dylib" "openclaw.dll" "openclaw.h"
rm -f "${OUTPUT_NAME}.so" "${OUTPUT_NAME}.dylib" "${OUTPUT_NAME}.dll" "${OUTPUT_NAME}.h"

# Build command
# -buildmode=c-shared is required for FFI
# Note: Go automatically adds 'lib' prefix for c-shared builds
go build -buildvcs=false -o "${OUTPUT_NAME}${EXT}" -buildmode=c-shared .

# Check both possible output names (Go adds 'lib' prefix)
BUILT_FILE=""
if [ -f "lib${OUTPUT_NAME}${EXT}" ]; then
    BUILT_FILE="lib${OUTPUT_NAME}${EXT}"
elif [ -f "${OUTPUT_NAME}${EXT}" ]; then
    BUILT_FILE="${OUTPUT_NAME}${EXT}"
fi

if [ -n "$BUILT_FILE" ]; then
    echo "✅ Build Successful!"
    echo "📦 Output: $GO_LIB_DIR/$BUILT_FILE"
    
    # Copy to plugins directory
    echo "📋 Copying to plugins directory..."
    mkdir -p "$PLUGINS_DIR"
    
    # Clean old files in plugins directory (both openclaw and botsec)
    rm -f "$PLUGINS_DIR/openclaw${EXT}" "$PLUGINS_DIR/libopenclaw${EXT}"
    rm -f "$PLUGINS_DIR/${OUTPUT_NAME}${EXT}" "$PLUGINS_DIR/lib${OUTPUT_NAME}${EXT}"
    
    # Copy with the final name (without 'lib' prefix)
    cp "$BUILT_FILE" "$PLUGINS_DIR/${OUTPUT_NAME}${EXT}"
    
    # Copy header file if exists
    HEADER_FILE="${BUILT_FILE%.dylib}.h"
    if [ -f "$HEADER_FILE" ]; then
        cp "$HEADER_FILE" "$PLUGINS_DIR/${OUTPUT_NAME}.h"
    fi
    
    echo "✅ Copied to: $PLUGINS_DIR/${OUTPUT_NAME}${EXT}"
else
    echo "❌ Build Failed!"
    exit 1
fi

echo "🚀 You can now run the Flutter app."
