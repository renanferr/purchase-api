# Setup script for test database (Windows PowerShell)
# Creates the purchase_api_test database for integration tests

param(
    [string]$DbHost = "localhost",
    [int]$DbPort = 5432,
    [string]$DbUser = "postgres",
    [string]$DbPassword = "postgres"
)

$testDbName = "purchase_api_test"

Write-Host "Setting up test database: $testDbName" -ForegroundColor Green
Write-Host "Database host: $DbHost`:$DbPort"
Write-Host ""

# Create connection string for postgres database
$connString = "Server=$DbHost;Port=$DbPort;User Id=$DbUser;Password=$DbPassword;Database=postgres;"

try {
    # Load PostgreSQL client
    $pgPath = "C:\Program Files\PostgreSQL\*\bin\psql.exe"
    $psqlExe = Get-Item $pgPath | Select-Object -Last 1 -ExpandProperty FullName
    
    if (-not (Test-Path $psqlExe)) {
        throw "psql.exe not found. Please ensure PostgreSQL is installed and in PATH."
    }

    # Create SQL script
    $sqlScript = @"
-- Drop existing test database if it exists
DROP DATABASE IF EXISTS "$testDbName";

-- Create fresh test database
CREATE DATABASE "$testDbName" 
  OWNER "$DbUser"
  ENCODING 'UTF8';
"@

    # Save script to temp file
    $tempFile = [System.IO.Path]::GetTempFileName() -replace '\.tmp$', '.sql'
    Set-Content -Path $tempFile -Value $sqlScript
    
    # Execute script
    Write-Host "Creating test database..."
    $env:PGPASSWORD = $DbPassword
    & $psqlExe -h $DbHost -p $DbPort -U $DbUser -d "postgres" -f $tempFile
    $env:PGPASSWORD = $null
    
    # Clean up temp file
    Remove-Item $tempFile -Force
    
    Write-Host ""
    Write-Host "Test database '$testDbName' created successfully" -ForegroundColor Green
    Write-Host ""
    Write-Host "You can now run integration tests with:" -ForegroundColor Cyan
    Write-Host "  go test ./tests/integration -v" -ForegroundColor White
    Write-Host ""
    Write-Host "Or with custom database URL:" -ForegroundColor Cyan
    Write-Host "  TEST_DATABASE_URL='postgres://user:pass@host:5432/$testDbName?sslmode=disable' go test ./tests/integration -v" -ForegroundColor White
    
} catch {
    Write-Host "Error setting up test database: $_" -ForegroundColor Red
    exit 1
}
