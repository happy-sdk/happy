#!/bin/bash

ONLY_MODULE_LIST="$1"
GO_TEST_RACE="$2"
FAIL_FAST="$3"
CONTINUE_ON_ERROR="$4"

modules=()
test_failed=0

while IFS= read -r module; do
  modules+=("$module")

  if [ "$ONLY_MODULE_LIST" != "true" ]; then
    echo "Testing and generating coverage for module: $module"

    if [[ "$module" == "." ]]; then
      # Primary module: Cover only its own packages, not the entire monorepo
      coverpkg=$(go list ./... | tr '\n' ',')
    else
      # Submodules: Cover all their own packages
      coverpkg="./..."
    fi

    if [ "$GO_TEST_RACE" == "true" ]; then
      (cd "$module" && go test -race -coverpkg="$coverpkg" -coverprofile=coverage.out -timeout=1m ./...)
    else
      (cd "$module" && go test -coverpkg="$coverpkg" -coverprofile=coverage.out -timeout=1m ./...)
    fi
    test_exit_status=$?

    if [ $test_exit_status -ne 0 ]; then
      test_failed=1
      if [ "$FAIL_FAST" == "true" ]; then
        break
      fi
    fi
  fi

done < <(find . -type f -name 'go.mod' -exec dirname {} \;)

modules_json=$(printf '%s\n' "${modules[@]}" | jq -R . | jq -cs .)

# Output and GitHub Action handling
if [ "$GITHUB_ACTIONS" == "true" ]; then
  echo "modules=$modules_json" >> $GITHUB_OUTPUT
  echo "::info::Monorepo modules: $modules_json"
else
  echo "${modules_json}"
fi

# Exit handling
if [ $test_failed -ne 0 ]; then
  if [ "$CONTINUE_ON_ERROR" == "true" ]; then
    exit 0
  else
    exit 1
  fi
else
  exit 0
fi
