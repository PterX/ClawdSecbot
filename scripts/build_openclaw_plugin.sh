#!/bin/bash
set -e

PROJECT_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"
PLUGIN_DIR="$PROJECT_ROOT/plugins"
GO_LIB_DIR="$PROJECT_ROOT/go_lib"
OUTPUT_NAME="botsec"

# Detect OS and set file extension
echo "Building BotSec plugin..."

OS="$(uname -s)"
case "$OS" in
    Linux*)     EXT=".so" ;;
    Darwin*)    EXT=".dylib" ;;
    CYGWIN*|MINGW*|MSYS*) EXT=".dll" ;;
    *)          EXT=".so" ;;
esac

echo "Detected OS: $OS (extension: $EXT)"

mkdir -p "$PLUGIN_DIR"

cd "$GO_LIB_DIR"

# Build for current platform
# Note: Go automatically adds 'lib' prefix for c-shared builds
go build -buildvcs=false -buildmode=c-shared -o "${OUTPUT_NAME}${EXT}" .

# Check both possible output names (Go adds 'lib' prefix)
BUILT_FILE=""
if [ -f "lib${OUTPUT_NAME}${EXT}" ]; then
    BUILT_FILE="lib${OUTPUT_NAME}${EXT}"
elif [ -f "${OUTPUT_NAME}${EXT}" ]; then
    BUILT_FILE="${OUTPUT_NAME}${EXT}"
fi

if [ -n "$BUILT_FILE" ]; then
    echo "Build successful: $BUILT_FILE"
    
    # Clean old files in plugins directory (both openclaw and botsec)
    rm -f "$PLUGIN_DIR/openclaw${EXT}" "$PLUGIN_DIR/libopenclaw${EXT}"
    rm -f "$PLUGIN_DIR/openclaw.so" "$PLUGIN_DIR/openclaw.dylib"
    rm -f "$PLUGIN_DIR/${OUTPUT_NAME}${EXT}" "$PLUGIN_DIR/lib${OUTPUT_NAME}${EXT}"
    
    # Copy with the final name (without 'lib' prefix)
    cp "$BUILT_FILE" "$PLUGIN_DIR/${OUTPUT_NAME}${EXT}"
    
    # Copy header file if exists
    HEADER_FILE="${BUILT_FILE%.dylib}.h"
    HEADER_FILE="${HEADER_FILE%.so}.h"
    if [ -f "$HEADER_FILE" ]; then
        cp "$HEADER_FILE" "$PLUGIN_DIR/${OUTPUT_NAME}.h"
    fi
    
    echo "Copied to: $PLUGIN_DIR/${OUTPUT_NAME}${EXT}"
else
    echo "Build failed!"
    exit 1
fi
