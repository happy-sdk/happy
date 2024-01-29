#!/bin/bash

# Array to hold module paths
modules=()

# Find all directories containing a go.mod file
while IFS= read -r module; do
  modules+=("$module")
  echo "Testing and generating coverage for module: $module"
  (cd "$module" && go test -race -coverpkg=./... -coverprofile=coverage.out -timeout=1m ./...)
done < <(find . -type f -name 'go.mod' -exec dirname {} \;)

# Convert modules array to JSON array format
modules_json=$(printf '%s\n' "${modules[@]}" | jq -R . | jq -s .)

if [ -z "$GITHUB_OUTPUT" ]; then
  GITHUB_OUTPUT="/dev/null"
fi
# Output the modules for the matrix
echo "modules=$modules_json" >> $GITHUB_OUTPUT
echo "${modules_json}"
