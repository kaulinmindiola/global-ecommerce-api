# ============================================================================
# GO CODE FORMATTER - PowerShell Script for Windows
# ============================================================================
# Formats all Go code files in the project using gofmt and goimports

Write-Host "🎨 Formatting Go code..." -ForegroundColor Cyan

# Get all Go files (excluding vendor)
$goFiles = Get-ChildItem -Path . -Recurse -Include *.go -Exclude vendor | Where-Object { $_.FullName -notmatch '\\vendor\\' }

if ($goFiles.Count -eq 0) {
    Write-Host "ℹ️  No Go files found to format" -ForegroundColor Yellow
    exit 0
}

Write-Host "📝 Found $($goFiles.Count) Go files" -ForegroundColor Gray

# Format with gofmt
Write-Host "`n🔧 Running gofmt..." -ForegroundColor Cyan
foreach ($file in $goFiles) {
    gofmt -w $file.FullName
}

# Organize imports with goimports
Write-Host "`n📦 Running goimports..." -ForegroundColor Cyan
foreach ($file in $goFiles) {
    goimports -w $file.FullName
}

Write-Host "`n✅ Code formatting complete!" -ForegroundColor Green
Write-Host "📊 Formatted $($goFiles.Count) files" -ForegroundColor Gray
