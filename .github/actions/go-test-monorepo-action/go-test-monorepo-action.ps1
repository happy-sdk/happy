# Read script arguments
$ONLY_MODULE_LIST = $args[0]
$GO_TEST_RACE = $args[1]
$FAIL_FAST = $args[2]
$CONTINUE_ON_ERROR = $args[3]

# Array to hold module paths
$modules = @()
$testFailed = $false

# Get the repository root path
$repoRoot = Get-Location

# Find all directories containing a go.mod file
Get-ChildItem -File -Recurse -Filter "go.mod" | ForEach-Object {
  $module = $_.DirectoryName
  $modules += $module

  # Run tests only if ONLY_MODULE_LIST is not "true"
  if ($ONLY_MODULE_LIST -ne "true") {
    Write-Host "Testing and generating coverage for module: $module"
    Set-Location $module

    try {
      # Determine coverpkg value
      if ($module -eq $repoRoot.Path) {
        # Primary module: Cover only its own packages
        $coverpkg = (go list ./...) -join ","
      } else {
        # Submodules: Cover all their own packages
        $coverpkg = "./..."
      }

      # Run tests
      if ($GO_TEST_RACE -eq "true") {
        go test -race -coverpkg="$coverpkg" -coverprofile="coverage.out" -timeout=1m ./...
      } else {
        go test -coverpkg="$coverpkg" -coverprofile="coverage.out" -timeout=1m ./...
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
      Set-Location $repoRoot
    }
  }
}

# Convert modules array to JSON array format
$modules_json = $modules | ConvertTo-Json -Compress

# Output the modules for GitHub Actions or local use
Write-Host "modules=$modules_json"

if ($env:GITHUB_ACTIONS -eq "true") {
  Write-Host "::info::Monorepo modules: $modules_json"
}

# Handle exit status
if ($testFailed -and $CONTINUE_ON_ERROR -ne "true") {
  exit 1
} else {
  exit 0
}
