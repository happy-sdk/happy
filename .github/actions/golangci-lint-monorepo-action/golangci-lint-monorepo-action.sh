#!/bin/bash

# Determine the golangci-lint executable path
GOLANGCI_LINT="golangci-lint"
if [ -n "$GITHUB_WORKSPACE" ]; then
  GOLANGCI_LINT="$GITHUB_WORKSPACE/bin/golangci-lint"
fi

# Find all directories containing a go.mod file
modules=$(find . -type f -name 'go.mod' -exec dirname {} \;)

# Initialize a variable to track linting errors
lint_errors=0

# Loop through each module and run golangci-lint
for module in $modules; do
  echo "Linting module: $module"
  (cd "$module" && $GOLANGCI_LINT run ./...) || lint_errors=1
done

# Exit with 1 if there were linting errors
if [ $lint_errors -ne 0 ]; then
  echo "Linting errors found"
  exit 1
fi

# Exit with 0 if everything was fine
exit 0
