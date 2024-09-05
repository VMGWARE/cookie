#!/bin/bash
set -e

# Run migrations
/app/discuit migrate run

# Build the UI
echo "Building the UI..."
cd /app/ui
bun -b run build:prod
cd ..

# Start the Discuit server
echo "Starting Discuit..."
exec "$@"
