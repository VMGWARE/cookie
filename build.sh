#!/bin/bash

# Exit on error
set -e

# Build the backend
go build

# Build the React app
cd ui
bun install --frozen-lockfile
bun -b run build:prod
cd ..
