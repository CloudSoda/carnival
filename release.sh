#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Define the different values for GOOS and GOARCH
GOOS_VALUES=("linux" "darwin" "windows")
GOARCH_VALUES=("arm64" "amd64")

# Loop through all combinations of GOOS and GOARCH
for goos in "${GOOS_VALUES[@]}"; do
  for goarch in "${GOARCH_VALUES[@]}"; do
    echo "Building for GOOS=$goos GOARCH=$goarch"
    
    # Export the environment variables
    export GOOS=$goos
    export GOARCH=$goarch
    
    # Set binary name based on target OS
    if [ "$goos" = "windows" ]; then
      BINARY_NAME="carnival.exe"
      OUTPUT_FLAG="-o $BINARY_NAME"
    else
      BINARY_NAME="carnival"
      OUTPUT_FLAG="-o $BINARY_NAME"
    fi
    
    # Execute the build command directly in current directory
    echo "Building $BINARY_NAME for $goos/$goarch..."
    go build $OUTPUT_FLAG
    
    echo "✓ Build successful for $goos/$goarch"
    
    # Package the binary according to target OS
    if [ "$goos" = "windows" ]; then
      # Create zip file for Windows
      echo "Creating zip file for $goos/$goarch..."
      zip carnival-$goos-$goarch.zip $BINARY_NAME
      echo "✓ Package created: carnival-$goos-$goarch.zip"
    else
      # Create tar.gz for Unix-like systems
      echo "Creating tarball for $goos/$goarch..."
      tar -cf carnival-$goos-$goarch.tar $BINARY_NAME
      gzip carnival-$goos-$goarch.tar
      echo "✓ Package created: carnival-$goos-$goarch.tar.gz"
    fi
    
    echo "----------------------------------------"
  done
done

echo "All builds completed!"