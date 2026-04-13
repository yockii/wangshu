[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    
    [string]$Title = "",
    
    [switch]$Draft
)

$ErrorActionPreference = "Stop"

$VersionTag = if ($Version.StartsWith("v")) { $Version } else { "v$Version" }
$Today = Get-Date -Format "yyyy-MM-dd"

Write-Host "Preparing release $VersionTag ..." -ForegroundColor Cyan

git cliff --tag $VersionTag --unreleased --prepend CHANGELOG.md

if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to generate CHANGELOG. Please install git-cliff" -ForegroundColor Red
    Write-Host "Install: cargo install git-cliff or winget install git-cliff" -ForegroundColor Yellow
    exit 1
}

$content = Get-Content CHANGELOG.md -Raw
$content = $content -replace "## \[$VersionTag\]", "## [$VersionTag] - $Today"
Set-Content CHANGELOG.md $content -NoNewline

Write-Host "CHANGELOG.md updated" -ForegroundColor Green

git add CHANGELOG.md
git commit -m "Update CHANGELOG for $VersionTag"
git push

$ReleaseTitle = if ($Title) { $Title } else { $VersionTag }

Write-Host "Creating GitHub Release..." -ForegroundColor Cyan

if ($Draft) {
    gh release create $VersionTag --draft --generate-notes --title $ReleaseTitle
} else {
    gh release create $VersionTag --generate-notes --title $ReleaseTitle
}

Write-Host ""
Write-Host "Done!" -ForegroundColor Green
Write-Host "Now build and upload artifacts:" -ForegroundColor Yellow
Write-Host "  gh release upload $VersionTag ./bin/*" -ForegroundColor White
