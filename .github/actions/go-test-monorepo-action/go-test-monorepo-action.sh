#!/bin/bash

# Check if only module list should be outputted
ONLY_MODULE_LIST="$1"
GO_TEST_RACE="$2"

# Array to hold module paths
modules=()

# Find all directories containing a go.mod file
while IFS= read -r module; do
  modules+=("$module")

  # Run tests only if ONLY_MODULE_LIST is not "true"
  if [ "$ONLY_MODULE_LIST" -ne "true" ]; then
    echo "Testing and generating coverage for module: $module"
    if [ "$GO_TEST_RACE" -eq "true" ]; then
      (cd "$module" && go test -race -coverpkg=./... -coverprofile=coverage.out -timeout=1m ./...)
    else
       (cd "$module" && go test -coverpkg=./... -coverprofile=coverage.out -timeout=1m ./...)
    fi
  fi

done < <(find . -type f -name 'go.mod' -exec dirname {} \;)

# Convert modules array to JSON array format
modules_json=$(printf '%s\n' "${modules[@]}" | jq -R . | jq -cs .)

# Check if GITHUB_OUTPUT is set, else use /dev/null
if [ -z "$GITHUB_OUTPUT" ]; then
  GITHUB_OUTPUT="/dev/null"
fi

# Output the modules for the matrix
echo "modules=$modules_json" >> $GITHUB_OUTPUT

# If running in GitHub Actions, set info message
if [ "$GITHUB_ACTIONS" == "true" ]; then
  echo "::info::Monorepo modules: $modules_json"
else
  echo "${modules_json}"
fi
