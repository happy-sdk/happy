# Read script arguments
$ONLY_MODULE_LIST = $args[0]
$GO_TEST_RACE = $args[1]
$FAIL_FAST = $args[2]
$CONTINUE_ON_ERROR = $args[3]

# Array to hold module paths
$modules = @()
$testFailed = $false

# Find all directories containing a go.mod file
Get-ChildItem -File -Recurse -Filter "go.mod" | ForEach-Object {
  $module = $_.DirectoryName
  $modules += $module

  # Run tests only if ONLY_MODULE_LIST is not "true"
  if ($ONLY_MODULE_LIST -ne "true") {
    Write-Host "Testing and generating coverage for module: $module"
    Set-Location $module

    try {
      # Set correct coverpkg value
      if ($module -eq (Get-Location).Path) {
        # Primary module: Cover only its own packages
        $coverpkg = (go list ./...) -join ","
      } else {
        # Submodules: Cover all their own packages
        $coverpkg = "./..."
      }

      # Run tests with race condition check if enabled
      if ($GO_TEST_RACE -eq "true") {
        go test -race -coverpkg="$coverpkg" -coverprofile=coverage.out -timeout=1m ./...
      } else {
        go test -coverpkg="$coverpkg" -coverprofile=coverage.out -timeout=1m ./...
      }

      if ($LASTEXITCODE -ne 0) {
        $testFailed = $true
        if ($FAIL_FAST -eq "true") {
          throw "Test failed in $module"
        }
      }
    } catch {
      Write-Host "Error: $_"
      break
    } finally {
      Set-Location ..
    }
  }
}

# Convert modules array to JSON array format
$modules_json = $modules | ConvertTo-Json -Compress

# Output the modules for the matrix
Write-Host "modules=$modules_json"

# If running in GitHub Actions, set info message
if ($env:GITHUB_ACTIONS -eq "true") {
  Write-Host "::info::Monorepo modules: $modules_json"
}

# Exit with 1 if any test failed and continue-on-error is false
if ($testFailed -and $CONTINUE_ON_ERROR -ne "true") {
  exit 1
} else {
  exit 0
}
