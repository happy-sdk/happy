# Check if only module list should be outputted
$ONLY_MODULE_LIST = $args[0]

# Array to hold module paths
$modules = @()

# Find all directories containing a go.mod file
Get-ChildItem -File -Recurse -Filter "go.mod" | ForEach-Object {
    $module = $_.DirectoryName
    $modules += $module

    # Run tests only if ONLY_MODULE_LIST is not "true"
    if ($ONLY_MODULE_LIST -ne "true") {
        Write-Host "Testing and generating coverage for module: $module"
        Set-Location $module
        go test -race -coverpkg=./... -coverprofile=coverage.out -timeout=1m ./...
        Set-Location ..
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
