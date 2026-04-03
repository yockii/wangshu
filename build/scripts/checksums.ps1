$files = Get-ChildItem bin/*.exe, bin/wangshu-desktop-* -ErrorAction SilentlyContinue
if ($files) {
    $files | ForEach-Object {
        $hash = (Get-FileHash $_ -Algorithm SHA256).Hash.ToLower()
        "$hash  $($_.Name)"
    } | Out-File -Encoding utf8 bin/checksums.txt
    Write-Host "Checksums saved to bin/checksums.txt"
    Get-Content bin/checksums.txt
} else {
    Write-Host "No binaries found in bin/"
}
