# Check if only module list should be outputted
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

  # Run tests only if ONLY_MODULE_LIST is not "true" and not null or empty
  if (-not [string]::IsNullOrEmpty($ONLY_MODULE_LIST) -and $ONLY_MODULE_LIST -ne "true") {
    Write-Host "Testing and generating coverage for module: $module"
    Set-Location $module
    try {
      if ($GO_TEST_RACE -eq "true") {
        go test -race -coverpkg=./... -coverprofile=coverage.out -timeout=1m ./...
      } else {
        go test -coverpkg=./... -coverprofile=coverage.out -timeout=1m ./...
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
$modules_json = $modules | ConvertTo-Json

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
