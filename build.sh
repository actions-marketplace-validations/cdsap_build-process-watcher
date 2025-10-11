#!/bin/bash

set -xe  # Exit immediately on error, print commands as they run

# Clean and recreate dist directory
rm -rf dist
mkdir -p dist

# Build src/index_with_backend.ts -> dist/index_with_backend.js
echo "Building src/index_with_backend.ts..."
npx ncc build src/index_with_backend.ts -o dist-main || { echo "❌ Failed to build src/index_with_backend.ts"; exit 1; }
mv dist-main/index.js dist/index_with_backend.js || { echo "❌ dist-main/index.js not found"; exit 1; }
rm -rf dist-main

# Build src/cleanup.ts -> dist/cleanup.js
echo "Building src/cleanup.ts..."
npx ncc build src/cleanup.ts -o dist-tmp || { echo "❌ Failed to build src/cleanup.ts"; exit 1; }

mv dist-tmp/index.js dist/cleanup.js || { echo "❌ dist-tmp/index.js not found"; exit 1; }

rm -rf dist-tmp
