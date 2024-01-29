#!/bin/bash

# Array to hold module paths
modules=()

# Find all directories containing a go.mod file
while IFS= read -r module; do
    modules+=("$module")
    echo "Testing and generating coverage for module: $module"
    (cd "$module" && go test -race -coverpkg=./... -coverprofile=coverage.out -timeout=30s ./...)
done < <(find . -type f -name 'go.mod' -exec dirname {} \;)

# Output the modules for the matrix
echo "modules=$(IFS=,; echo "${modules[*]}")" >> $GITHUB_OUTPUT
