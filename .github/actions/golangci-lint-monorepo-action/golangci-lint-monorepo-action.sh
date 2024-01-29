#!/bin/bash

# Determine the golangci-lint executable path
GOLANGCI_LINT="golangci-lint"
if [ -n "$GITHUB_WORKSPACE" ]; then
  GOLANGCI_LINT="$GITHUB_WORKSPACE/bin/golangci-lint"
fi

# Find all directories containing a go.mod file
modules=$(find . -type f -name 'go.mod' -exec dirname {} \;)

# Loop through each module and run golangci-lint
for module in $modules; do
  echo "Linting module: $module"
  (cd "$module" && $GOLANGCI_LINT run ./...)
done
